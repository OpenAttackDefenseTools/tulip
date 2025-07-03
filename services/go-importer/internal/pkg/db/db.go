package db

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/netip"
	"runtime"
	"sync"
	"time"
	"math"

	"github.com/gammazero/workerpool"
	"github.com/gofrs/uuid/v5"

	pgxuuid "github.com/jackc/pgx-gofrs-uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Database struct {
	pool *pgxpool.Pool
	workerPool *workerpool.WorkerPool
	batcherFlowEntry *CopyBatcher
	batcherFlowItem *CopyBatcher
	batcherFlowIndex *CopyBatcher
	knownTags map[string]struct{}
	knownTagsMutex *sync.RWMutex
	fingerprints [][]int32
	fingerprintsMutex *sync.Mutex
	suricataIdWindow time.Duration
}

func NewDatabase(connectionString string) *Database {
	poolConfig, err := pgxpool.ParseConfig(connectionString)
	if err != nil {
		log.Fatalln("Unable to parse database config: ", err)
	}

	database := &Database {}
	poolConfig.AfterConnect = database.AfterConnect

	// Database connection pool
	database.pool, err = pgxpool.New(context.Background(), connectionString)
	if err != nil {
		log.Fatalln("Unable to create db connection pool: ", err)
	}

	for {
		err := database.pool.Ping(context.Background())
		if err == nil {
			break
		}

		log.Println("Unable to connect to database (retrying in 5s):", err)
		time.Sleep(5 * time.Second)
	}

	database.workerPool = workerpool.New(runtime.NumCPU() / 2)
	database.batcherFlowEntry = NewCopyBatcher(CopyBatcherConfig {
		db: database,
		tableName: pgx.Identifier{"flow"},
		columns: []string {
			"id", "port_src", "port_dst", "ip_src", "ip_dst", "duration", "tags",
			"flags", "flagids", "pcap_id", "link_child_id", "link_parent_id",
			"fingerprints", "packets_count", "packets_size", "flags_in", "flags_out",
		},
	})
	database.batcherFlowItem = NewCopyBatcher(CopyBatcherConfig {
		db: database,
		tableName: pgx.Identifier{"flow_item"},
		columns: []string{"id", "flow_id", "kind", "direction", "data"},
		batchSize: 2000,
	})
	database.batcherFlowIndex = NewCopyBatcher(CopyBatcherConfig {
		db: database,
		tableName: pgx.Identifier{"flow_index"},
		columns: []string{"flow_id", "text"},
		batchSize: 4000,
	})

	// Fingerprints
	// Periodically flush them into database
	database.fingerprintsMutex = &sync.Mutex{}
	go func() {
		for range time.Tick(5 * time.Second) {
			database.FingerprintsFlush()
		}
	}()

	// Known tags
	// Periodically update them in the background
	// in case someone else added some, or we did
	database.knownTagsMutex = &sync.RWMutex{}
	database.knownTags = make(map[string]struct{})
	database.KnownTagsUpdate()
	go func() {
		for range time.Tick(5 * time.Second) {
			database.KnownTagsUpdate()
		}
	}()

	// Suricata id search window
	if database.suricataIdWindow == 0 {
		database.suricataIdWindow = time.Duration(time.Minute)
	}

	return database
}

func (db *Database) AfterConnect(ctx context.Context, conn *pgx.Conn) error {
	pgxuuid.Register(conn.TypeMap())
	return nil
}

// Pcap files
type Pcap struct {
	Id uuid.UUID
	Name string
	Position int64
}

