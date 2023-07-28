package converters

import (
	"encoding/base64"
	"fmt"
	"go-importer/internal/pkg/db"
	"log"
)

func RunPipeline(originalEntry db.FlowEntry) []db.FlowEntry {
	// TODO: should we also check src port?
	config, ok := serviceConfig[originalEntry.Dst_port]
	if !ok {
		return nil
	}

	entries := []db.FlowEntry{originalEntry}
	for _, converters := range config {
		for _, converter := range converters {
			for _, entry := range entries {
				converterEntry, err := TryConverter(converter, entry)
				if err != nil {
					// This is most likely a useless print outside debug purposes
					log.Printf("WARN: Failed to run converter %s: %s\n", converter, err.Error())
					continue
				}

				entries = append(entries, converterEntry) // TODO: does this affect the range loop?

				break // TODO: we want only first (i.e. original) entry to be ran for now as it has b64
				// TODO: another alternative is to handle the insertion here instead?
			}
		}
	}

	if len(entries) > 1 {
		return entries[1:] // Do not return the original entry
	}

	return nil
}

type StreamChunk struct {
	From          string
	Base64Content string `json:"base64_content"`
}

func TryConverter(converter string, entry db.FlowEntry) (db.FlowEntry, error) {
	process, err := GetWorker(converter)
	if err != nil {
		return db.FlowEntry{}, fmt.Errorf("failed to get worker for converter %s: %w", converter, err)
	}
	process.Mutex.Lock()
	defer process.Mutex.Unlock()

	// TODO: some kind of timeout mechanism

	if err := process.Encoder.Encode(entry); err != nil {
		return db.FlowEntry{}, fmt.Errorf("failed to marshal flow entry: %w", err)
	}

	var streamChunks []StreamChunk
	if err := process.Decoder.Decode(&streamChunks); err != nil {
		return db.FlowEntry{}, fmt.Errorf("failed to unmarshal stream chunks: %w", err)
	}

	// TODO: pkappa2 does some post-processing here - same direction streams are merged into one (is this worth the effort?)
	// TODO: if streamChunks is empty, assume error/failure

	// TODO: we need some checks to guarantee the output actually changes (on python side?) - otherwise it's of no value to us
	var flowItems []db.FlowItem
	for _, chunk := range streamChunks {
		// TODO: refactor how data is passed around in tulip to []byte?
		data, err := base64.StdEncoding.DecodeString(chunk.Base64Content)
		if err != nil {
			log.Printf("WARN: Failed to decode base64 from stream chunk: %s\n", err.Error())
			continue
		}

		flowItems = append(flowItems, db.FlowItem{
			From: chunk.From,
			Data: string(data),
			B64:  "",
			Time: 0, // TODO: how much do we need this?
		})
	}

	return db.FlowEntry{
		Src_port:     entry.Src_port,
		Dst_port:     entry.Dst_port,
		Src_ip:       entry.Src_ip,
		Dst_ip:       entry.Dst_ip,
		Time:         entry.Time,
		Duration:     entry.Duration,
		Num_packets:  entry.Num_packets,
		Blocked:      entry.Blocked,
		Filename:     entry.Filename,
		Parent_id:    entry.Parent_id,
		Child_id:     entry.Child_id,
		Fingerprints: entry.Fingerprints,
		Suricata:     entry.Suricata,
		Flow:         flowItems,
		Tags:         entry.Tags,
		Size:         entry.Size,
	}, nil
}
