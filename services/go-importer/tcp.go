// Copyright 2012 Google, Inc. All rights reserved.
//
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file in the root of the source
// tree.

// The pcapdump binary implements a tcpdump-like command line tool with gopacket
// using pcap as a backend data collection mechanism.
package main

import (
	"encoding/hex"

	//	"fmt"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/reassembly"
)

var maxcount = -1
var nooptcheck = true
var ignorefsmerr = true
var allowmissinginit = true
var verbose = false
var debug = false
var quiet = true

const closeTimeout time.Duration = time.Hour * 24 // Closing inactive: TODO: from CLI
const timeout time.Duration = time.Minute * 5     // Pending bytes: TODO: from CLI

/*
 * The TCP factory: returns a new Stream
 */
type tcpStreamFactory struct {
	// The source of every tcp stream in this batch.
	// Traditionally, this would be the pcap file name
	source             string
	reassemblyCallback func(f flowEntry)
	wg                 sync.WaitGroup
}

func (factory *tcpStreamFactory) New(net, transport gopacket.Flow, tcp *layers.TCP, ac reassembly.AssemblerContext) reassembly.Stream {
	fsmOptions := reassembly.TCPSimpleFSMOptions{
		SupportMissingEstablishment: allowmissinginit,
	}
	stream := &tcpStream{
		net:                net,
		transport:          transport,
		tcpstate:           reassembly.NewTCPSimpleFSM(fsmOptions),
		optchecker:         reassembly.NewTCPOptionCheck(),
		source:             factory.source,
		flowItems:          []flowItem{},
		src_port:           tcp.SrcPort,
		dst_port:           tcp.DstPort,
		reassemblyCallback: factory.reassemblyCallback,
	}
	return stream
}

func (factory *tcpStreamFactory) WaitGoRoutines() {
	factory.wg.Wait()
}

/*
 * The assembler context
 */
type Context struct {
	CaptureInfo gopacket.CaptureInfo
}

func (c *Context) GetCaptureInfo() gopacket.CaptureInfo {
	return c.CaptureInfo
}

/*
 * TCP stream
 */

/*

/* It's a connection (bidirectional) */
type tcpStream struct {
	tcpstate       *reassembly.TCPSimpleFSM
	fsmerr         bool
	optchecker     reassembly.TCPOptionCheck
	net, transport gopacket.Flow
	sync.Mutex
	// RDJ; These field are added to make mongo convertion easier
	source             string
	reassemblyCallback func(f flowEntry)
	flowItems          []flowItem
	src_port           layers.TCPPort
	dst_port           layers.TCPPort
}

func (t *tcpStream) Accept(tcp *layers.TCP, ci gopacket.CaptureInfo, dir reassembly.TCPFlowDirection, nextSeq reassembly.Sequence, start *bool, ac reassembly.AssemblerContext) bool {
	// FSM
	if !t.tcpstate.CheckState(tcp, dir) {
		if !t.fsmerr {
			t.fsmerr = true
		}
		if !ignorefsmerr {
			return false
		}
	}
	// Options
	err := t.optchecker.Accept(tcp, ci, dir, nextSeq, start)
	if err != nil {
		if !nooptcheck {
			return false
		}
	}
	// We just ignore the `Checksum` for now
	accept := true

	return accept
}

// ReassembledSG is called zero or more times.
// ScatterGather is reused after each Reassembled call,
// so it's important to copy anything you need out of it,
// especially bytes (or use KeepFrom())
func (t *tcpStream) ReassembledSG(sg reassembly.ScatterGather, ac reassembly.AssemblerContext) {
	dir, _, _, _ := sg.Info()
	length, _ := sg.Lengths()
	capInfo := ac.GetCaptureInfo()
	timestamp := capInfo.Timestamp

	// RDJ; don't add empty streams to the DB
	if length == 0 {
		return
	}

	data := sg.Fetch(length)

	var from string
	// This does not make any sense to me (RDJ)
	if dir == reassembly.TCPDirClientToServer {
		from = "s"
	} else {
		from = "c"
	}

	// RDJ; Add a flowItem based on the data we just reassembled
	t.flowItems = append(t.flowItems, flowItem{
		Data: string(data),
		Hex:  hex.EncodeToString(data),
		From: from,
		Time: int(timestamp.UnixNano() / 1000000), // TODO; maybe use int64?
	})

}

// ReassemblyComplete is called when assembly decides there is
// no more data for this Stream, either because a FIN or RST packet
// was seen, or because the stream has timed out without any new
// packet data (due to a call to FlushCloseOlderThan).
// It should return true if the connection should be removed from the pool
// It can return false if it want to see subsequent packets with Accept(), e.g. to
// see FIN-ACK, for deeper state-machine analysis.
func (t *tcpStream) ReassemblyComplete(ac reassembly.AssemblerContext) bool {

	//RDJ; Insert the stream into the mogodb.

	/*
		{
			"src_port": 32858,
			"dst_ip": "10.10.3.1",
			"contains_flag": false,
			"flow": [{}],
			"filename": "services/test_pcap/dump-2018-06-27_13:25:31.pcap",
			"src_ip": "10.10.3.126",
			"dst_port": 8080,
			"time": 1530098789655,
			"duration": 96,
			"inx": 0,
			"starred": 0,
		}
	*/
	src, dst := t.net.Endpoints()
	var time, duration int
	if len(t.flowItems) > 0 {
		time = t.flowItems[0].Time
		duration = t.flowItems[len(t.flowItems)-1].Time - time
	} else {
		time = 0
		duration = 0
	}

	entry := flowEntry{
		Src_port:      int(t.src_port),
		Dst_port:      int(t.dst_port),
		Src_ip:        src.String(),
		Dst_ip:        dst.String(),
		Time:          time,
		Duration:      duration,
		Inx:           0,
		Starred:       0,
		Contains_flag: false,
		Filename:      t.source,
		Flow:          t.flowItems,
	}

	t.reassemblyCallback(entry)

	// do not remove the connection to allow last ACK
	return false
}
