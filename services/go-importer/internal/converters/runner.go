package converters

import (
	"fmt"
	"go-importer/internal/pkg/db"
	"log"
	"time"
)

func RunPipeline(g_db *db.Database, entry *db.FlowEntry) {
	// TODO: should we also check src port?
	config, ok := serviceConfig[int(entry.Dst_port)]
	if !ok {
		return
	}

	for _, converters := range config {
		// Split flows into groups by their kinds
		var flows map[string][]db.FlowItem
		for _, item := range entry.Flow {
			_, ok := flows[item.Kind]
			if !ok {
				flows[item.Kind] = []db.FlowItem{}
			}

			flows[item.Kind] = append(flows[item.Kind], item)
		}

		for parent_kind, items := range flows {
			for _, converter := range converters {
				child_kind := fmt.Sprintf("%s -> %s", parent_kind, converter)

				converterFlow, err := TryConverter(converter, entry, items)
				if err != nil {
					log.Printf("WARN: Failed to run converter %s: %s\n", converter, err.Error())
					continue
				}

				// Something went wrong or there's no difference in the data
				if len(converterFlow) == 0 {
					continue
				}

				for _, flow := range converterFlow {
					flow.Kind = child_kind
					entry.Flow = append(entry.Flow, flow)
				}
			}
		}
	}
}

type RequestChunk struct {
	Src_ip   string
	Src_port uint16
	Dst_ip   string
	Dst_port uint16
	Flow     []db.FlowItem
}

type ProcessedChunk struct {
	From string
	Data []byte
	Time time.Time
}

func TryConverter(converter string, entry *db.FlowEntry, flow []db.FlowItem) ([]db.FlowItem, error) {
	process, err := GetWorker(converter)
	if err != nil {
		return nil, fmt.Errorf("failed to get worker for converter %s: %w", converter, err)
	}

	process.RestartMutex.RLock()
	restarting := process.Restarting
	process.RestartMutex.RUnlock()

	if restarting {
		<-process.RestartWaiter
	}

	process.Mutex.Lock()
	defer process.Mutex.Unlock()

	ch := make(chan error, 1)

	var streamChunks []ProcessedChunk
	go func() {
		if err := process.Encoder.Encode(RequestChunk{
			Src_ip:   entry.Src_ip.String(),
			Src_port: entry.Src_port,
			Dst_ip:   entry.Dst_ip.String(),
			Dst_port: entry.Dst_port,
			Flow:     flow,
		}); err != nil {
			ch <- fmt.Errorf("failed to marshal flow entry: %w", err)
			return
		}

		if err := process.Decoder.Decode(&streamChunks); err != nil {
			ch <- fmt.Errorf("failed to unmarshal stream chunks: %w", err)
			return
		}

		if len(flow) != 0 {
			streamChunks[len(streamChunks)].Time = flow[0].Time
		}

		ch <- nil
	}()

	select {
	case <-time.After(time.Second):
		log.Printf("WARN: Converter %s somehow timed out, restarting it...\n", converter)
		if err := process.Restart(); err != nil {
			log.Printf("WARN: Failed to restart the converter: %s\n", err.Error())
		}

		// Trying again will (most likely) just lead to it crashing again
		return nil, fmt.Errorf("timed out encoding flow entry")
	case err := <-ch:
		if err != nil {
			return nil, err
		}
	}

	// TODO: pkappa2 does some post-processing here - same direction streams are merged into one (is this worth the effort?)

	var flowItems []db.FlowItem
	for _, chunk := range streamChunks {
		flowItems = append(flowItems, db.FlowItem{
			From: chunk.From,
			Data: chunk.Data,
			Time: chunk.Time,
		})
	}

	return flowItems, nil
}