func (db *Database) PcapFindOrInsert(name string) Pcap {
	// With the amount of concurrency here we have to use ON CONFLICT,
	// any other solution (except maybe explicit locking) will cause
	// concurrency problems
	// UUIDs are used because sequence + ON CONFLICT increments evenry time this is run,
	// this is bad if the check is run a lot so we stick with random uuid
	// INDEX: Unique on pcap.name
	_, err := db.pool.Exec(context.Background(), `
		INSERT INTO pcap (id, name)
		VALUES (uuid_generate_v4(), @name)
		ON CONFLICT (name) DO NOTHING
	`, pgx.NamedArgs {
		"name": name,
	})

	if err != nil {
		log.Fatalln("Error inserting pcap: ", err)
	}

	// When DO NOTHING happens, no rows are returned
	// even with RETURNING so we kinda need a second query here
	// INDEX: Unique on pcap.name
	rows, _ := db.pool.Query(context.Background(), `
		SELECT *
		FROM pcap
		WHERE name = @name
	`, pgx.NamedArgs {
		"name": name,
	})
	defer rows.Close()

	pcap, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[Pcap])
	if err != nil {
		log.Fatalln("Error inserting pcap: ", err)
	}

	return pcap
}

func (db *Database) PcapSetPosition(id uuid.UUID, position int64) error {
	// INDEX: Primary on pcap.id
	_, err := db.pool.Exec(context.Background(), `
		UPDATE pcap
		SET position = @position
		WHERE id = @id
	`, pgx.NamedArgs {
		"id": id,
		"position": position,
	})

	if err != nil {
		log.Println("Error updating pcap position: ", err)
	}

	return err
}

// Known tags
// The Database struct has a list of all tags previously encountered
// Any new tags are asyncronusly inserted to the db and added to this list
func (db *Database) KnownTagsUpdate() {
	db.knownTagsMutex.Lock()
	defer db.knownTagsMutex.Unlock()

	// Insert any new tags
	// This will trigger ON CONFLICT if two assemblers
	// try to do this at the same time, which is fine
	if len(db.knownTags) != 0 {
		var knownTags []string
		for tag := range db.knownTags {
			knownTags = append(knownTags, tag)
		}

		// INDEX: Primary on tag.name
		_, err := db.pool.Exec(context.Background(), `
			INSERT INTO tag (name)
			SELECT jsonb_array_elements_text(@tags::jsonb - array_agg(name)) FROM tag
			ON CONFLICT (name) DO NOTHING
		`, pgx.NamedArgs {
			"tags": knownTags,
		})

		if err != nil {
			log.Fatalln("Error inserting tags: ", err)
		}
	}

	var tags []string
	err := db.pool.QueryRow(context.Background(), `
		SELECT array_agg(t.name)
		FROM (SELECT name FROM tag ORDER BY sort ASC) AS t
	`).Scan(&tags)

	if err != nil {
		log.Println("Error updating known tags: ", err)
		return
	}

	for _, tag := range tags {
		db.knownTags[tag] = struct{}{}
	}
}

func (db *Database) KnownTagExists(tag string) bool {
	db.knownTagsMutex.RLock()
	defer db.knownTagsMutex.RUnlock()
	_, ok := db.knownTags[tag]
	return ok
}

func (db *Database) KnownTagsUpsert(tag string) {
	if db.KnownTagExists(tag) {
		return
	}

	db.knownTagsMutex.Lock()
	defer db.knownTagsMutex.Unlock()

	// Tags are periodically upserted into database
	// see Database::KnownTagsUpdate
	db.knownTags[tag] = struct{}{}
}

// Flows
type FlowEntry struct {
	Id           uuid.UUID
	Src_port     uint16 `db:"port_src"`
	Dst_port     uint16 `db:"port_dst"`
	Src_ip       netip.Addr `db:"ip_src"`
	Dst_ip       netip.Addr `db:"ip_dst"`
	Time         time.Time
	Duration     time.Duration
	Filename     string `db:"-"`
	PcapId       uuid.UUID `db:"pcap_id"`
	Parent_id    *uuid.UUID `db:"link_parent_id"`
	Child_id     *uuid.UUID `db:"link_child_id"`
	Fingerprints []uint32
	Flow         []FlowItem `db:"-"`
	Tags         []string `db:"tags"`
	Flags        []string `db:"flags"`
	Flagids      []string `db:"flagids"`
	Num_packets  int `db:"packets_count"`
	Size         int `db:"packets_size"`
	Flags_In     int `db:"flags_in"`
	Flags_Out    int `db:"flags_out"`
}

