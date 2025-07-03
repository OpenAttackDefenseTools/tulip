package main

import (
	"go-importer/internal/converters"
	"go-importer/internal/pkg/db"
	"io/ioutil"
	"runtime"

	"github.com/gammazero/workerpool"

	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/google/gopacket"
	"github.com/google/gopacket/examples/util"
	"github.com/google/gopacket/ip4defrag"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/pcapgo"
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
var timescale = flag.String("timescale", "", "Timescale connection string (e. g. postgres://usr:pwd@host:5432/tulip)")
var flag_regex = flag.String("flag", "", "flag regex, used for flag in/out tagging")
var pcap_over_ip = flag.String("pcap-over-ip", "", "PCAP-over-IP host + port (e.g. remote:1337)")
var bpf = flag.String("bpf", "", "BPF filter")
var nonstrict = flag.Bool("nonstrict", false, "Do not check strict TCP / FSM flags")

var flagid = flag.Bool("flagid", false, "Check for flagids in traffic (must be present in mong)")
var ticklength = flag.Int("tick-length", -1, "the length (in seconds) of a tick")
var flaglifetime = flag.Int("flag-lifetime", -1, "the lifetime of a flag in ticks")
var flagTickStartRaw = flag.String("flag-tick-start", "", "CTF start time (used for flag validation)")
var flagTickStart time.Time
var flagValidatorType = flag.String("flag-validator-type", "", "Flag validator type, this must be set to enable flag validation. Must be one of the following: FAUST, ENO/ENOWARS, ITAD")
var flagValidatorTeam = flag.Int("flag-validator-team", -1, "Team ID used for flag validation")

var skipchecksum = flag.Bool("skipchecksum", false, "Do not check the TCP checksum")
var http_session_tracking = flag.Bool("http-session-tracking", false, "Enable http session tracking.")
var disableConverters = flag.Bool("disable-converters", false, "Disable converters in case they cause issues")
var concurrentConverters = flag.Int("concurrent-converters", 2, "How many processes should be started per single converter")
var concurrentFlows = flag.Int("concurrent-flows", 0, "How many flows should be processed at the same time")

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
var dumpPcaps = flag.String("dump-pcaps", "", `Generate a pcap in this directory every "dump-pcaps-interval".
Empty string (default) disables this behavior. This is useful for saving pcaps from PCAP-over-IP.`)
var dumpPcapsInterval = flag.String("dump-pcaps-interval", "5m", `Period for PCAP dumping. Requres "dump-pcaps" to be set.
Any string parsed by time.ParseDuration is acceptable here (ie. "3m", "2h45m").`)
var dumpPcapsFilename = flag.String("dump-pcaps-filename", "2006-01-02_15-04-05.pcap", `Filename for dumped PCAP.
Reference: https://pkg.go.dev/time#Layout`)
var maxFlowItemSize = flag.Int("max-flow-item-size", 16, `Maximum size in MiB of one flow item record.
While PostgreSQL technically supports values up to 1GiB, they are not very nice to work with.`)

var g_db *db.Database
var workerPool *workerpool.WorkerPool
var flagValidator FlagValidator

// flagid caching (only once per tick)
var flagids []db.FlagId
var flagidUpdate int64 = 0

// TODO; FIXME; RDJ; this is kinda gross, but this is PoC level code
func reassemblyCallback(entry db.FlowEntry) {
	// By default, the callback passed is blocking per single packet. If for some reason converters hang,
	// we *really* don't want to end up in a situation where we don't get any packets ingested until the converter
	// times out.
	workerPool.Submit(func() {
		// Parsing HTTP will decode encodings to a plaintext format
		ParseHttpFlow(g_db, &entry)

		if !*disableConverters {
			converters.RunPipeline(g_db, &entry)
		}

		// Apply flag in / flagout
		if *flag_regex != "" {
			ApplyFlagTags(&entry, flag_regex, flagValidator)
		}

		// Apply flagid in / out
		if *flagid {
			unix := time.Now().Unix()
			if flagidUpdate+int64(*ticklength) < unix {
				flagidUpdate = unix
				zwi, err := g_db.FlagIdsQuery(*flaglifetime)
				if err != nil {
					log.Fatal(err)
				}
				flagids = zwi
			}
			ApplyFlagids(&entry, flagids)
		}

		// Finally, insert the new entry
		g_db.FlowInsert(entry)
	})
}

type AssemblerService struct {
	Defragmenter         *ip4defrag.IPv4Defragmenter
	StreamFactory        *TcpStreamFactory
	StreamPool           *reassembly.StreamPool
	AssemblerTcp         *reassembly.Assembler
	AssemblerUdp         *UdpAssembler
	ConnectionTcpTimeout time.Duration
	ConnectionUdpTimeout time.Duration
	FlushInterval        time.Duration
	BpfFilter            string
	PcapOverIp           bool
	DumpDirectory        string
	DumpInterval         time.Duration
	DumpFile             *os.File
	DumpWriter           *pcapgo.Writer
	DumpLast             time.Time
	DumpCount            uint64
	DumpFilename         string
}

func NewAssemblerService() *AssemblerService {
	streamFactory := &TcpStreamFactory{reassemblyCallback: reassemblyCallback}
	streamPool := reassembly.NewStreamPool(streamFactory)
	assemblerUdp := NewUdpAssembler()

	return &AssemblerService{
		Defragmenter:  ip4defrag.NewIPv4Defragmenter(),
		StreamFactory: streamFactory,
		StreamPool:    streamPool,
		AssemblerTcp:  reassembly.NewAssembler(streamPool),
		AssemblerUdp:  &assemblerUdp,
		DumpLast:      time.Now(),
	}
}

func (service *AssemblerService) FlushConnections() {
	thresholdTcp := time.Now().Add(-service.ConnectionTcpTimeout)
	thresholdUdp := time.Now().Add(-service.ConnectionUdpTimeout)
	flushed, closed, discarded := 0, 0, 0

	if service.ConnectionTcpTimeout != 0 {
		flushed, closed = service.AssemblerTcp.FlushCloseOlderThan(thresholdTcp)
		discarded = service.Defragmenter.DiscardOlderThan(thresholdTcp)
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

	// DELAY for testing
	strdelay := os.Getenv("DELAY")
	if strdelay != "" {
		delay, err := strconv.Atoi(strdelay)
		if err != nil {
			log.Println("Error: ", err)
		} else {
			time.Sleep(time.Second * time.Duration(delay))
		}
	}

	// get TICK_LENGTH
	strticklength := os.Getenv("TICK_LENGTH")
	if *ticklength == -1 && strticklength != "" {
		zwi, err := strconv.ParseInt(strticklength, 10, 64)
		if err != nil {
			log.Println("Error: ", err)
		} else {
			*ticklength = int(zwi / 1000)
		}
	}

	// get Flag_LIFETIME
	if strflaglifetime := os.Getenv("FLAG_LIFETIME"); *flaglifetime == -1 && strflaglifetime != "" {
		zwi, err := strconv.Atoi(strflaglifetime)
		if err != nil {
			log.Println("Error: ", err)
		} else {
			*flaglifetime = zwi
		}
	}

	if *ticklength != -1 && *flaglifetime != -1 {
		*flaglifetime *= *ticklength
	}

	// get TICK_START
	if *flagTickStartRaw == "" {
		*flagTickStartRaw = os.Getenv("TICK_START")
	}
	if *flagTickStartRaw != "" {
		startTime, err := time.Parse("2006-01-02T15:04Z07:00", *flagTickStartRaw)
		if err != nil {
			// If that format fail, we try it to parse it as RFC3339 ("2006-01-02T15:04:05Z07:00")
			startTime, err = time.Parse(time.RFC3339, *flagTickStartRaw)
		}
		if err != nil {
			log.Fatal("Invalid start time: ", err)
		}
		flagTickStart = startTime
	} 

	if concurrentFlows == nil || *concurrentFlows == 0 {
		*concurrentFlows = runtime.NumCPU() / 2
		if *concurrentFlows < 4 {
			*concurrentFlows = 4
		}
	}

	workerPool = workerpool.New(*concurrentFlows)

	// If no timescale connection string was supplied, use env variable
	if *timescale == "" {
		*timescale = os.Getenv("TIMESCALE")
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

	// if flagid scans should be done
	if !*flagid {
		flagid_val := os.Getenv("FLAGID_SCAN")
		*flagid = flagid_val != "" && flagid_val != "0" && !strings.EqualFold(flagid_val, "false")

	}

	if *bpf == "" {
		*bpf = os.Getenv("BPF")
	}

	// Load flag validator variables
	if *flagValidatorType == "" {
		*flagValidatorType = os.Getenv("FLAG_VALIDATOR_TYPE")
	}
	if unparsed := os.Getenv("FLAG_VALIDATOR_TEAM"); *flagValidatorTeam == -1 && unparsed != "" {
		parsed, err := strconv.Atoi(unparsed)
		if err != nil {
			log.Fatal("Invalid flag validator team: ", err)
		}
		*flagValidatorTeam = parsed
	}
	
	// Flag validator setup
	if *flagValidatorType != "" && *flag_regex == "" {
		log.Println("WARNING: Flag validation enabled but no flag regex specified. No flag validation will be done.")
	}
	switch strings.ToLower(*flagValidatorType) {
	case "faust":
		flagValidator = &FaustFlagValidator{*flagValidatorTeam, time.Hour, "CTF-GAMESERVER"}
	case "enowars", "eno":
		// I don't think that there will be more than 20 services and 20 flag stores (per service)...
		flagValidator = &EnowarsFlagValidator{
			*flagValidatorTeam,
			20,
			20,
			time.Hour,
			flagTickStart,
			time.Duration(*ticklength) * time.Second,
		}
	case "itad":
		// 20 services should be more than enough...
		flagValidator = &ItallyADFlagValidator{
			*flagValidatorTeam,
			20,
			time.Hour,
			flagTickStart,
			time.Duration(*ticklength) * time.Second,
		}
	case "":
		if *flagValidatorTeam != -1  {
			log.Println("WARNING: No flag validator type specified but additional flag validator options are set. No flag validation will be done.")
		}
		flagValidator = &DummyFlagValidator{}
	default:
		log.Fatalln("Uknown -flag-validator-type: ", *flagValidatorType)
	}


	log.Println("Connecting to Timescale:", *timescale)
	g_db = db.NewDatabase(*timescale)

	service := NewAssemblerService()
	service.BpfFilter = *bpf

	// PCAP dumping parameters
	if os.Getenv("DUMP_PCAPS") != "" {
		*dumpPcaps = os.Getenv("DUMP_PCAPS")
	}
	if os.Getenv("DUMP_PCAPS_INTERVAL") != "" {
		*dumpPcapsInterval = os.Getenv("DUMP_PCAPS_INTERVAL")
	}
	if os.Getenv("DUMP_PCAPS_FILENAME") != "" {
		*dumpPcapsFilename = os.Getenv("DUMP_PCAPS_FILENAME")
	}

	dumpInterval, err := time.ParseDuration(*dumpPcapsInterval)
	if err != nil {
		log.Fatal("Invalid dump-pcaps-interval duration: ", *dumpPcapsInterval)
	}
	service.DumpInterval = dumpInterval
	service.DumpDirectory = *dumpPcaps

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
			log.Fatal("Invalid flush-interval duration: ", *flushInterval)
		}

		service.FlushInterval = flushIntervalDuration
	}

	if !*disableConverters {
		converters.StartWorkers(*concurrentConverters)
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

func connectToPCAPOverIP(service *AssemblerService, pcapIP string) {
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
		sourceName := pcapIP + ":" + fmt.Sprintf("%d", time.Now().Unix())

		log.Println("Connected to PCAP-over-IP:", sourceName)
		service.PcapOverIp = true
		service.HandlePcapFile(pcapFile, sourceName)
		log.Println("Disconnected from PCAP-over-IP:", sourceName)
		conn.Close()
		pcapFile.Close()
	}
}

func (service *AssemblerService) WatchDir(watch_dir string) {
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
		if strings.HasPrefix(filepath.Ext(file.Name()), ".pcap") {
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

func (service *AssemblerService) HandlePcapUri(sourceName string) {
	var handle *pcap.Handle
	var err error

	if handle, err = pcap.OpenOffline(sourceName); err != nil {
		log.Println("PCAP OpenOffline error:", err)
		return
	}
	defer handle.Close()

	service.ProcessPcapHandle(handle, sourceName)
}

func (service *AssemblerService) HandlePcapFile(file *os.File, sourceName string) {
	var handle *pcap.Handle
	var err error

	if handle, err = pcap.OpenOfflineFile(file); err != nil {
		log.Println("PCAP OpenOfflineFile error:", err)
		return
	}
	defer handle.Close()

	service.ProcessPcapHandle(handle, sourceName)
}

func (service *AssemblerService) ProcessPcapHandle(handle *pcap.Handle, sourceName string) {
	if service.BpfFilter != "" {
		if err := handle.SetBPFFilter(service.BpfFilter); err != nil {
			log.Println("Set BPF Filter error: ", err)
			return
		}
	}

	pcap := g_db.PcapFindOrInsert(sourceName)
	if pcap.Position != 0 {
		log.Println("Skipped", pcap.Position, "packets from", sourceName)
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
	service.DumpFlush()

	for packet := range source.Packets() {
		// Try flushing connections here. When using PCAP-over-IP this is required, since it treats whole connection as one pcap.
		// NOTE: PCAP-over-IP: pcapOpenOfflineFile is blocking so we need at least see some packets passing by to get here.
		if service.FlushInterval != 0 && lastFlush.Add(service.FlushInterval).Unix() < time.Now().Unix() {
			service.FlushConnections()
			log.Println("Processed", count - pcap.Position, "packets from", sourceName, "(so far)")
			lastFlush = time.Now()
		}

		count++

		// Skip packets that were already processed from this pcap
		if count < pcap.Position + 1 {
			continue
		}

		// PCAP dump
		service.DumpFlush()
		service.DumpPacket(&packet)

		// Replace name with dumped if PCAP-over-IP is enabled to allow downloads
		flowSourceName := sourceName
		if service.DumpFilename != "" && service.PcapOverIp {
			flowSourceName = service.DumpFilename
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
			captureInfo := packet.Metadata().CaptureInfo
			captureInfo.AncillaryData = []interface{}{flowSourceName}
			context := &Context{CaptureInfo: captureInfo}

			if !*skipchecksum {
				// TODO: sijisu: this is broken
				// Compute the checksum
				tcp.SetNetworkLayerForChecksum(packet.NetworkLayer())
				csum, err := tcp.ComputeChecksum()
				if err != nil {
					fmt.Printf("Failed to compute checksum: %s\n", err)
					break
				}
				// check if the checksum is valid
				if csum != tcp.Checksum {
					fmt.Printf("Invalid checksum: 0x%x\n", csum)
					break
				}
			}

			service.AssemblerTcp.AssembleWithContext(flow, tcp, context)
			break
		case layers.LayerTypeUDP:
			udp := transport.(*layers.UDP)
			flow := packet.NetworkLayer().NetworkFlow()
			captureInfo := packet.Metadata().CaptureInfo
			service.AssemblerUdp.Assemble(flow, udp, &captureInfo, flowSourceName)
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

	g_db.PcapSetPosition(pcap.Id, count)
	service.FlushConnections()
	log.Println("Processed", count - pcap.Position, "packets from", sourceName)
}

func (service *AssemblerService) DumpPacket(packet *gopacket.Packet) {
	if service.DumpDirectory == "" {
		return
	}

	if service.DumpWriter == nil {
		now := time.Now()
		service.DumpFilename = filepath.Join(service.DumpDirectory, now.Format(*dumpPcapsFilename))

		// Do this to make sure we dont try to read this pcap with watch-dir
		pcap := g_db.PcapFindOrInsert(service.DumpFilename)
		g_db.PcapSetPosition(pcap.Id, math.MaxInt64)

		file, err := os.Create(service.DumpFilename)
		if err != nil {
			log.Println("Unable to open PCAP file", service.DumpFilename, err)
			return
		}

		service.DumpFile = file
		service.DumpWriter = pcapgo.NewWriter(service.DumpFile)
		service.DumpLast = now
		service.DumpCount = 0

		err = service.DumpWriter.WriteFileHeader(65536, layers.LinkTypeEthernet)
		if err != nil {
			log.Println("Unable to write packet header", err)
			return
		}

		log.Println("Created PCAP file", service.DumpFilename)
	}

	err := service.DumpWriter.WritePacket((*packet).Metadata().CaptureInfo, (*packet).Data())
	if err != nil {
		log.Println("Unable to write packet", err)
		return
	}
	service.DumpCount += 1
}

func (service *AssemblerService) DumpFlush() {
	if service.DumpWriter != nil && time.Now().Unix() > service.DumpLast.Add(service.DumpInterval).Unix() {
		service.DumpFile.Close()
		service.DumpWriter = nil

		log.Println("Closed PCAP file", service.DumpFilename, "with", service.DumpCount, "packets")
	}
}
