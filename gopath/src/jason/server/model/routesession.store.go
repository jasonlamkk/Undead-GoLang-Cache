package model

import (
	"fmt"
	"jason/server/network"
	"sync"
	"time"

	"jason/server/configstore"
)

const (
	msgErrPending       = "pending"
	msgErrNoSuchRequest = "no such request"
)

type routeRequestStore struct {
	pMutex        sync.RWMutex
	pending       map[string][][]string
	pendingExpire map[int64][]string
	rMutex        sync.RWMutex
	result        map[string][]byte
	resultExpire  map[int64][]string
}

var initOnceRouteRequestStore sync.Once
var sharedRouteRequestStore *routeRequestStore

//getRouteRequestStore get / create singleton store
func getRouteRequestStore() *routeRequestStore {
	initOnceRouteRequestStore.Do(func() {
		sharedRouteRequestStore = &routeRequestStore{
			pending:       make(map[string][][]string),
			pendingExpire: make(map[int64][]string),
			result:        make(map[string][]byte),
			resultExpire:  make(map[int64][]string),
		}
		// //
		// sharedRouteRequestStore.fnGetResult = func(gc groupcache.Context, key string, sk groupcache.Sink) error {
		// 	sharedRouteRequestStore.rMutex.RLock()
		// 	r, ok := sharedRouteRequestStore.result[key]
		// 	sharedRouteRequestStore.rMutex.RUnlock()

		// 	if !ok {
		// 		return errors.New(msgErrPending)
		// 	}
		// 	sk.SetString(r)
		// 	return nil
		// }

		// sharedRouteRequestStore.fnWasRequested = func(gc groupcache.Context, key string, sk groupcache.Sink) error {
		// 	sharedRouteRequestStore.pMutex.RLock()
		// 	_, ok := sharedRouteRequestStore.pending[key]
		// 	sharedRouteRequestStore.pMutex.RUnlock()
		// 	if !ok {
		// 		return errors.New(msgErrNoSuchRequest)
		// 	}
		// 	sk.SetBytes([]byte{1})
		// 	return nil
		// }

	})
	return sharedRouteRequestStore
}

func (s *routeRequestStore) getLocaleRouteByToken(token string) (r []byte, exists, ready bool) {
	//local
	s.rMutex.RLock()
	tmp, ok := s.result[token]
	s.rMutex.RUnlock()

	if ok {
		return tmp, true, true
	}

	s.pMutex.RLock()
	_, ok = s.pending[token]
	s.pMutex.RUnlock()

	if ok {
		return nil, true, false
	}

	return nil, false, false
}

//putRequest store request in cache and database
func (s *routeRequestStore) putRequest(key string, expire int64, data [][]string) error {
	expire = time.Now().Add(configstore.PendingExpireInSeconds * time.Second).UnixNano()
	s.pMutex.Lock()
	s.pending[key] = data
	s.pendingExpire[expire] = append(s.pendingExpire[expire], key)
	s.pMutex.Unlock()
	logRequest(key, data)
	return nil
}

//logRequest - dummy function which presist data into database - database (logging) layer not implemented to remove dependency and simplifed setup
func logRequest(key string, data [][]string) {
	fmt.Println("request logged: ", key, data)

}

func (s *routeRequestStore) injectResult(key string, output []byte, expire int64) {
	s.rMutex.Lock()
	s.result[key] = output
	s.resultExpire[expire] = append(s.resultExpire[expire], key)
	s.rMutex.Unlock()
	s.pMutex.Lock()
	delete(s.pending, key)
	s.pMutex.Unlock()
}

//putRequest store request in cache and database
func (s *routeRequestStore) putResult(key string, output []byte) (expire int64, err error) {
	expire = time.Now().Add(configstore.RecordExpireInSeconds * time.Second).UnixNano()
	s.rMutex.Lock()
	s.result[key] = output
	s.resultExpire[expire] = append(s.resultExpire[expire], key)
	s.rMutex.Unlock()
	s.pMutex.Lock()
	delete(s.pending, key)
	s.pMutex.Unlock()

	go logResult(key, output) //async
	return 0, nil
}

func (s *routeRequestStore) cleanExpiredItems() {

	var ct = time.Now().UnixNano()

	s.rMutex.Lock()
	for xt, tks := range s.resultExpire {
		if xt > ct {
			break
		}
		if tks != nil {
			for _, tk := range tks {
				delete(s.result, tk)
			}
			delete(s.resultExpire, xt)
		}
	}
	s.rMutex.Unlock()

	s.pMutex.Lock()
	for xt, tks := range s.pendingExpire {
		if xt > ct {
			break
		}
		if tks != nil {
			for _, tk := range tks {
				delete(s.pending, tk)
			}
			delete(s.pendingExpire, xt)
		}
	}
	s.pMutex.Unlock()

}

//logResult - dummy function which presist data into database - database (logging) layer not implemented to remove dependency and simplifed setup
func logResult(key string, result []byte) {
	fmt.Println("result logged: ", key, string(result))
}

func (s *routeRequestStore) exportResults() (map[string][]byte, map[int64][]string) {
	defer s.rMutex.RUnlock()
	s.rMutex.RLock()
	return s.result, s.resultExpire
}

func GetRouteRequestStoreForInject() network.KeyValueInjecter {
	return getRouteRequestStore()
}

//implement network.KeyValueInjector
func (s *routeRequestStore) InjectResult(key string, output []byte, expire int64) {

	s.rMutex.Lock()
	s.result[key] = output
	s.resultExpire[expire] = append(s.resultExpire[expire], key)
	s.rMutex.Unlock()

	s.pMutex.Lock()
	delete(s.pending, key)
	s.pMutex.Unlock()
}