type FlowItem struct {
	Id uuid.UUID
	FlowId uuid.UUID `db:"flow_id"`
	Kind string
	/// From: "s" / "c" for server or client
	From string `db:"direction"`
	/// The raw packet bytes
	Data []byte `msgpack:"-"`
	/// Timestamp of the first packet in the flow
	Time time.Time
}

// Flows are either coming from a file, in which case we'll dedupe them by pcap file name.
// If they're coming from a live capture, we can do pretty much the same thing, but we'll
// just have to come up with a label. (e.g. capture-<epoch>)
// We can always swap this out with something better, but this is how flower currently handles deduping.
//
// A single flow is defined by a db.FlowEntry" struct, containing an array of flowitems and some metadata
func (db *Database) FlowInsert(flow FlowEntry) {
	// Dont even try to insert empty flows
	if len(flow.Flow) == 0 {
		return
	}

	// Make sure tags exist
	// This can be done async
	for _, tag := range flow.Tags {
		tag := tag
		db.workerPool.Submit(func() {
			db.KnownTagsUpsert(tag)
		})
	}

	// Fallback to filename for pcap id
	pcap_id := flow.PcapId
	if pcap_id == uuid.Nil {
		pcap_id = db.PcapFindOrInsert(flow.Filename).Id
	}

	// Generate flow id
	flow_id := FidCreate(flow.Time)

	// Prepare index rows
	// These are split to chunks of maximum 1024 chars
	// This is to ensure length of records is not too different
	// between rows and to avoid rechecking large chunks of data
	// in memory after a lossy index search has been used
	chunkLength := 1024
	chunkOverlap := 64
	indexes := make([][]any, 0)
	for _, item := range flow.Flow {
		text := []rune(string(bytes.Replace(bytes.ToValidUTF8(item.Data, []byte{}), []byte{0}, []byte{}, -1)))
		chunkCount := int(math.Ceil(float64(len(text)) / float64(chunkLength)))

		// Each split between index rows has a 64 char overlap
		// This is to accomodate searches hitting the boundary
		for i := 0; i < chunkCount; i++ {
			startIndex := i * chunkLength
			endIndex := i * chunkLength + chunkLength + chunkOverlap
			if endIndex >= len(text) {
				endIndex = len(text)
			}

			chunk := string(text[startIndex:endIndex])
			indexes = append(indexes, []any { flow_id, chunk })
		}
	}

	// Insert index rows
	// This is async, since the index is not required to be peresent when we insert the flow
	// At worst it will take a few seconds before this flow is searchable
	db.batcherFlowIndex.PushAllCallback(indexes, func(errors <-chan error) {
		// Error inserting flow indexes
		if len(errors) != 0 {
			log.Println("Error inserting flow indexes (flow will not be fully searchable): ", <-errors)
		}
	})

	// Prepare flow items
	items := make([][]any, len(flow.Flow))
	for i := range flow.Flow {
		items[i] = []any {
			FidCreate(flow.Flow[i].Time),
			flow_id,
			flow.Flow[i].Kind,
			flow.Flow[i].From,
			&flow.Flow[i].Data,
		}
	}

	// Insert the flow items first, so that when flow is inserted, it is complete
	db.batcherFlowItem.PushAllCallback(items, func(errors <-chan error) {
		// Error inserting flow items
		// Only continue if we managed to insert at least one flow
		// If we got here with and empty flow, I guess just insert it
		if len(errors) != 0 && len(errors) == len(items) {
			// Just print the first error, they will all be the same probably
			log.Println("Error inserting flow items (flow will not be inserted): ", <-errors)
			return
		}

		// Fingerprints are uint32, but psql only has signed integer types
		// So we make them into int32, instead of using a larger psql int
		fingerprints := make([]int32, len(flow.Fingerprints))
		for i, fingerprint := range flow.Fingerprints {
			fingerprints[i] = int32(fingerprint)
		}

		// Push fingerprints for async flow connecting
		db.FingerprintsPush(fingerprints)

		// Now insert the flow
		db.batcherFlowEntry.PushCallback([]any {
			flow_id,
			flow.Src_port, flow.Dst_port,
			flow.Src_ip, flow.Dst_ip,
			// Postgres keeps duration with 1 microsecond precision
			// If we round down we risk some flow items being ouside of this duration
			flow.Duration.Truncate(time.Microsecond) + time.Microsecond,
			flow.Tags,
			flow.Flags,
			flow.Flagids,
			pcap_id,
			flow.Child_id,
			flow.Parent_id,
			fingerprints,
			flow.Num_packets,
			flow.Size,
			flow.Flags_In,
			flow.Flags_Out,
		}, func(err error) {
			if err != nil {
				log.Println("Error inserting flow: ", err)
			}
		})
	})
}

