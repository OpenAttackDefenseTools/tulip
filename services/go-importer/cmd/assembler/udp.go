package main

import (
	"go-importer/internal/pkg/db"

	"time"
	"net/netip"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

type UdpAssembler struct {
	Streams map[UdpStreamIdendifier]*UdpStream
}

func NewUdpAssembler() UdpAssembler {
	return UdpAssembler{
		Streams: map[UdpStreamIdendifier]*UdpStream{},
	}
}

func (assembler *UdpAssembler) Assemble(flow gopacket.Flow, udp *layers.UDP, captureInfo *gopacket.CaptureInfo, source string) *UdpStream {
	endpointSrc := flow.Src().FastHash()
	endpointDst := flow.Dst().FastHash()
	portSrc := uint16(udp.SrcPort)
	portDst := uint16(udp.DstPort)
	id := UdpStreamIdendifier{}

	if endpointSrc > endpointDst {
		id.EndpointLower = endpointDst
		id.EndpointUpper = endpointSrc
	} else {
		id.EndpointLower = endpointSrc
		id.EndpointUpper = endpointDst
	}

	if portSrc > portDst {
		id.PortLower = portDst
		id.PortUpper = portSrc
	} else {
		id.PortLower = portSrc
		id.PortUpper = portDst
	}

	stream, ok := assembler.Streams[id]
	if !ok {
		stream = &UdpStream{
			Identifier: id,
			Flow:       flow,
			PortSrc:    udp.SrcPort,
			PortDst:    udp.DstPort,
			Source:     source,
		}

		assembler.Streams[id] = stream
	}

	stream.ProcessSegment(flow, udp, captureInfo)
	return stream
}

func (assembler *UdpAssembler) CompleteOlderThan(threshold time.Time) []*db.FlowEntry {
	flows := make([]*db.FlowEntry, 0)

	for id, stream := range assembler.Streams {
		if stream.LastSeen.Unix() < threshold.Unix() {
			flow := stream.CompleteReassembly()
			if flow != nil {
				flows = append(flows, flow)
			}
			delete(assembler.Streams, id)
		}
	}

	return flows
}

type UdpStreamIdendifier struct {
	EndpointLower uint64
	EndpointUpper uint64
	PortLower     uint16
	PortUpper     uint16
}

type UdpStream struct {
	Identifier  UdpStreamIdendifier
	Flow        gopacket.Flow
	PacketCount uint
	PacketSize  uint
	Items       []db.FlowItem
	PortSrc     layers.UDPPort
	PortDst     layers.UDPPort
	Source      string
	LastSeen    time.Time
}

func (stream *UdpStream) ProcessSegment(flow gopacket.Flow, udp *layers.UDP, captureInfo *gopacket.CaptureInfo) {
	if len(udp.Payload) == 0 {
		return
	}

	from := "s"
	if flow.Dst().FastHash() == stream.Flow.Src().FastHash() {
		from = "c"
	}

	stream.LastSeen = captureInfo.Timestamp
	stream.PacketCount += 1
	stream.PacketSize += uint(len(udp.Payload))

	// We have to make sure to stay under the document limit
	available := uint(*maxFlowItemSize * 1024 * 1024) - stream.PacketSize
	length := uint(len(udp.Payload))
	if length > available {
		length = available
	}
	if length < 0 {
		length = 0
	}

	stream.Items = append(stream.Items, db.FlowItem{
		Kind: "raw",
		From: from,
		Data: udp.Payload[:length],
		Time: captureInfo.Timestamp,
	})
}

func (stream *UdpStream) CompleteReassembly() *db.FlowEntry {
	if len(stream.Items) == 0 {
		return nil
	}

	src, dst := stream.Flow.Endpoints()
	ip_src, _ := netip.ParseAddr(src.String())
	ip_dst, _ := netip.ParseAddr(dst.String())

	timeStart := stream.Items[0].Time
	timeEnd := stream.Items[0].Time
	for _, item := range stream.Items {
		if timeEnd.Before(item.Time) {
			timeEnd = item.Time
		}
	}

	return &db.FlowEntry{
		Src_port:    uint16(stream.PortSrc),
		Dst_port:    uint16(stream.PortDst),
		Src_ip:      ip_src,
		Dst_ip:      ip_dst,
		Time:        timeStart,
		Duration:    timeEnd.Sub(timeStart),
		Num_packets: int(stream.PacketCount),
		Parent_id:   nil,
		Child_id:    nil,
		Tags:        []string{"udp"},
		Filename:    stream.Source,
		Flow:        stream.Items,
		Size:        int(stream.PacketSize),
		Flags:       make([]string, 0),
		Flagids:     make([]string, 0),
	}
}
