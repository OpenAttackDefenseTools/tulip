package db

import (
	"context"
	"log"
	"sync"
	"time"

	"go-importer/internal/pkg/event"

	"github.com/jackc/pgx/v5"
)

const BATCH_DEFAULT_SIZE = 1000;
const BATCH_DEFAULT_TIMEOUT = time.Duration(5 * float64(time.Second));

type CopyBatcherConfig struct {
	db *Database
	context context.Context
	tableName pgx.Identifier
	columns []string
	batchSize int
	batchTimeout time.Duration
	errorHook func(*CopyBatcherConfig, error)
}

type CopyBatcher struct {
	dataIn chan<- CopyBatcherItem
	config CopyBatcherConfig
}

func NewCopyBatcher(config CopyBatcherConfig) *CopyBatcher {
	dataIn := make(chan CopyBatcherItem)
	errorOut := make(chan error)

	if config.batchSize == 0 {
		config.batchSize = BATCH_DEFAULT_SIZE
	}

	if config.batchTimeout == 0 {
		config.batchTimeout = BATCH_DEFAULT_TIMEOUT
	}

	if config.context == nil {
		config.context = context.Background()
	}

	if config.errorHook == nil {
		config.errorHook = CopyBatcherLoggerErrorHook
	}

	batcher := &CopyBatcher {
		dataIn: dataIn,
		config: config,
	}

	go func() {
		for {
			err, open := <-errorOut
			if err != nil {
				config.errorHook(&batcher.config, err)
			}
			if !open {
				break
			}
		}
	}()

	go func() {
		defer close(dataIn)
		defer close(errorOut)

		var index int
		batchDataIn, batchErrorIn, batchTimeout := NewCopyBatcherBatch(&batcher.config, errorOut)

		newBatch := func() {
			close(batchDataIn)
			close(batchErrorIn)

			batchDataIn, batchErrorIn, batchTimeout = NewCopyBatcherBatch(&batcher.config, errorOut)
			index = 0
		}

		for {
			select {
			case <-batchTimeout.Select():
				newBatch()
			case record, open := <-dataIn:
				if !open {
					break
				}

				if batchTimeout.IsSet() {
					newBatch()
				}

				batchDataIn <- record
				index++

				if index == batcher.config.batchSize {
					newBatch()
				}
			case <-batcher.config.context.Done():
				if err := batcher.config.context.Err(); err != nil {
					batchErrorIn <- err
					errorOut <- err
					break
				}
			}
		}
	}()

	return batcher
}

func (batcher *CopyBatcher) Push(data []any) {
	batcher.PushCallback(data, nil)
}

func (batcher *CopyBatcher) PushCallback(data []any, callback func(error)) {
	batcher.dataIn <- CopyBatcherItem { data: data, callback: callback }
}

func (batcher *CopyBatcher) PushAll(data [][]any) {
	for i := range data {
		batcher.Push(data[i])
	}
}

func (batcher *CopyBatcher) PushAllCallback(data [][]any, callback func(<-chan error)) {
	errors := make(chan error, len(data))
	var wg sync.WaitGroup
	wg.Add(len(data))

	for i := range data {
		batcher.PushCallback(data[i], func(err error) {
			if err != nil {
				errors <- err
			}
			wg.Done()
		})
	}

	go func() {
		wg.Wait()
		callback(errors)
	}()
}

type CopyBatcherItem struct {
	data []any
	callback func(error)
}

type CopyBatcherBatch struct {
	dataIn <-chan CopyBatcherItem
	errorIn <-chan error
	start event.Event
	close event.Event
	timeout event.Event
	data []any
	error error
	callbacks []func(error)
	config *CopyBatcherConfig
}

func NewCopyBatcherBatch(config *CopyBatcherConfig, errorOut chan<- error) (chan<- CopyBatcherItem, chan<- error, event.Event) {
	dataIn := make(chan CopyBatcherItem)
	errorIn := make(chan error)
	timeout := event.New()

	config.db.workerPool.Submit(func() {
		batch := CopyBatcherBatch {
			dataIn: dataIn,
			errorIn: errorIn,
			config: config,
			start: event.New(),
			close: event.New(),
			timeout: timeout,
		}

		go func() {
			batch.start.Wait()
			time.AfterFunc(batch.config.batchTimeout, batch.timeout.Set)
		}()

		count, err := config.db.pool.CopyFrom(
			config.context,
			config.tableName,
			config.columns,
			&batch,
		)

		if !batch.start.IsSet() {
			CopyBatcherLoggerErrorHook(config, err)
			log.Fatalln("CopyBatcher: CopyFrom returned before any data was received")
		}

		if err != nil {
			errorOut <- err
		} else if count > 0 {
			log.Printf("Copied %d rows into table %s\n", count, config.tableName)
		}

		for _, callback := range batch.callbacks {
			callback(err)
		}
	})

	return dataIn, errorIn, timeout
}

func (batch *CopyBatcherBatch) Next() bool {
	if !batch.start.IsSet() {
		batch.start.Set()
	}

	if batch.close.IsSet() {
		return false
	}

	var open bool
	var item CopyBatcherItem
	select {
	case item, open = <-batch.dataIn:
		batch.data = item.data
		if item.callback != nil {
			batch.callbacks = append(batch.callbacks, item.callback)
		}
	case batch.error = <-batch.errorIn:
	}

	if !open {
		batch.close.Set()
		return false
	}

	if batch.error != nil {
		return false
	}

	return true
}

func (batch *CopyBatcherBatch) Values() ([]any, error) {
	return batch.data, batch.error
}

func (batch *CopyBatcherBatch) Err() error {
	return batch.error
}

func CopyBatcherLoggerErrorHook(config *CopyBatcherConfig, err error) {
	log.Printf("Error in copy channel for table %s: %s", config.tableName.Sanitize(), err)
}
