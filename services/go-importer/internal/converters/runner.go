package converters

import (
	"fmt"
	"go-importer/internal/pkg/db"
	"log"
)

func RunPipeline(entry *db.FlowEntry) {
	// TODO: should we also check src port?
	config, ok := serviceConfig[entry.Dst_port]
	if !ok {
		return
	}

	for _, converters := range config {
		for _, converter := range converters {
			flowLen := len(entry.Flow)
			for idx := 0; idx < flowLen; idx++ {
				flow := &entry.Flow[idx]

				converterFlow, err := TryConverter(converter, entry, flow.Flow)
				if err != nil {
					// This is most likely a useless print outside debug purposes
					log.Printf("WARN: Failed to run converter %s: %s\n", converter, err.Error())
					continue
				}

				entry.Flow = append(entry.Flow, db.FlowRepresentation{
					Type: fmt.Sprintf("%s -> %s", flow.Type, converter),
					Flow: converterFlow,
				})
			}
		}
	}
}

type RequestChunk struct {
	Src_ip   string
	Src_port int
	Dst_ip   string
	Dst_port int
	Flow     []db.FlowItem
}

type ProcessedChunk struct {
	From    string
	RawData []byte
}

func TryConverter(converter string, entry *db.FlowEntry, flow []db.FlowItem) ([]db.FlowItem, error) {
	process, err := GetWorker(converter)
	if err != nil {
		return nil, fmt.Errorf("failed to get worker for converter %s: %w", converter, err)
	}
	process.Mutex.Lock()
	defer process.Mutex.Unlock()

	// TODO: some kind of timeout mechanism?

	if err := process.Encoder.Encode(RequestChunk{
		Src_ip:   entry.Src_ip,
		Src_port: entry.Src_port,
		Dst_ip:   entry.Dst_ip,
		Dst_port: entry.Dst_port,
		Flow:     flow,
	}); err != nil {
		return nil, fmt.Errorf("failed to marshal flow entry: %w", err)
	}

	var streamChunks []ProcessedChunk
	if err := process.Decoder.Decode(&streamChunks); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stream chunks: %w", err)
	}
	log.Println(streamChunks)

	// TODO: pkappa2 does some post-processing here - same direction streams are merged into one (is this worth the effort?)
	// TODO: if streamChunks is empty, assume error/failure

	// TODO: we need some checks to guarantee the output actually changes (on python side?) - otherwise it's of no value to us
	var flowItems []db.FlowItem
	for _, chunk := range streamChunks {
		flowItems = append(flowItems, db.FlowItem{
			From:    chunk.From,
			RawData: chunk.RawData,
			Time:    0, // TODO: how much do we need this?
		})
	}

	return flowItems, nil
}
