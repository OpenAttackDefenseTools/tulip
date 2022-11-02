package main

import (
	"go-importer/internal/pkg/db"

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
var checksum = false
var nohttp = true

var snaplen = 65536
var tstype = ""
var promisc = true

var watch_dir = flag.String("dir", "", "Directory to watch for new pcaps")
var mongodb = flag.String("mongo", "", "MongoDB dns name + port (e.g. mongo:27017)")
var flag_regex = flag.String("flag", "", "flag regex, used for flag in/out tagging")
var nonstrict = flag.Bool("nonstrict", false, "Do not check strict TCP / FSM flags")
var experimental = flag.Bool("experimental", false, "Enable experimental features.")

var g_db db.Database

// TODO; FIXME; RDJ; this is kinda gross, but this is PoC level code
func reassemblyCallback(entry db.FlowEntry) {
	// Parsing HTTP will decode encodings to a plaintext format
	ParseHttpFlow(&entry)
	// Apply flag in / flagout
	if *flag_regex != "" {
		ApplyFlagTags(&entry, flag_regex)
	}
	// Finally, insert the new entry
	g_db.InsertFlow(entry)
}

func main() {
	defer util.Run()()

	flag.Parse()
	if flag.NArg() < 1 && *watch_dir == "" {
		log.Fatal("Usage: ./go-importer <file0.pcap> ... <fileN.pcap>")
	}

	// If no mongo DB was supplied, try the env variable
	if *mongodb == "" {
		*mongodb = os.Getenv("TULIP_MONGO")
		// if that didn't work, just guess a reasonable default
		if *mongodb == "" {
			*mongodb = "localhost:27017"
		}
	}

	// If no flag regex was supplied via cli, check the env
	if *flag_regex == "" {
		*flag_regex = os.Getenv("FLAG_REGEX")
		// if that didn't work, warn the user and continue
		if *flag_regex == "" {
			log.Print("WARNING; no flag regex found. No flag-in or flag-out tags will be applied.")
		}
	}

	db_string := "mongodb://" + *mongodb
	g_db = db.ConnectMongo(db_string)
	g_db.ConfigureDatabase()

	// Pass positional arguments to the pcap handler
	handlePcaps(flag.Args())

	// If a watch dir was configured, handle all files in the directory, then
	// keep monitoring it for new files.
	if *watch_dir != "" {
		watchDir(*watch_dir)
	}
}

func watchDir(watch_dir string) {

	stat, err := os.Stat(watch_dir)
	if err != nil {
		log.Fatal("Failed to open the watch_dir with error: ", err)
	}

	if !stat.IsDir() {
		log.Fatal("watch_dir is not a directory")
	}

	log.Println("Monitoring dir: ", watch_dir)

	files, err := ioutil.ReadDir(watch_dir)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".pcap") {
			handlePcap(filepath.Join(watch_dir, file.Name())) //FIXME; this is a little clunky
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

	err = watcher.Add(watch_dir)
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
	defer handle.Close()

	if g_db.ContainsPcap(fname) {
		log.Println("Skipped: ", fname)
		return
	}

	var source *gopacket.PacketSource
	nodefrag := false
	linktype := handle.LinkType()
	switch linktype {
	case layers.LinkTypeIPv4:
		source = gopacket.NewPacketSource(handle, layers.LayerTypeIPv4)
		break
	default:
		source = gopacket.NewPacketSource(handle, linktype)
	}

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
		// (TODO; IPv6 will not be defragged)
		ip4Layer := packet.Layer(layers.LayerTypeIPv4)
		if !nodefrag && ip4Layer != nil {
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

		transport := packet.TransportLayer()
		if transport == nil {
			continue
		}

		switch transport.LayerType() {
		case layers.LayerTypeTCP:
			tcp := transport.(*layers.TCP)
			c := Context{
				CaptureInfo: packet.Metadata().CaptureInfo,
			}
			assembler.AssembleWithContext(packet.NetworkLayer().NetworkFlow(), tcp, &c)
			break
		default:
			// pass
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

	log.Println("Processed file:", fname)
	g_db.InsertPcap(fname)
}
