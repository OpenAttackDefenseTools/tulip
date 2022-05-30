package main

import (
	"go-importer/internal/pkg/db"

	"bufio"
	"flag"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/tidwall/gjson"
)

var eve_file = flag.String("eve", "", "Eve file to watch for suricata's tags")
var mongodb = flag.String("mongo", "", "MongoDB dns name + port (e.g. mongo:27017)")

var g_db db.Database

const WINDOW = 500 // ms

func main() {
	flag.Parse()
	if *eve_file == "" {
		log.Fatal("Usage: ./enricher -eve eve.json")
	}

	// If no mongo DB was supplied, try the env variable
	if *mongodb == "" {
		*mongodb = os.Getenv("TULIP_MONGO")
		// if that didn't work, just guess a reasonable default
		if *mongodb == "" {
			*mongodb = "mongo:27017"
		}
	}

	db_string := "mongodb://" + *mongodb
	g_db = db.ConnectMongo(db_string)

	watchEve(*eve_file)
}

func watchEve(eve_file string) {
	stat, err := os.Stat(eve_file)
	if err != nil {
		log.Fatal("Failed to open the eve file with error: ", err)
	}

	if stat.IsDir() {
		log.Fatal("eve file is not a file")
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// Open a handle to the eve file
	eve_handle, err := os.Open(eve_file)
	if err != nil {
		log.Fatal(err)
	}

	// Do the initial scan
	log.Println("Parsing initial eve contents...")
	updateEve(eve_handle)

	log.Println("Monitoring eve file: ", eve_file)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	// Keep running until Interrupt
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write != 0 {
					log.Println("Eve file was updated", event.Name, event.Op.String())
					updateEve(eve_handle)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("watcher error:", err)
			}
		}
	}()

	err = watcher.Add(eve_file)
	if err != nil {
		log.Fatal(err)
	}
	<-signalChan
	log.Println("Watcher stopped")

}

// The eve file was just written to, let's parse some logs!
func updateEve(eve_handle *os.File) {

	// BUG: I'm re-reading the entire file for now, we can probably pin it the last successfully synced log?
	_, _ = eve_handle.Seek(0, 0)
	scanner := bufio.NewScanner(eve_handle)

	// iterate over each line in the file
	for scanner.Scan() {
		line := scanner.Text()
		// Line parsing failed. Probably incomplete?
		if !handleEveLine(line) {
			break
		}
	}
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
	"proto": "TCP",
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
type suricataLog struct {
	flow      db.FlowID
	signature db.Signature
}

func handleEveLine(json string) bool {
	if !gjson.Valid(json) {
		return false
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

	// TODO; Double check this, might be broken for non-UTC?
	start_time_obj, _ := time.Parse("2006-01-02T15:04:05.999999999-0700", start_time.String())

	id := db.FlowID{
		Src_port: int(src_port.Int()),
		Src_ip:   src_ip.String(),
		Dst_port: int(dst_port.Int()),
		Dst_ip:   dst_ip.String(),
		Time:     start_time_obj,
	}

	sig := db.Signature{
		ID:     int(sig_id.Int()),
		Msg:    sig_msg.String(),
		Action: sig_action.String(),
	}

	g_db.AddSignatureToFlow(id, sig, WINDOW)
	return true
}
