package model

import (
	"context"
	"errors"
	"fmt"
	"sync"
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
	max      int                                  //never change
	count    int                                  //single thread access
	interval time.Duration                        //never change
	mtx      sync.RWMutex                         //guard queue
	async    bool                                 //shall the task be start in async way? ( async will burst in all pending requests in the begining of a interval )
	runner   func(id string, payload interface{}) //normal execution of the task, with one piece of pending data
	cancel   func(id string)                      //task not executed due to timeout / server stopping
	queue    []*pendingObject                     //queue to hold payloads
	ticker   *time.Ticker                         //time ticker to reset count
	stopping bool
	stopper  func()
}

//NewRateLocker Create new rateLocker object
func NewRateLocker(normalProcHandler func(id string, payload interface{}), onCancelHandler func(id string), async bool, rate int, unit time.Duration) *RateLocker {
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
	rl.queue = make([]*pendingObject, 0)
	rl.count = 0

	rl.ticker = time.NewTicker(rl.interval)
	var ctx context.Context
	ctx, rl.stopper = context.WithCancel(srvCtx)

	go func() {
		fmt.Println("start!!!!")
		//stop order : 1
	loop:
		for {
			select {
			case t := <-rl.ticker.C:
				fmt.Println("rate locker ticked!", t, "total request / last period:", rl.count)
				rl.count = 0
				go getRouteRequestStore().cleanExpiredItems()
			case <-ctx.Done():
				fmt.Println("rate locker stop!!!!")
				break loop
			}
		}
		// for t := range rl.ticker.C {
		// 	fmt.Println("ticked!", t)
		// 	rl.count = 0
		// } // exits after tick.Stop()
		fmt.Println("stoped....")
		rl.stopping = true
		rl.stopper() // no side effect to call again
	}()

	go func() {
		//stop order : 2
		for {
			if ctx.Err() != nil {
				fmt.Println("context ended")
				break
			} // was ctx ended?

			if rl.count > rl.max {
				fmt.Println("sleep triggered due to time limit reached")
				time.Sleep(rl.interval / 50) // wait if limit reached
			}

			if ctx.Err() != nil {
				fmt.Println("context ended, after sleep")
				break
			} // was ctx ended?

			var o *pendingObject
			rl.mtx.RLock()
			if len(rl.queue) > 0 {
				o = rl.queue[0]
				rl.queue = rl.queue[1:]
			}
			rl.mtx.RUnlock()

			if o != nil {
				if o.expired() {
					rl.cancel(o.id)
					continue
				} else {
					rl.count++ //count is only accessed on single thread
				}
				if rl.async {
					fmt.Println("task start async", o.id, o.payload)
					go rl.runner(o.id, o.payload)
				} else {
					fmt.Println("task start sync", o.id, o.payload)
					rl.runner(o.id, o.payload)
					fmt.Println("task sync finish")
				}
			}
		}
		//ctx ended
		for _, o := range rl.queue {
			rl.cancel(o.id)
		}
		rl.queue = nil //stop accept new tasks
		rl.ticker = nil
		if rl.stopper != nil {
			rl.stopper() // no side effect to call again
		}
	}()
	rl.stopping = false
}

//IsRunning - is the locker running?
func (rl *RateLocker) IsRunning() bool {
	return !rl.stopping && rl.queue != nil
}

//Stop - stop the ratelocker
func (rl *RateLocker) Stop() {
	fmt.Println("Stop Locker")
	rl.stopping = true // prevent accept new task when stopping
	if rl.ticker != nil {
		fmt.Println("Stop Ticker")
		rl.ticker.Stop()
	}
	if rl.stopper != nil {
		rl.stopper()
	}
}

//Dispatch - dispatch a task with rate limit
func (rl *RateLocker) Dispatch(payload interface{}) (token string, err error) {
	if !rl.IsRunning() {
		return "", errors.New(msgErrRateLimitQueueNotRunning)
	}
	token = newToken()
	po := pendingObject{
		token,
		payload,
		time.Now().Add(time.Second * 30).UnixNano(),
	}
	rl.mtx.Lock()
	rl.queue = append(rl.queue, &po)
	rl.mtx.Unlock()

	return token, nil
}