func (db *Database) FlowAddSignatures(flow_id uuid.UUID, signatures []Signature) {
	db.workerPool.Submit(func() {
		signaturesJson, _ := json.Marshal(signatures)

		// INDEX: Primary on flow.id
		_, err := db.pool.Exec(context.Background(), `
			UPDATE flow
			SET signatures = jsonb_unique(signatures || @signatures)
			WHERE id = @flow_id
		`, pgx.NamedArgs {
			"flow_id": flow_id,
			"signatures": signaturesJson,
		})

		if err != nil {
			log.Printf("Error inserting signatures for flow %s: %s\n", flow_id, err)
		}
	})
}

func (db *Database) FlowAddTags(flow_id uuid.UUID, tags []string) {
	// Make sure tags exist
	// This can (and will) be done async
	for _, tag := range tags {
		tag := tag
		db.workerPool.Submit(func() {
			db.KnownTagsUpsert(tag)
		})
	}

	db.workerPool.Submit(func() {
		tagsJson, _ := json.Marshal(tags)

		// INDEX: Primary on flow.id
		_, err := db.pool.Exec(context.Background(), `
			UPDATE flow
			SET tags = jsonb_unique(tags || @tags)
			WHERE id = @flow_id
		`, pgx.NamedArgs {
			"flow_id": flow_id,
			"tags": tagsJson,
		})

		if err != nil {
			log.Printf("Error inserting tags for flow %s: %s\n", flow_id, err)
		}
	})
}

// Suricata flows
type SuricataId struct {
	Src_port int
	Dst_port int
	Src_ip   netip.Addr
	Dst_ip   netip.Addr
	Time     time.Time
}

func (db *Database) SuricataIdFindFlow(id SuricataId) (uuid.UUID, error) {
	var flow_id uuid.UUID

	// INDEX: `f.id` is parition key, limiting scope to specific chunks
	// INDEX: `time_start` and `time_end` should be computed before passing as parameters
	// INDEX: Btree on (id, port_src, port_dst, ip_src, ip_dst)
	// INDEX: Make sure to run EXPLAIN ANALYZE when changing this
	err := db.pool.QueryRow(context.Background(), `
		SELECT id
		FROM flow
		WHERE port_src = @port_src
			AND port_dst = @port_dst
			AND ip_src = @ip_src
			AND ip_dst = @ip_dst
			AND id > fid_pack_low(@time_start)
			AND id < fid_pack_high(@time_end)
	`, pgx.NamedArgs {
		"time_start": id.Time.Add(-db.suricataIdWindow),
		"time_end": id.Time.Add(db.suricataIdWindow),
		"port_src": id.Src_port,
		"port_dst": id.Dst_port,
		"ip_src": id.Src_ip,
		"ip_dst": id.Dst_ip,
	}).Scan(&flow_id)

	return flow_id, err
}

