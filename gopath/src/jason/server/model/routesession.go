package model

import (
	"context"
	"errors"
	"jason/server/configstore"
	"time"

	"github.com/satori/go.uuid"
)

const (
	msgErrCommonInvalidInput   = "invalid input"
	msgErrCommonInvalidContent = "invalid content"
)

func newToken() string {
	return uuid.NewV4().String()
}

var (
	rateLimiterForRoute *RateLocker
)

//StartBgTask start rate limiter
func StartBgTask(srvCtx context.Context) {
	StopBgTask()
	rateLimiterForRoute = NewRateLocker(processRouteRequest, cancelRouteRequest, false, 50, time.Minute)
	rateLimiterForRoute.StartAsync(srvCtx)
}

//StopBgTask stop leave the cluster and background worker
func StopBgTask() {
	if rateLimiterForRoute != nil {
		rateLimiterForRoute.Stop()
	}
}

//RegisterRouteRequestAsync Quickly add Request into queue
// return error if fail
func RegisterRouteRequestAsync(input [][]string) (token string, expire int64, err error) {
	if len(input) < 2 {
		err = errors.New(msgErrCommonInvalidInput)
		return
	}
	//validate all values
	for _, v := range input {
		if len(v) != 2 {
			err = errors.New(msgErrCommonInvalidInput)
			return
		}
	}

	token = newToken()
	expire = time.Now().Add(configstore.RecordExpireInSeconds * time.Second).UnixNano()
	err = nil
	go getRouteRequestStore().putRequest(token, expire, input)
	return
}

//GetRouteByToken return if the token is know and which server was it stored.
// return status by bool instead of enum to directly reflects if-else logics in next steps
func GetRouteByToken(token string) (result []byte, remoteAddr string, existsHere, ready, atPeer bool) {
	//check local first
	result, existsHere, ready = getRouteRequestStore().getLocaleRouteByToken(token)
	if !existsHere {
		//check if cluster know who has it
		remoteAddr, atPeer = GetRouteOwnershipStore().QueryRouteOwnership(token)
		if atPeer {
			return nil, remoteAddr, false, false, true
		}
		return nil, "", false, false, false
	}

	return result, "", existsHere, ready, false
}

//GetExportResult export result for rebalance before node die
func GetExportResult() (map[string][]byte, map[int64][]string) {
	return getRouteRequestStore().exportResults()
}
