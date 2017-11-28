package model

import (
	"context"
	"errors"
	"fmt"
	"jason/server/configstore"
	"time"
)

const (
	msgErrRateLimitQueueNotRunning = "rate limit queue not runnning"
)

//pendingObject - transfer object that pass the data to RateLocker handler - lambda/block is not stored : 1. less memory ussage, 2. pendingObject can be serialized properly (if needed)
type pendingObject struct {
	id      string
	payload interface{}
	expire  int64
}

func (o *pendingObject) expired() bool {
	return o.expire > time.Now().UnixNano()
}

//RateLocker limit the amount of task executed in set interval
type RateLocker struct {
	max      int                                                       //never change
	count    int                                                       //single thread access
	interval time.Duration                                             //never change
	async    bool                                                      //shall the task be start in async way? ( async will burst in all pending requests in the begining of a interval )
	runner   func(ctx context.Context, id string, payload interface{}) //normal execution of the task, with one piece of pending data
	cancel   func(id string)                                           //task not executed due to timeout / server stopping
	queue    chan *pendingObject                                       //queue to hold payloads
	ticker   *time.Ticker                                              //time ticker to reset count
	stopping bool
	stopper  func()
}

//NewRateLocker Create new rateLocker object
func NewRateLocker(normalProcHandler func(ctx context.Context, id string, payload interface{}), onCancelHandler func(id string), async bool, rate int, unit time.Duration) *RateLocker {
	var rl RateLocker
	rl.max = rate
	rl.count = 0
	rl.interval = unit
	//mtx created
	rl.async = async
	rl.runner = normalProcHandler
	rl.cancel = onCancelHandler
	rl.ticker = nil
	//queue create at Start
	//ticker create at Start
	return &rl
}

//StartAsync - start the ratelocker
func (rl *RateLocker) StartAsync(srvCtx context.Context) {
	waitQueue := make(chan bool)
	rl.queue = make(chan *pendingObject)
	// rl.queue = make([]*pendingObject, 0)
	rl.count = 0

	rl.ticker = time.NewTicker(rl.interval)
	var ctx context.Context
	ctx, rl.stopper = context.WithCancel(srvCtx)

	var limited bool
	go func() {
		fmt.Println("rate locker start!!!!")
		//stop order : 1
	loop:
		for {
			select {
			case <-rl.ticker.C:
				// fmt.Println("rate locker ticked!", t, "total request / last period:", rl.count)
				rl.count = 0
				if limited {
					waitQueue <- true
					// fmt.Println("rate limite released")
				}
				GetRouteOwnershipStore().CleanExpired()
				getRouteRequestStore().cleanExpiredItems()
			case <-ctx.Done():
				fmt.Println("rate locker stop!!!!")
				break loop
			}
		}
		// for t := range rl.ticker.C {
		// 	fmt.Println("ticked!", t)
		// 	rl.count = 0
		// } // exits after tick.Stop()
		close(waitQueue)
		rl.stopping = true
		rl.stopper() // no side effect to call again
		fmt.Println("rate locker stoped.")
	}()

	//work queue
	go func() {
		var last *pendingObject
		var ok bool
		for {
			if ctx.Err() != nil {
				// fmt.Println("context ended")
				break
			} // was ctx ended?
			last, ok = <-rl.queue

			if !ok {
				fmt.Println("queue closed, shutting down")
				break //
			}

			if rl.count > rl.max {
				// fmt.Println("sleep triggered due to time limit reached")
				limited = true
				_, ok = <-waitQueue
				// fmt.Println("wait was releaseed", ok)
				if !ok {
					fmt.Println("queue closed, shutting down")
					break //
				}
			}

			if ctx.Err() != nil {
				fmt.Println("context ended, after sleep")
				break
			} // was ctx ended?

			//good to go
			// context.WithTimeout(ctx, time.Sec)
			rl.runner(ctx, last.id, last.payload)
			last = nil
		}

		if last != nil {
			rl.cancel(last.id)
		}
		rl.stopper() // no side effect to call again
	}()
	rl.stopping = false
}

//IsRunning - is the locker running?
func (rl *RateLocker) IsRunning() bool {
	return !rl.stopping && rl.queue != nil //not stopped and started
}

//Stop - stop the ratelocker
func (rl *RateLocker) Stop() {
	// fmt.Println("Stop Locker")
	rl.stopping = true // prevent accept new task when stopping
	if rl.ticker != nil {
		// fmt.Println("Stop Ticker")
		rl.ticker.Stop()
	}
	if rl.stopper != nil {
		rl.stopper()
	}
}

//Dispatch - dispatch a task with rate limit
func (rl *RateLocker) Dispatch(payload interface{}) (token string, expire int64, err error) {
	if !rl.IsRunning() {
		return "", 0, errors.New(msgErrRateLimitQueueNotRunning)
	}
	token = newToken()
	expire = time.Now().Add(configstore.RecordExpireInSeconds * time.Second).UnixNano()
	rl.queue <- &pendingObject{
		token,
		payload,
		expire,
	}
	return token, expire, nil
}