// Suricata signature
type Signature struct {
	Id      int32 `json:"id"`
	Message string `json:"message"`
	Action  string `json:"action"`
}

// Fingerprints
func (db *Database) FingerprintsPush(fingerprints []int32) {
	if len(fingerprints) == 0 {
		return
	}

	db.fingerprintsMutex.Lock()
	defer db.fingerprintsMutex.Unlock()
	db.fingerprints = append(db.fingerprints, fingerprints)
}

func (db *Database) FingerprintsFlush() {
	db.fingerprintsMutex.Lock()

	if len(db.fingerprints) == 0 {
		db.fingerprintsMutex.Unlock()
		return
	}

	fingerprintsMap := make(map[int32]struct{})
	for _, ff := range db.fingerprints {
		if len(ff) > 1 {
			for _, f := range ff {
				fingerprintsMap[f] = struct{}{}
			}
		}
	}

	var fingerprintsUnique []int32
	for f := range fingerprintsMap {
		fingerprintsUnique = append(fingerprintsUnique, f)
	}

	fingerprintsJson, _ := json.Marshal(db.fingerprints)
	db.fingerprints = nil
	db.fingerprintsMutex.Unlock()

	// INDEX: Primary on fingerprint.id
	_, err := db.pool.Exec(context.Background(), `
		INSERT INTO fingerprint (id, grp)
		SELECT jsonb_array_elements(v.value)::int, coalesce(f.grp, v.value[0]::int)
			FROM jsonb_array_elements(@fingerprints) AS v
			LEFT JOIN fingerprint AS f
				ON f.id = ANY(ARRAY(SELECT value::int FROM jsonb_array_elements(v.value)))
		ON CONFLICT (id) DO NOTHING
	`, pgx.NamedArgs {
		"fingerprints": fingerprintsJson,
	})

	if err != nil {
		log.Println("Error inserting fingerprints: ", err)
	}

	// INDEX: Primary on fingerprint.id
	// INDEX: Btree on fingerprint.grp
	// INDEX: Gin on flow.fingerprints
	cmd, err := db.pool.Exec(context.Background(), `
		UPDATE flow AS ff
			SET link_parent_id = d.parent,
			link_child_id = d.child
		FROM (
			SELECT f.id, lag(f.id) OVER (w) AS parent, lead(f.id) OVER (w) AS child
			FROM flow AS f
			INNER JOIN fingerprint AS fp
				ON fp.grp = (SELECT grp FROM fingerprint AS fpp WHERE fpp.id = f.fingerprints[1])
			WHERE fp.id = ANY(@fingerprints)
			GROUP BY f.id, fp.grp
			WINDOW w AS (PARTITION BY fp.grp ORDER BY f.id)
			ORDER BY fp.grp, f.id
		) AS d
		WHERE d.id = ff.id
			AND (
				d.child != link_child_id
				OR d.parent != link_parent_id
				OR (d.child IS NULL) != (link_child_id IS NULL)
				OR (d.parent IS NULL) != (link_parent_id IS NULL)
			)
	`, pgx.NamedArgs {
		"fingerprints": fingerprintsUnique,
	})

	if err != nil {
		log.Println("Error linking flows: ", err)
	}

	if cmd.RowsAffected() != 0 {
		log.Printf("Linked %d flows\n", cmd.RowsAffected())
	}
}

// Flag ids
type FlagId struct {
	Id int32
	Content string
	Time time.Time
}

// Query all valid flag ids
func (db *Database) FlagIdsQuery(lifetime int) ([]FlagId, error) {
	rows, _ := db.pool.Query(context.Background(), `
		SELECT *
		FROM flag_id
		WHERE time > @time_limit
	`, pgx.NamedArgs {
		"time_limit": time.Now().Add(-time.Duration(float64(lifetime) * float64(time.Second))),
	});
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[FlagId])
}
