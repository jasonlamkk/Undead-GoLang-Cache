package model

import (
	"errors"

	"github.com/golang/groupcache"
	"github.com/satori/go.uuid"
)

const (
	groupRoutes          = "routes"
	groupCachePortSuffix = ":8012"

	msgErrCommonInvalidInput   = "invalid input"
	msgErrCommonInvalidContent = "invalid content"
)

func newToken() string {
	return uuid.NewV4().String()
}

var (
	rateLimiter RateLocker
	group       groupcache.Group
	gcPool      *groupcache.HTTPPool
)

//StartModelCluster join the cluster and start background worker for remote api tasks
// * only groupcache is used in this POC to similify the setup
func StartModelCluster(myIp string) {
}

//StopModelCluster stop leave the cluster and background worker
func StopModelCluster() {

}

//RegisterRouteRequestAsync Quickly add Request into queue
// return error if fail
func RegisterRouteRequestAsync(input [][]string) (token string, err error) {
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
	err = nil
	return
}
