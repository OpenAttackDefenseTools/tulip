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

	"github.com/gammazero/workerpool"
	"github.com/gofrs/uuid/v5"

	pgxuuid "github.com/jackc/pgx-gofrs-uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const WINDOW_FLOW_CONNECT = time.Duration(time.Hour)
const WINDOW_SURICATA_FIND = time.Duration(5 * time.Second)

type Database struct {
	pool *pgxpool.Pool
	workerPool *workerpool.WorkerPool
	batcherFlowEntry *CopyBatcher
	batcherFlowItem *CopyBatcher
	knownTags map[string]struct{}
	knownTagsMutex *sync.RWMutex
	fingerprints [][]int32
	fingerprintsMutex *sync.Mutex
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

		log.Println("Unable to connect to database (retrying in 5s): ", err)
		time.Sleep(5 * time.Second)
	}

	database.workerPool = workerpool.New(runtime.NumCPU() / 2)
	database.batcherFlowEntry = NewCopyBatcher(CopyBatcherConfig {
		db: database,
		tableName: pgx.Identifier{"flow"},
		columns: []string {
			"id", "port_src", "port_dst", "ip_src", "ip_dst", "duration", "tags",
			"blocked", "pcap_id", "link_child_id", "link_parent_id",
			"fingerprints", "packets_count", "packets_size", "flags_in", "flags_out",
		},
	})
	database.batcherFlowItem = NewCopyBatcher(CopyBatcherConfig {
		db: database,
		tableName: pgx.Identifier{"flow_item"},
		columns: []string{"id", "flow_id", "kind", "direction", "data", "text"},
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
		for range time.Tick(30 * time.Second) {
			database.KnownTagsUpdate()
		}
	}()

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
	Blocked      bool
	Filename     string `db:"-"`
	PcapId       uuid.UUID `db:"pcap_id"`
	Parent_id    *uuid.UUID `db:"link_parent_id"`
	Child_id     *uuid.UUID `db:"link_child_id"`
	Fingerprints []uint32
	Flow         []FlowItem `db:"-"`
	Tags         []string `db:"tags"`
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
	Data []byte
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
		go db.KnownTagsUpsert(tag)
	}

	// Fallback to filename for pcap id
	pcap_id := flow.PcapId
	if pcap_id == uuid.Nil {
		pcap_id = db.PcapFindOrInsert(flow.Filename).Id
	}

	// Generate flow id
	flow_id := UuidCreate(flow.Time)

	// Prepare flow items
	items := make([][]any, len(flow.Flow))
	for i := range flow.Flow {
		items[i] = []any {
			UuidCreate(flow.Flow[i].Time),
			flow_id,
			flow.Flow[i].Kind,
			flow.Flow[i].From,
			&flow.Flow[i].Data,
			string(bytes.Replace(bytes.ToValidUTF8(flow.Flow[i].Data, []byte{}), []byte{0}, []byte{}, -1)),
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
			flow.Duration,
			flow.Tags,
			flow.Blocked,
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
	tags := []string{ "suricata" }

	for _, signature := range signatures {
		if signature.Action == "blocked" {
			tags = append(tags, "blocked")
			break
		}
	}

	signaturesJson, _ := json.Marshal(signatures)
	tagsJson, _ := json.Marshal(tags)

	go db.pool.Exec(context.Background(), `
		UPDATE flow
		SET signatures = jsonb_unique(signatures || @signatures),
			tags = jsonb_unique(tags || @tags),
			blocked = blocked OR @tags ? 'blocked'
		WHERE id = @flow_id
	`, pgx.NamedArgs {
		"flow_id": flow_id,
		"signatures": signaturesJson,
		"tags": tagsJson,
	})
}

func (db *Database) FlowAddTags(flow_id uuid.UUID, tags []string) {
	tagsJson, _ := json.Marshal(tags)

	// Make sure tags exist
	// This can be done async
	for _, tag := range tags {
		go db.KnownTagsUpsert(tag)
	}

	go db.pool.Exec(context.Background(), `
		UPDATE flow
		SET tags = jsonb_unique(tags || @tags)
			blocked = blocked OR @tags ? 'blocked'
		WHERE id = @flow_id
	`, pgx.NamedArgs {
		"flow_id": flow_id,
		"tags": tagsJson,
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
			AND f.id > uuid_pack_low(@time_start)
			AND f.id < uuid_pack_high(@time_end)
	`, pgx.NamedArgs {
		"time_start": time.Now().Add(-WINDOW_SURICATA_FIND),
		"time_end": time.Now().Add(WINDOW_SURICATA_FIND),
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
