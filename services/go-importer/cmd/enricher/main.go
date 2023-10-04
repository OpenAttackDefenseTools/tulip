package main

import (
	"go-importer/internal/pkg/db"
	"net/netip"

	"bufio"
	"errors"
	"flag"
	"log"
	"os"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/tidwall/gjson"
)

var eve_file = flag.String("eve", "", "Eve file to watch for suricata's tags")
var timescale = flag.String("timescale", "", "Timescale connection string (e. g. postgres://usr:pwd@host:5432/tulip)")
var tag_flowbits = flag.Bool("flowbits", true, "Tag flows with their flowbits")
var rescan_period = flag.Int("t", 30, "rescan period (in seconds).")

var g_db *db.Database

func main() {
	flag.Parse()
	if *eve_file == "" {
		log.Fatal("Usage: ./enricher -eve eve.json")
	}

	// If no timescale connection string was supplied, use env variable
	if *timescale == "" {
		*timescale = os.Getenv("TIMESCALE")
	}

	log.Println("Connecting to Timescale:", *timescale, "...")
	g_db = db.NewDatabase(*timescale)

	watchEve(*eve_file)
}

func watchEve(eve_file string) {
	// Do the initial scan
	log.Println("Parsing initial eve contents...")
	ratchet := updateEve(eve_file, 0)

	log.Println("Monitoring eve file: ", eve_file)
	stat, err := os.Stat(eve_file)
	prevSize := int64(0)
	if err == nil {
		prevSize = stat.Size()
	}

	for {
		time.Sleep(time.Duration(*rescan_period) * time.Second)

		new_stat, err := os.Stat(eve_file)
		if err != nil {
			log.Println("Failed to open the eve file with error: ", err)
			continue
		}

		if new_stat.Size() > prevSize {
			log.Println("Eve file was updated. New size:, ", new_stat.Size())
			ratchet = updateEve(eve_file, ratchet)
		}
		prevSize = new_stat.Size()

	}

}

// The eve file was just written to, let's parse some logs!
func updateEve(eve_file string, ratchet int64) int64 {

	// Open a handle to the eve file
	eve_handle, err := os.Open(eve_file)
	if err != nil {
		log.Println("Failed to open the eve file")
		return ratchet
	}
	eve_handle.Seek(ratchet, 0)
	eve_reader := bufio.NewReader(eve_handle)
	defer eve_handle.Close()

	log.Println("Start scanning eve file @ ", ratchet)

	// iterate over each line in the file
	for {
		line, err := eve_reader.ReadString('\n')
		if err != nil {
			// This is most likely to be EOF, which is fine
			// TODO; check the error code and log if it is something else
			break
		}
		// Line parsing failed. Probably incomplete?
		applied, err := handleEveLine(line)
		if err == nil {
			// parsing this line failed. It may be incomplete for a couple reasons.
			// * First, we might have caught the file in the middle of a write.
			//   That's fine, next pass we'll get new data
			// * The second case is worse, this line may just be corrupt. In that case,
			//   we need to skip over it, after verifying that we're not in case 1.

			// For now, I'm just gonna solve this by ratchetting to the last successfully
			// applied rule. This will cause us to rescan a few lines needlessly, but I'm okay with that.
			if applied {
				ratchet += int64(len(line))
			}
		}
	}

	// Roll the eve handle back to the last successfully applied rule, so it can continue there
	// next time this function is called.
	return ratchet
}

/*
	{
		"timestamp": "2022-05-17T19:39:57.283547+0000",
		"flow_id": 1905964640824789,
		"in_iface": "eth0",
		"event_type": "alert",
		"src_ip": "131.155.9.104",
		"src_port": 53604,
		"dest_ip": "165.232.89.44",
		"dest_port": 1337,
		"protobufs": "TCP",
		"pkt_src": "stream (flow timeout)",
		"alert": {
			"action": "allowed",
			"gid": 1,
			"signature_id": 1338,
			"rev": 1,
			"signature": "Detected too many A's (smart)",
			"category": "",
			"severity": 3
		},
		"app_proto": "failed",
		"flow": {
			"pkts_toserver": 6,
			"pkts_toclient": 6,
			"bytes_toserver": 437,
			"bytes_toclient": 477,
			"start": "2022-05-17T19:37:02.978389+0000"
		}
	}
*/

func handleEveLine(json string) (bool, error) {
	if !gjson.Valid(json) {
		return false, errors.New("Invalid json in eve line")
	}

	// TODO; error check this
	src_port := gjson.Get(json, "src_port")
	src_ip := gjson.Get(json, "src_ip")
	dst_port := gjson.Get(json, "dest_port")
	dst_ip := gjson.Get(json, "dest_ip")
	start_time := gjson.Get(json, "flow.start")

	sig_msg := gjson.Get(json, "alert.signature")
	sig_id := gjson.Get(json, "alert.signature_id")
	sig_action := gjson.Get(json, "alert.action")
	flowbits := gjson.Get(json, "metadata.flowbits")

	// canonicalize the IP address notation to make sure it matches what the assembler entered
	// into the database.
	// TODO; just assuming these are all valid for now. Should be fine, since this is coming from
	// suricata and is not _really_ user controlled. Might panic in some obscure case though.
	ip_src, _ := netip.ParseAddr(src_ip.String())
	ip_dst, _ := netip.ParseAddr(dst_ip.String())

	// TODO; Double check this, might be broken for non-UTC?
	start_time_obj, _ := time.Parse("2006-01-02T15:04:05.999999999-0700", start_time.String())

	// If no action was taken, there's no need for us to do anything with this line.
	if !(sig_action.Exists() || (flowbits.Exists() && *tag_flowbits)) {
		return false, nil
	}

	flow_id, _ := g_db.SuricataIdFindFlow(db.SuricataId {
		Src_port: int(src_port.Int()),
		Src_ip:   ip_src,
		Dst_port: int(dst_port.Int()),
		Dst_ip:   ip_dst,
		Time:     start_time_obj,
	})

	if flow_id == uuid.Nil {
		flow_id, _ = g_db.SuricataIdFindFlow(db.SuricataId {
			Dst_port: int(src_port.Int()),
			Dst_ip:   ip_src,
			Src_port: int(dst_port.Int()),
			Src_ip:   ip_dst,
			Time:     start_time_obj,
		})
	}

	if flow_id == uuid.Nil {
		return false, nil
	}

	ret := false
	if sig_action.Exists() {

		sig := db.Signature{
			Id:      int32(sig_id.Int()),
			Message: sig_msg.String(),
			Action:  sig_action.String(),
		}

		g_db.FlowAddSignatures(flow_id, []db.Signature{sig})
		ret = true
	}

	if !(flowbits.Exists() && *tag_flowbits) {
		return ret, nil
	}

	tags := []string{}
	flowbits.ForEach(func(key, value gjson.Result) bool {
		tags = append(tags, value.String())
		return true // keep iterating
	})

	g_db.FlowAddTags(flow_id, tags)
	return true, nil
}
