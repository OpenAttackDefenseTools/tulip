// Copyright 2012 Google, Inc. All rights reserved.
//
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file in the root of the source
// tree.

// The pcapdump binary implements a tcpdump-like command line tool with gopacket
// using pcap as a backend data collection mechanism.
package main

import (
	"go-importer/internal/pkg/db"

	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/reassembly"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var allowmissinginit = true
var verbose = false
var debug = false
var quiet = true

const closeTimeout time.Duration = time.Hour * 24 // Closing inactive: TODO: from CLI
const timeout time.Duration = time.Minute * 5     // Pending bytes: TODO: from CLI
const streamdoc_limit int = 6_000_000 - 0x1000    // 16 MB (6 + (4/3)*6) - some overhead

/*
 * The TCP factory: returns a new Stream
 */
type TcpStreamFactory struct {
	reassemblyCallback func(db.FlowEntry)
}

func (factory *TcpStreamFactory) New(net, transport gopacket.Flow, tcp *layers.TCP, ac reassembly.AssemblerContext) reassembly.Stream {
	source := ac.GetCaptureInfo().AncillaryData[0].(string);
	fsmOptions := reassembly.TCPSimpleFSMOptions{
		SupportMissingEstablishment: *nonstrict,
	}
	stream := &TcpStream{
		net:                net,
		transport:          transport,
		tcpstate:           reassembly.NewTCPSimpleFSM(fsmOptions),
		optchecker:         reassembly.NewTCPOptionCheck(),
		source:             source,
		FlowItems:          []db.FlowItem{},
		src_port:           tcp.SrcPort,
		dst_port:           tcp.DstPort,
		reassemblyCallback: factory.reassemblyCallback,
	}
	return stream
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

/* It's a connection (bidirectional) */
type TcpStream struct {
	tcpstate       *reassembly.TCPSimpleFSM
	fsmerr         bool
	optchecker     reassembly.TCPOptionCheck
	net, transport gopacket.Flow
	sync.Mutex
	// RDJ; These field are added to make mongo convertion easier
	source             string
	reassemblyCallback func(db.FlowEntry)
	FlowItems          []db.FlowItem
	src_port           layers.TCPPort
	dst_port           layers.TCPPort
	total_size         int
	num_packets        int
}

func (t *TcpStream) Accept(tcp *layers.TCP, ci gopacket.CaptureInfo, dir reassembly.TCPFlowDirection, nextSeq reassembly.Sequence, start *bool, ac reassembly.AssemblerContext) bool {
	// FSM
	if !t.tcpstate.CheckState(tcp, dir) {
		if !t.fsmerr {
			t.fsmerr = true
		}
		if !*nonstrict {
			return false
		}
	}

	// We just ignore the Checksum
	return true
}

// ReassembledSG is called zero or more times.
// ScatterGather is reused after each Reassembled call,
// so it's important to copy anything you need out of it,
// especially bytes (or use KeepFrom())
func (t *TcpStream) ReassembledSG(sg reassembly.ScatterGather, ac reassembly.AssemblerContext) {
	dir, _, _, _ := sg.Info()
	length, _ := sg.Lengths()
	capInfo := ac.GetCaptureInfo()
	timestamp := capInfo.Timestamp
	t.num_packets += 1

	// Don't add empty streams to the DB
	if length == 0 {
		return
	}

	data := sg.Fetch(length)

	// We have to make sure to stay under the document limit
	t.total_size += length
	bytes_available := streamdoc_limit - t.total_size
	if length > bytes_available {
		length = bytes_available
	}
	if length < 0 {
		length = 0
	}
	string_data := string(data[:length])

	var from string
	if dir == reassembly.TCPDirClientToServer {
		from = "c"
	} else {
		from = "s"
	}

	// consolidate subsequent elements from the same origin
	l := len(t.FlowItems)
	if l > 0 {
		if t.FlowItems[l-1].From == from {
			t.FlowItems[l-1].Data += string_data
			// All done, no need to add a new item
			return
		}
	}

	// Add a FlowItem based on the data we just reassembled
	t.FlowItems = append(t.FlowItems, db.FlowItem{
		Data: string_data,
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
func (t *TcpStream) ReassemblyComplete(ac reassembly.AssemblerContext) bool {

	// Insert the stream into the mogodb.

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
		}
	*/
	src, dst := t.net.Endpoints()
	var time, duration int
	if len(t.FlowItems) == 0 {
		// No point in inserting this element, it has no data and even if we wanted to,
		// we can't timestamp it so the front-end can't display it either
		return false
	}

	time = t.FlowItems[0].Time
	duration = t.FlowItems[len(t.FlowItems)-1].Time - time

	entry := db.FlowEntry{
		Src_port:    int(t.src_port),
		Dst_port:    int(t.dst_port),
		Src_ip:      src.String(),
		Dst_ip:      dst.String(),
		Time:        time,
		Duration:    duration,
		Num_packets: t.num_packets,
		Parent_id:   primitive.NilObjectID,
		Child_id:    primitive.NilObjectID,
		Blocked:     false,
		Tags:        []string { "tcp" },
		Suricata:    make([]int, 0),
		Filename:    t.source,
		Flow:        t.FlowItems,
		Size:        t.total_size,
	}

	t.reassemblyCallback(entry)

	// do not remove the connection to allow last ACK
	return false
}
