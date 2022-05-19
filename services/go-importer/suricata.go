package main

import (
	"bufio"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/tidwall/gjson"
)

func watchEve(eve_file string) {
	stat, err := os.Stat(eve_file)
	if err != nil {
		log.Fatal("Failed to open the watch_dir with error: ", err)
	}

	if stat.IsDir() {
		log.Fatal("eve file is not a file")
	}

	log.Println("Monitoring eve file: ", eve_file)

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
	updateEve(eve_handle)

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
				if event.Op&(fsnotify.Write) != 0 {
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

	err = watcher.Add(*watch_dir)
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
	flow      flowID
	signature string
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
	signature := gjson.Get(json, "alert.signature")

	// TODO; Double check this, might be broken for non-UTC?
	start_time_obj, _ := time.Parse("2006-01-02T15:04:05.999999999-0700", start_time.String())

	logItem := suricataLog{
		flow: flowID{
			src_port: int(src_port.Int()),
			src_ip:   src_ip.String(),
			dst_port: int(dst_port.Int()),
			dst_ip:   dst_ip.String(),
			time:     start_time_obj,
		},
		signature: signature.String(),
	}

	db.AddSignatureToFlow(logItem)
	return true
}
