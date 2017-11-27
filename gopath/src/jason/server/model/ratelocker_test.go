package model

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU() - 1)
}
func TestRateLock(t *testing.T) {

	fn := func(id string, payload interface{}) {
		t.Log("task run:", payload, ", auto token:", id, time.Now().UnixNano())
		time.Sleep(time.Millisecond * 500)
	}
	cancel := func(id string) {
		t.Log("cancel task:", id)
	}

	t.Run("shall not accept task before start", func(t *testing.T) {

		loc := NewRateLocker(fn, cancel, false, 2, time.Second*1)
		tk, err := loc.Dispatch("A")
		if err == nil || tk != "" {
			t.Error("task accepted")
		}
	})

	// chReadyToStop := make(chan bool)

	t.Run("accept task beyond rate limit, add to queue", func(t *testing.T) {

		ctx, cancel := context.WithCancel(context.Background())

		loc := NewRateLocker(fn, cancel, false, 2, time.Second*1)
		loc.StartAsync(ctx)
		loc.Dispatch("B")
		loc.Dispatch("C")
		loc.Dispatch("D")
		loc.Dispatch("E")
		// loc.Dispatch("F", cancel)
		// loc.Dispatch("G", cancel)
		tk, err := loc.Dispatch("H")
		if err == nil || tk != "" {
			t.Log("task accepted")
		} else {
			t.Error("task not accepted")
		}
		loc.Stop()
		cancel()
	})

	t.Run("onCancel triggered if server shutdown", func(t *testing.T) {

		ctx, ctxCancel := context.WithCancel(context.Background())

		recv := make(chan string, 1)
		var idCancelTest string
		customCancel := func(id string) {
			if idCancelTest == id {
				recv <- id
				t.Log("OnCancel called")
			}
		}

		loc := NewRateLocker(func(id string, payload interface{}) {
			t.Log("task run:", payload, ", auto token:", id, time.Now().UnixNano())
			if payload.(string) == "Blocker" {
				time.Sleep(time.Second * 2)
				// t.Error("oncancel shall not be here")
			} else if payload.(string) == "OnCancel" {
				t.Error("oncancel shall not be here")
			}
		}, customCancel, false, 1, time.Second*1)
		loc.StartAsync(ctx)

		go func() {
			loc.Dispatch("2A")
			loc.Dispatch("Blocker")
			token, err := loc.Dispatch("OnCancel")
			idCancelTest = token

			if err != nil {
				t.Error("OnCancel test to started, fail to add task")
			}
			// chReadyToStop <- true
			loc.Stop()
		}()

		select {
		case <-ctx.Done():
			t.Error("OnCancel not called")
			ctxCancel()
		case id := <-recv:
			t.Log("validate that OnCancel was callled", id)
			ctxCancel()
		}
	})
	// t.Run("")
	// loc.Dispatch()
	// loc.Dispatch(fn, cancel)
	// loc.Dispatch(fn, cancel)
	// loc.Dispatch(fn, cancel)
}
