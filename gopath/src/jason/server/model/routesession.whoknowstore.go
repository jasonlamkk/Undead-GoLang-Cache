package model

import "time"

//RouteOwnershipStore store which server know which token
type RouteOwnershipStore struct {
	whoKnows   map[ /*token*/ string] /*addr*/ string //store data include self data for easier clone to other new join machine
	whenExpire map[int64][] /*token*/ string
}

var sharedRouteOwnershipStore = RouteOwnershipStore{
	whoKnows:   make(map[string]string),
	whenExpire: make(map[int64][]string),
}

//GetRouteOwnershipStore get singleton instance of RouteOwnershipStore
func GetRouteOwnershipStore() *RouteOwnershipStore {
	return &sharedRouteOwnershipStore
}

//RemoveToken remove a token; usually used when remote server give up processing a record
func (s *RouteOwnershipStore) RemoveToken(tk string) {
	delete(s.whoKnows, tk)
}

//QueryRouteOwnership let server ask
func (s *RouteOwnershipStore) QueryRouteOwnership(token string) (addr string, found bool) {
	//read only no modification, no mutex lock
	addr, found = s.whoKnows[token]
	return
}

//AddRouteOwnership get
func (s *RouteOwnershipStore) AddRouteOwnership(addr, token string, expire int64) {
	s.whenExpire[expire] = append(s.whenExpire[expire], token)
	s.whoKnows[token] = addr
}

//
func (s *RouteOwnershipStore) ExportTokenOwners() (map[ /*token*/ string] /*addr*/ string, map[int64][] /*token*/ string) {
	return s.whoKnows, s.whenExpire
}

//CleanExpired clear records that expired
func (s *RouteOwnershipStore) CleanExpired() {

	var ct = time.Now().UnixNano()
	for xt, tks := range s.whenExpire {
		if ct > xt {
			return //all handled
		}
		if tks != nil {
			for _, tk := range tks {
				delete(s.whoKnows, tk)
			}
		}
		delete(s.whenExpire, xt)
	}
}
