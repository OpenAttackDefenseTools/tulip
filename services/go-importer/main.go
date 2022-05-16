package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/google/gopacket"
	"github.com/google/gopacket/examples/util"
	"github.com/google/gopacket/ip4defrag"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/reassembly"
)

var decoder = ""
var lazy = false
var nodefrag = false
var checksum = false
var nohttp = true

var snaplen = 65536
var tstype = ""
var promisc = true

var watch_dir = flag.String("dir", "", "Directory to watch for new pcaps")
var mongodb = flag.String("mongo", "mongo:27017", "MongoDB dns name + port (e.g. mongo:27017)")

var db database

// TODO; FIXME; RDJ; this is kinda gross, but this is PoC level code
func reassemblyCallback(entry flowEntry) {
	db.InsertFlow(entry)
}

func main() {
	defer util.Run()()

	if flag.NArg() < 1 && *watch_dir == "" {
		log.Fatal("Usage: ./go-importer <file0.pcap> ... <fileN.pcap>")
	}

	db_string := "mongodb://" + *mongodb
	db = ConnectMongo(db_string)

	// Pass positional arguments to the pcap handler
	handlePcaps(flag.Args())

	// If a watch dir was configured, handle all files in the directory, then
	// keep monitoring it for new files.
	if *watch_dir != "" {
		watchDir()
	}
}

func watchDir() {

	stat, err := os.Stat(*watch_dir)
	if err != nil {
		log.Fatal("Failed to open the watch_dir with error: ", err)
	}

	if !stat.IsDir() {
		log.Fatal("watch_dir is not a directory")
	}

	log.Println("Monitoring dir: ", *watch_dir)

	files, err := ioutil.ReadDir(*watch_dir)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".pcap") {
			handlePcap(filepath.Join(*watch_dir, file.Name())) //FIXME; this is a little clunky
		}
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	defer watcher.Close()

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
				if event.Op&(fsnotify.Rename|fsnotify.Create) != 0 {
					if strings.HasSuffix(event.Name, ".pcap") {
						log.Println("Found new file", event.Name, event.Op.String())
						time.Sleep(2 * time.Second) // FIXME; bit of race here between file creation and writes.
						handlePcap(event.Name)
					}
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

func handlePcaps(file_list []string) {
	for _, uri := range file_list {
		handlePcap(uri)
	}
}

func handlePcap(fname string) {
	var handle *pcap.Handle
	var err error

	if handle, err = pcap.OpenOffline(fname); err != nil {
		log.Println("PCAP OpenOffline error:", err)
		return
	}

	if db.InsertPcap(fname) {
		log.Println("Processing file:", fname)
	} else {
		log.Println("Skipped: ", fname)
		return
	}

	var dec gopacket.Decoder
	var ok bool
	decoder_name := fmt.Sprintf("%s", handle.LinkType())
	if dec, ok = gopacket.DecodersByLayerName[decoder_name]; !ok {
		log.Println("No decoder named", decoder_name)
		return
	}
	source := gopacket.NewPacketSource(handle, dec)
	source.Lazy = lazy
	source.NoCopy = true
	count := 0
	bytes := int64(0)
	defragger := ip4defrag.NewIPv4Defragmenter()

	streamFactory := &tcpStreamFactory{source: fname, reassemblyCallback: reassemblyCallback}
	streamPool := reassembly.NewStreamPool(streamFactory)
	assembler := reassembly.NewAssembler(streamPool)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	for packet := range source.Packets() {
		count++
		data := packet.Data()
		bytes += int64(len(data))
		done := false

		// defrag the IPv4 packet if required
		if !nodefrag {
			ip4Layer := packet.Layer(layers.LayerTypeIPv4)
			if ip4Layer == nil {
				continue
			}
			ip4 := ip4Layer.(*layers.IPv4)
			l := ip4.Length
			newip4, err := defragger.DefragIPv4(ip4)
			if err != nil {
				log.Fatalln("Error while de-fragmenting", err)
			} else if newip4 == nil {
				continue // packet fragment, we don't have whole packet yet.
			}
			if newip4.Length != l {
				pb, ok := packet.(gopacket.PacketBuilder)
				if !ok {
					panic("Not a PacketBuilder")
				}
				nextDecoder := newip4.NextLayerType()
				nextDecoder.Decode(newip4.Payload, pb)
			}
		}

		tcp := packet.Layer(layers.LayerTypeTCP)
		if tcp != nil {
			tcp := tcp.(*layers.TCP)
			if checksum {
				err := tcp.SetNetworkLayerForChecksum(packet.NetworkLayer())
				if err != nil {
					log.Fatalf("Failed to set network layer for checksum: %s\n", err)
				}
			}
			c := Context{
				CaptureInfo: packet.Metadata().CaptureInfo,
			}
			assembler.AssembleWithContext(packet.NetworkLayer().NetworkFlow(), tcp, &c)
		}

		select {
		case <-signalChan:
			fmt.Fprintf(os.Stderr, "\nCaught SIGINT: aborting\n")
			done = true
		default:
			// NOP: continue
		}
		if done {
			break
		}
	}

	// This flushes connections that are still lingering, for example because
	// the never sent a FIN. This case is _super_ common in ctf captures
	assembler.FlushAll()
	streamFactory.WaitGoRoutines()
}
