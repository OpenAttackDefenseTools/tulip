package db

import (
	"context"
	"log"
	"sync"
	"time"

	"go-importer/internal/pkg/event"

	"github.com/jackc/pgx/v5"
)

const CHANNEL_DEFAULT_BATCH_SIZE = 1000;
const CHANNEL_DEFAULT_BATCH_TIMEOUT = time.Duration(5 * float64(time.Second));

type CopyChannelContext struct {
	db *Database
	dbCtx context.Context
	table_name pgx.Identifier
	columns []string
	batchSize int
	batchTimeout time.Duration
	errorHook func(*CopyChannelContext, error)
}

type CopyChannelPool struct {
	dataIn chan<- CopyChannelItem
	ctx CopyChannelContext
}

func NewCopyChannelPool(ctx CopyChannelContext) CopyChannelPool {
	dataIn := make(chan CopyChannelItem)
	errorOut := make(chan error)

	if ctx.batchSize == 0 {
		ctx.batchSize = CHANNEL_DEFAULT_BATCH_SIZE;
	}

	if ctx.batchTimeout == 0 {
		ctx.batchTimeout = CHANNEL_DEFAULT_BATCH_TIMEOUT
	}

	if ctx.dbCtx == nil {
		ctx.dbCtx = context.Background()
	}

	if ctx.errorHook == nil {
		ctx.errorHook = CopyChannelLoggerErrorHook
	}

	pool := CopyChannelPool {
		dataIn: dataIn,
		ctx: ctx,
	}

	go func() {
		for {
			err, open := <-errorOut
			if err != nil {
				ctx.errorHook(&pool.ctx, err)
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
		batchDataIn, batchErrorIn, batchTimeout := NewCopyChannelBatch(&pool.ctx, errorOut)

		newBatch := func() {
			close(batchDataIn)
			close(batchErrorIn)

			batchDataIn, batchErrorIn, batchTimeout = NewCopyChannelBatch(&pool.ctx, errorOut)
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

				if index == pool.ctx.batchSize {
					newBatch()
				}
			case <-pool.ctx.dbCtx.Done():
				if err := pool.ctx.dbCtx.Err(); err != nil {
					batchErrorIn <- err
					errorOut <- err
					break
				}
			}
		}
	}()

	return pool
}

func (pool *CopyChannelPool) Push(data []any) {
	pool.PushCallback(data, nil)
}

func (pool *CopyChannelPool) PushCallback(data []any, callback func(error)) {
	pool.dataIn <- CopyChannelItem { data: data, callback: callback }
}

func (pool *CopyChannelPool) PushAll(data [][]any) {
	for i := range data {
		pool.Push(data[i])
	}
}

func (pool *CopyChannelPool) PushAllCallback(data [][]any, callback func(<-chan error)) {
	errors := make(chan error, len(data))
	var wg sync.WaitGroup
	wg.Add(len(data))

	for i := range data {
		pool.PushCallback(data[i], func(err error) {
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

type CopyChannelItem struct {
	data []any
	callback func(error)
}

type CopyChannelBatch struct {
	dataIn <-chan CopyChannelItem
	errorIn <-chan error
	start event.Event
	close event.Event
	timeout event.Event
	data []any
	error error
	callbacks []func(error)
	ctx *CopyChannelContext
}

func NewCopyChannelBatch(ctx *CopyChannelContext, errorOut chan<- error) (chan<- CopyChannelItem, chan<- error, event.Event) {
	dataIn := make(chan CopyChannelItem)
	errorIn := make(chan error)
	timeout := event.New()

	ctx.db.workerPool.Submit(func() {
		batch := CopyChannelBatch {
			dataIn: dataIn,
			errorIn: errorIn,
			ctx: ctx,
			start: event.New(),
			close: event.New(),
			timeout: timeout,
		}

		go func() {
			batch.start.Wait()
			time.AfterFunc(batch.ctx.batchTimeout, batch.timeout.Set)
		}()

		_, err := ctx.db.pool.CopyFrom(
			ctx.dbCtx,
			ctx.table_name,
			ctx.columns,
			&batch,
		)

		if !batch.start.IsSet() {
			CopyChannelLoggerErrorHook(ctx, err)
			log.Fatalln("CopyChannel: CopyFrom returned before any data was received")
		}

		if err != nil {
			errorOut <- err
		}

		for _, callback := range batch.callbacks {
			callback(err)
		}
	})

	return dataIn, errorIn, timeout
}

func (batch *CopyChannelBatch) Next() bool {
	if !batch.start.IsSet() {
		batch.start.Set()
	}

	if batch.close.IsSet() {
		return false
	}

	var open bool
	var item CopyChannelItem
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

func (batch *CopyChannelBatch) Values() ([]any, error) {
	return batch.data, batch.error
}

func (batch *CopyChannelBatch) Err() error {
	return batch.error
}

func CopyChannelLoggerErrorHook(ctx *CopyChannelContext, err error) {
	log.Printf("Error in copy channel for table %s: %s", ctx.table_name.Sanitize(), err)
}
