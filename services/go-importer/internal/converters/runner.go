package converters

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"go-importer/internal/pkg/db"
	"log"
	"os/exec"
)

// TODO: we need some configuration file for this/re-use configuration.py somehow
// Waterfall-like effect, each stage's outputs keep falling towards next group, e.g.
// using 2 converters will cause the next group to get the output of those two passed to it.
// Additionally, the original entry is always sent to all of the groups.
var serviceConfig = map[int][][]string{
	3003: {
		// Protocol
		{"websockets"},
		// Various encodings one could use (should always be last)
		//{"b64decode"},
	},
}

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

// TODO: consider dropping json, too slow
func TryConverter(converter string, entry db.FlowEntry) (db.FlowEntry, error) {
	// TODO: cache this
	path, err := exec.LookPath("python3")
	if err != nil {
		return db.FlowEntry{}, fmt.Errorf("failed to find python3: %v", err)
	}

	// TODO: timeout
	cmd := exec.Command(path, fmt.Sprintf("converters/%s.py", converter))

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return db.FlowEntry{}, fmt.Errorf("failed to create stdin pipe: %v", err)
	}

	stdout := bytes.Buffer{}
	cmd.Stdout = &stdout

	stderr := bytes.Buffer{}
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return db.FlowEntry{}, fmt.Errorf("failed starting decoder: %v", err)
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return db.FlowEntry{}, fmt.Errorf("failed to marshal flow entry: %v", err)
	}

	if _, err := stdin.Write(append(data, 0xa)); err != nil {
		return db.FlowEntry{}, fmt.Errorf("failed to write to stdin the flow entry: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		return db.FlowEntry{}, fmt.Errorf("decoder failed running: %v", err)
	}

	var streamChunks []StreamChunk
	if err := json.Unmarshal(stdout.Bytes(), &streamChunks); err != nil {
		// TODO: remove after debug
		fmt.Println(string(data))
		fmt.Println(stdout.String())
		return db.FlowEntry{}, fmt.Errorf("failed unmarshaling decoder output: %v", err)
	}

	// TODO: we need some checks to guarantee the output actually changes - otherwise it's of no value to us
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
