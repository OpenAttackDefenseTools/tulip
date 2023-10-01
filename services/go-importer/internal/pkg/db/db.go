package db

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/netip"
	"runtime"
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
	channelFlowEntry CopyChannelPool
	channelFlowItem CopyChannelPool
}

func NewDatabase(connectionString string) Database {
	poolConfig, err := pgxpool.ParseConfig(connectionString)
	if err != nil {
		log.Fatalln("Unable to parse database config: ", err)
	}

	database := Database {}
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
	database.channelFlowEntry = NewCopyChannelPool(CopyChannelContext {
		db: &database,
		table_name: pgx.Identifier{"flow"},
		columns: []string{
			"id", "port_src", "port_dst", "ip_src", "ip_dst", "duration", "tags",
			"blocked", "pcap_id", "connected_child_id", "connected_parent_id",
			"fingerprints", "packets_count", "packets_size", "flags_in", "flags_out",
		},
	})
	database.channelFlowItem = NewCopyChannelPool(CopyChannelContext {
		db: &database,
		table_name: pgx.Identifier{"flow_item"},
		columns: []string{"id", "flow_id", "kind", "direction", "data", "text"},
	})

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
	Parent_id    *uuid.UUID `db:"connected_parent_id"`
	Child_id     *uuid.UUID `db:"connected_child_id"`
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

	// Try finding a child flow by this flows fingerprints
	if len(flow.Fingerprints) > 0 {
		// Find a child flow. There should be at most one flow with
		// empty parent_id and matching fingerprint, otherwise, take the newest one
		// INDEX: `f.id` is parition key, limiting scope to specific chunks
		// INDEX: `time_start` and `time_end` should be computed before passing as parameters
		// INDEX: Make sure to run EXPLAIN ANALYZE when changing this
		var id uuid.UUID
		err := db.pool.QueryRow(context.Background(), `
			SELECT f.id
			FROM flow AS f
			WHERE f.connected_parent_id IS NULL
				AND f.fingerprints ?| @fingerprints
				AND f.id > uuid_pack_low(@time_start)
				AND f.id < uuid_pack_high(@time_end)
			ORDER BY f.id DESC
		`, pgx.NamedArgs {
			"fingerprints": flow.Fingerprints,
			"time_start": flow.Time.Add(-WINDOW_FLOW_CONNECT),
			"time_end": flow.Time.Add(WINDOW_FLOW_CONNECT),
		}).Scan(&id)

		// Found a child id
		if err == nil {
			// TODO Maybe add the childs fingerprints to mine?
			flow.Child_id = &id
		}
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
	db.channelFlowItem.PushAllCallback(items, func(errors <-chan error) {
		// Error inserting flow items
		// Only continue if we managed to insert at least one flow
		// If we got here with and empty flow, I guess just insert it
		if len(errors) > 0 && len(errors) == len(items) {
			// Just print the first error, they will all be the same probably
			log.Println("Error inserting flow items (flow will not be inserted): ", <-errors)
			return
		}

		// Now insert the flow
		db.channelFlowEntry.PushCallback([]any {
			flow_id,
			flow.Src_port, flow.Dst_port,
			flow.Src_ip, flow.Dst_ip,
			flow.Duration,
			flow.Tags,
			flow.Blocked,
			pcap_id,
			flow.Child_id,
			flow.Parent_id,
			flow.Fingerprints,
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

	// Flow had a child_id, lets find the child
	// and set its parent_id to this flow
	if flow.Child_id != nil {
		// INDEX: Primary on flow.id
		_, err := db.pool.Exec(context.Background(), `
			UPDATE flow
			SET connected_parent_id = @parent_id
			WHERE id = @child_id
		`, pgx.NamedArgs {
			"parent_id": flow.Id,
			"child_id": flow.Child_id,
		})

		// This is not a fatal error, just print it as a warning
		if err != nil {
			log.Printf("Error setting parent_id (%d) for flow (%d): %s\n", flow.Id, flow.Child_id, err)
		}
	}
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
