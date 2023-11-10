package main

import (
	"go-importer/internal/pkg/db"

	"flag"
	"fmt"
	"net"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
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
var pcap_over_ip = flag.String("pcap-over-ip", "", "PCAP-over-IP host + port (e.g. remote:1337)")
var bpf = flag.String("bpf", "", "BPF filter")
var nonstrict = flag.Bool("nonstrict", false, "Do not check strict TCP / FSM flags")
var experimental = flag.Bool("experimental", false, "Enable experimental features.")
var flushAfter = flag.String("flush-after", "30s", `(TCP) Connections which have buffered packets (they've gotten packets out of order and
are waiting for old packets to fill the gaps) can be flushed after they're this old
(their oldest gap is skipped). This is particularly useful for pcap-over-ip captures.
Any string parsed by time.ParseDuration is acceptable here (ie. "3m", "2h45m").
This setting defaults to "30s" unless specified. To prevent connection flooding,
it is not recommended setting this to a high value, since assembler persists between pcaps.
Setting this to empty value disables TCP flushing.`)
var flushAfterUdp = flag.String("flush-after-udp", "30s", `Same as flush-after, except for UDP connections.
UDP connections are assembled by unique pairings of ip addressed and ports on both sides.
The only way a UDP connection is considered closed, is if this timeout passes without seeing any new packets.
This setting defaults to "30s" unless specified. To prevent connection flooding,
it is not recommended setting this to a high value, since assembler persists between pcaps.
Setting this to empty value disables UDP flushing.`)
var flushInterval = flag.String("flush-interval", "15s", `Period of flushing while processing one pcap.
Any string parsed by time.ParseDuration is acceptable here (ie. "3m", "2h45m").
Flushing always happens between pcaps, but sometimes (for example with PCAP-over-IP) it is required to flush periodically
while processing one file (since PCAP-over-IP treats whole connection as one pcap file). This is also the period for debug prints.`)

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

type AssemblerService struct {
	Defragmenter *ip4defrag.IPv4Defragmenter
	StreamFactory *TcpStreamFactory
	StreamPool *reassembly.StreamPool
	AssemblerTcp *reassembly.Assembler
	AssemblerUdp *UdpAssembler
	ConnectionTcpTimeout time.Duration
	ConnectionUdpTimeout time.Duration
	FlushInterval time.Duration
	BpfFilter string
}

func NewAssemblerService() AssemblerService {
	streamFactory := &TcpStreamFactory { reassemblyCallback: reassemblyCallback }
	streamPool := reassembly.NewStreamPool(streamFactory)
	assemblerUdp := NewUdpAssembler()

	return AssemblerService {
		Defragmenter: ip4defrag.NewIPv4Defragmenter(),
		StreamFactory: streamFactory,
		StreamPool: streamPool,
		AssemblerTcp: reassembly.NewAssembler(streamPool),
		AssemblerUdp: &assemblerUdp,
	}
}

func (service AssemblerService) FlushConnections() {
	thresholdTcp := time.Now().Add(-service.ConnectionTcpTimeout)
	thresholdUdp := time.Now().Add(-service.ConnectionUdpTimeout)
	flushed, closed, discarded := 0, 0, 0

	if service.ConnectionTcpTimeout != 0 {
		flushed, closed = service.AssemblerTcp.FlushCloseOlderThan(thresholdTcp);
		discarded = service.Defragmenter.DiscardOlderThan(thresholdTcp);
	}

	if flushed != 0 || closed != 0 || discarded != 0 {
		log.Println("Flushed", flushed, "closed", closed, "and discarded", discarded, "connections")
	}

	if service.ConnectionUdpTimeout != 0 {
		udpFlows := service.AssemblerUdp.CompleteOlderThan(thresholdUdp)
		for _, flow := range udpFlows {
			reassemblyCallback(*flow)
		}

		if len(udpFlows) != 0 {
			log.Println("Assembled", len(udpFlows), "udp flows")
		}
	}
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

	if *pcap_over_ip == "" {
		*pcap_over_ip = os.Getenv("PCAP_OVER_IP")
	}

	if *bpf == "" {
		*bpf = os.Getenv("BPF")
	}

	db_string := "mongodb://" + *mongodb
	log.Println("Connecting to MongoDB:", db_string, "...")
	g_db = db.ConnectMongo(db_string)
	log.Println("Connected, configuring MongoDB database")
	g_db.ConfigureDatabase()
	service := NewAssemblerService()
	service.BpfFilter = *bpf;

	// Parse flush duration parameter (TCP)
	if *flushAfter != "" {
		flushDuration, err := time.ParseDuration(*flushAfter)
		if err != nil {
			log.Fatal("Invalid flush-after duration: ", *flushAfter)
		}

		service.ConnectionTcpTimeout = flushDuration
	}

	// Parse flush duration parameter (UDP)
	if *flushAfterUdp != "" {
		flushDurationUdp, err := time.ParseDuration(*flushAfterUdp)
		if err != nil {
			log.Fatal("Invalid flush-after-udp duration: ", *flushAfterUdp)
		}

		service.ConnectionUdpTimeout = flushDurationUdp
	}

	// Parse flush interval
	if *flushAfter != "" {
		flushIntervalDuration, err := time.ParseDuration(*flushInterval)
		if err != nil {
			log.Fatal("Invalid flush-interval duration: ", *&flushIntervalDuration)
		}

		service.FlushInterval = flushIntervalDuration
	}

	// Pass positional arguments to the pcap handler
	for _, uri := range flag.Args() {
		service.HandlePcapUri(uri)
	}

	// If PCAP-over-IP was configured, connect to it
	// NOTE: Configuring PCAP-over-IP ignores watch dir
    if *pcap_over_ip != "" {
        // for handling multiple pcap over ip
        if strings.Contains(*pcap_over_ip, ",") {
            pcapOverIPs := strings.Split(*pcap_over_ip, ",")
            waitGroup := sync.WaitGroup{}
            waitGroup.Add(len(pcapOverIPs))
            for _, pcapIP := range pcapOverIPs {
                go func(pcapIP string) {
                    defer waitGroup.Done()
                    connectToPCAPOverIP(service, pcapIP)
                }(pcapIP)
            }
            waitGroup.Wait()
        } else {
            connectToPCAPOverIP(service, *pcap_over_ip)
        }
    } else {
        // If a watch dir was configured, handle all files in the directory, then
        // keep monitoring it for new files.
        if *watch_dir != "" {
            service.WatchDir(*watch_dir)
        }
    }
}

func connectToPCAPOverIP(service AssemblerService, pcapIP string) {
    for {
        time.Sleep(5 * time.Second)
        log.Println("Connecting to PCAP-over-IP:", pcapIP)
        tcpServer, err := net.ResolveTCPAddr("tcp", pcapIP)
        if err != nil {
            log.Println(err)
            continue
        }
        conn, err := net.DialTCP("tcp", nil, tcpServer)
        if err != nil {
            log.Println(err)
            continue
        }
        pcapFile, err := conn.File()
        if err != nil {
            log.Println(err)
            conn.Close()
            continue
        }
        // Name the file uniquely per connection to not skip packets on reconnect
        fname := pcapIP + ":" + fmt.Sprintf("%d", time.Now().Unix())

        log.Println("Connected to PCAP-over-IP:", fname)
        service.HandlePcapFile(pcapFile, fname)
        log.Println("Disconnected from PCAP-over-IP:", fname)
        conn.Close()
        pcapFile.Close()
    }
}

func (service AssemblerService) WatchDir(watch_dir string) {
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
		// accepts files with prefixes that start with .pcap (.pcapng .pcap1 etc)
		if strings.HasPrefix(filepath.Ext(file.Name()),".pcap") {
			service.HandlePcapUri(filepath.Join(watch_dir, file.Name())) //FIXME; this is a little clunky
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
				if event.Op&(fsnotify.Rename|fsnotify.Create|fsnotify.Write) != 0 {
					// accepts files with prefixes that start with .pcap (.pcapng .pcap1 etc)
					if strings.HasPrefix(filepath.Ext(event.Name), ".pcap") {
						log.Println("Found new file", event.Name, event.Op.String())
						time.Sleep(2 * time.Second) // FIXME; bit of race here between file creation and writes.
						service.HandlePcapUri(event.Name)
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

func (service AssemblerService) HandlePcapUri(fname string) {
	var handle *pcap.Handle
	var err error

	if handle, err = pcap.OpenOffline(fname); err != nil {
		log.Println("PCAP OpenOffline error:", err)
		return
	}
	defer handle.Close()

	service.ProcessPcapHandle(handle, fname)
}

func (service AssemblerService) HandlePcapFile(file *os.File, fname string) {
	var handle *pcap.Handle
	var err error

	if handle, err = pcap.OpenOfflineFile(file); err != nil {
		log.Println("PCAP OpenOfflineFile error:", err)
		return
	}
	defer handle.Close()

	service.ProcessPcapHandle(handle, fname)
}

func (service AssemblerService) ProcessPcapHandle(handle *pcap.Handle, fname string) {
	if service.BpfFilter != "" {
		if err := handle.SetBPFFilter(service.BpfFilter); err != nil {
			log.Println("Set BPF Filter error: ", err)
			return
		}
	}

	processedCount := int64(0)
	processedExists, processedPcap := g_db.GetPcap(fname)
	if processedExists {
		processedCount = processedPcap.Position
		log.Println("Skipped", processedCount, "packets from file", fname)
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
	count := int64(0)
	bytes := int64(0)
	lastFlush := time.Now()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	service.FlushConnections()

	for packet := range source.Packets() {
		count++

		// Skip packets that were already processed from this pcap
		if count < processedCount + 1 {
			continue
		}

		data := packet.Data()
		bytes += int64(len(data))
		done := false

		// defrag the IPv4 packet if required
		// (TODO; IPv6 will not be defragged)
		ip4Layer := packet.Layer(layers.LayerTypeIPv4)
		if !nodefrag && ip4Layer != nil {
			ip4 := ip4Layer.(*layers.IPv4)
			l := ip4.Length
			newip4, err := service.Defragmenter.DefragIPv4(ip4)
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
				flow := packet.NetworkLayer().NetworkFlow()
				captureInfo := packet.Metadata().CaptureInfo;
				captureInfo.AncillaryData = []interface{}{ fname };
				context := &Context { CaptureInfo: captureInfo };
				service.AssemblerTcp.AssembleWithContext(flow, tcp, context)
				break
			case layers.LayerTypeUDP:
				udp := transport.(*layers.UDP)
				flow := packet.NetworkLayer().NetworkFlow()
				captureInfo := packet.Metadata().CaptureInfo;
				service.AssemblerUdp.Assemble(flow, udp, &captureInfo, fname)
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

		// Try flushing connections here. When using PCAP-over-IP this is required, since it treats whole connection as one pcap.
		// NOTE: PCAP-over-IP: pcapOpenOfflineFile is blocking so we need at least see some packets passing by to get here.
		if service.FlushInterval != 0 && lastFlush.Add(service.FlushInterval).Unix() < time.Now().Unix() {
			service.FlushConnections()
			log.Println("Processed", count - processedCount, "packets from", fname)
			lastFlush = time.Now()
		}
	}

	log.Println("Processed", count - processedCount, "packets from", fname)
	g_db.InsertPcap(fname, count)
}
