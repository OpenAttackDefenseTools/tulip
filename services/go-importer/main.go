// Copyright 2012 Google, Inc. All rights reserved.
//
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file in the root of the source
// tree.

// The pcapdump binary implements a tcpdump-like command line tool with gopacket
// using pcap as a backend data collection mechanism.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/google/gopacket"
	"github.com/google/gopacket/examples/util"
	"github.com/google/gopacket/ip4defrag"
	"github.com/google/gopacket/layers" // pulls in all layers decoders
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

var db database

// TODO; FIXME; RDJ; this is kinda gross, but this is PoC level code
func reassemblyCallback(entry flowEntry) {
	db.InsertFlow(entry)
}

func main() {
	defer util.Run()()

	if flag.NArg() < 1 {
		log.Fatal("Usage: ./go-importer <file0.pcap> ... <fileN.pcap>")
	}

	// TODO; Make this configurable
	db = ConnectMongo("mongodb://mongo:27017")

	for _, uri := range flag.Args() {
		if db.InsertPcap(uri) {
			log.Println("Processing file:", uri)
			handlePcap(uri)
		} else {
			log.Println("Skipped: ", uri)
		}
	}

}

func handlePcap(fname string) {
	var handle *pcap.Handle
	var err error

	if handle, err = pcap.OpenOffline(fname); err != nil {
		log.Fatal("PCAP OpenOffline error:", err)
	}

	var dec gopacket.Decoder
	var ok bool
	decoder_name := fmt.Sprintf("%s", handle.LinkType())
	if dec, ok = gopacket.DecodersByLayerName[decoder_name]; !ok {
		log.Fatalln("No decoder named", decoder_name)
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
