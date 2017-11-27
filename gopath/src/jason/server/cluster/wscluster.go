package cluster

//a small websocket client that talk between server nodes, same data center
//although there are general purpose db cluster / distributed key-value store, it is faster when code is simple; so I build the basic cluster as follow

import (
	"bytes"
	stdCtx "context"
	"encoding/binary"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	wsc "github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/websocket"

	"jason/server/configstore"
	"jason/server/model"
	"jason/server/network"
)

const (
	ePeerStatusKnownOffline = iota
	ePeerStatusOutgoing     = iota
	ePeerStatusIncoming     = iota

	intervalClearUpExpired = time.Second * 60 * 5

	lenToken = 16 //8+1+4+1+4+1+4+1+12
	lenInt64 = 8

	headerClusterProtoVersion    = "X-Cluster-Proto-Version"
	systemClusterProtocolVersion = "v1"

	strRoomCluster = "cluster"

	//RouteClusterSocket is a fixed route for websocket path
	RouteClusterSocket = "/ws_cluster"

	// headerMyHTTPAddress = "X-HTTP-Address"
	simpleMessageTypeAddr        = "ip"
	simpleMessageTypePeer        = "pe"
	simpleMessageTypeReq         = "rq"
	simpleMessageTypeFwdReq      = "fw"
	simpleMessageTypeKillPending = "kp"
	simpleMessageTypeRebalance   = "rb"
)

type ePeerStatus int

// type network.SendCloser interface {
// 	WriteMessage(int, []byte) error
// 	Close()
// }

type peer struct {
	status  ePeerStatus
	srcAddr string
	conn    network.SendCloser
}

type tmpRebalanceRoute struct {
	token  string
	expire int64
	data   []byte
}

// type whoKnowRequest struct {
// 	addr   string
// 	expire int64
// }

var (
	wsInitOnce sync.Once
	ws         *websocket.Server
	connMutex  sync.RWMutex
	knownHosts map[ /*ip*/ string]*peer
	// client map[ /*ip*/ string]websocket.Connection

	rebalanceInjector network.KeyValueInjecter
)

func init() {
	knownHosts = make(map[string]*peer)
}

// func handleInit(c websocket.Connection, ip []byte) {
// 	var ipStr = string(bytes.TrimSpace(ip))

// 	knownHosts[ipStr] = &peer{
// 		status:  ePeerStatusIncoming,
// 		srcAddr: ipStr,
// 		closer:  c,
// 	}

// 	c.To(websocket.All).EmitMessage([]byte(simpleMessageTypeServer + ipStr))
// }

func addPeer(isIncoming bool, addr string, conn network.SendCloser) {
	fmt.Println("register peer", isIncoming, addr)
	var p = peer{}
	if isIncoming {
		p.status = ePeerStatusIncoming
	} else {
		p.status = ePeerStatusOutgoing
	}
	p.srcAddr = addr
	p.conn = conn

	connMutex.Lock()
	old, exists := knownHosts[addr]
	knownHosts[addr] = &p
	connMutex.Unlock()

	if exists && old != nil && old.conn != nil {
		old.conn.Close()
	}
}

//disconnect  and don't keep history, another possible way it set status = ePeerStatusKnownOffline
// but thinking that address may change and retrying is not really necessary as new peer can actively join cluster , all info deleted after disconnected
func disconnectPeer(addr string) {

	connMutex.Lock()
	old, exists := knownHosts[addr]
	delete(knownHosts, addr)
	connMutex.Unlock()

	if exists && old != nil && old.conn != nil {
		old.conn.Close()
	}
}

func handleRemoteKillPending(message []byte) {
	if len(message) != lenToken {
		return
	}
	u, err := uuid.FromBytes(message[:lenToken])
	if err != nil {
		return
	}

	model.GetRouteOwnershipStore().RemoveToken(u.String())
}

//SetRebalanceInjector set a callback which allow passing data store when rebalance message received
func SetRebalanceInjector(ijt network.KeyValueInjecter) {
	rebalanceInjector = ijt
}

//used when remote server not just disconnect, but it will lost in memory records
func handleRemoteRebalance(message []byte) {
	if rebalanceInjector == nil {
		fmt.Println("handleRemoteRebalance: skipped, rebalanceInjector not available.")
		return
	}
	if !(len(message) > (lenInt64 + lenToken)) {
		return
	}
	var expire = int64(binary.LittleEndian.Uint64(message[:lenInt64]))
	u, err := uuid.FromBytes(message[lenInt64 : lenInt64+lenToken])
	if err != nil {
		return
	}
	result := message[lenInt64+lenToken:]
	rebalanceInjector.InjectResult(u.String(), result, expire)
}

//PublishRouteOwnership Publish Route Ownership and expire time
func PublishRouteOwnership(token string, expire int64) {
	//add local - allow clone to new joiner
	model.GetRouteOwnershipStore().AddRouteOwnership(configstore.GetAddressStore().GetServerAddress(), token, expire)
	//local okay

	msg, err := serializeKnownRequest(token, expire)
	if err != nil {
		fmt.Println("Error format message", err)
		return
	}

	fmt.Println("Publishing", token, expire, msg)

	var count = 0
	var tmp []*peer
	connMutex.RLock()
	tmp = make([]*peer, len(knownHosts))
	for _, p := range knownHosts {
		tmp[count] = p
		count++
	}
	connMutex.RUnlock()

	fmt.Println("to hosts", tmp)

	for _, p := range tmp {
		if p != nil && p.conn != nil {
			err = p.conn.WriteMessage(wsc.TextMessage, msg)
			if err != nil {
				p.conn.Close()
			}
		}
	}
}

//PublishRebalance is used when main server is dying; it forward data on hand to other peers
func PublishRebalance(gracefulCtx stdCtx.Context) {
	var peers = []string{}
	connMutex.RLock()
	for k, c := range knownHosts {
		if c.status == ePeerStatusKnownOffline {
			continue
		}
		peers = append(peers, k)
	}
	connMutex.RUnlock()

	if len(peers) < 1 {
		fmt.Println("[CRIT]", "all nodes down", "rebalance skipped")
		return
	}

	var maxIdx = len(peers) - 1

	var failConns = map[int]string{}
	var sIdx = 0
	var failTmps = []tmpRebalanceRoute{}

	var ct = time.Now().UnixNano()
	datas, expires := model.GetExportResult()
	//fast loop - which shall forward all records in the fastest manner ; while fails catch up in a slower but safer logic
	var err error
	for xt, tks := range expires {
		if xt < ct {
			continue
		} //ignore expired
		for _, tk := range tks {

			if gracefulCtx.Err() != nil {
				fmt.Println("[TimesUp]", "graceful shutdown timeout", "rebalance stopped")
				return //
			}
			if data, ok := datas[tk]; ok {
				sIdx = (sIdx + 1) % maxIdx
				var tmp = tmpRebalanceRoute{tk, xt, data}

				if _, dontuse := failConns[sIdx]; dontuse {
					failTmps = append(failTmps, tmp)

					if len(failConns) > maxIdx {
						fmt.Println("[CRIT]", "all nodes down", "rebalance stopped", len(failConns), "/", maxIdx)
						return
					}
					continue //skip to next
				}

				if host, found := knownHosts[peers[sIdx]]; found && host.conn != nil {
					err = sendRebalanceRecord(host.conn, &tmp)
					if err == nil {
						continue //done, go next
					}
					failConns[sIdx] = peers[sIdx]
				}
				failTmps = append(failTmps, tmp)
			}
		}
	}
	if len(failTmps) < 1 {
		fmt.Println("[Nice]", "graceful shutdown server", configstore.GetAddressStore().GetServerAddress())
		return
	}
	//
	//slow logic - with mutex lock
	//
	var removes = make([]int, len(failConns)) //a reversed slice
	{
		var idx = len(failConns) - 1
		for i := range failConns {
			removes[idx] = i
			idx--
			if idx < 0 {
				break
			}
		}
	}

	for _, idx := range removes {
		peers = append(peers[:idx], peers[idx+1:]...)
	}
outter:
	for _, tmp := range failTmps {

		if gracefulCtx.Err() != nil {
			fmt.Println("[TimesUp]", "graceful shutdown timeout", "rebalance stopped")
			return //
		}

		for _, p := range peers {
			var conn network.SendCloser
			connMutex.RLock()
			if host, ok := knownHosts[p]; ok && host.status != ePeerStatusKnownOffline {
				conn = host.conn
			}
			connMutex.RUnlock()

			if conn == nil {
				continue
			}

			err = sendRebalanceRecord(conn, &tmp)
			if err == nil {
				continue outter //expected
			}
		}
		//
		// all connection failed
		fmt.Println("[CRIT]", "all nodes down", "rebalance slow logic failed")
		break
	}
}

func checkPeerExists(addr string) (exists bool) {
	connMutex.RLock()
	_, exists = knownHosts[addr]
	connMutex.RUnlock()
	return exists
}

func publishMyPeer(c network.SendCloser) {
	var addrs = []string{}
	connMutex.RLock()
	for addr, host := range knownHosts {
		if host.conn != nil && host.status != ePeerStatusKnownOffline {
			addrs = append(addrs, addr)
		}
	}
	connMutex.RUnlock()
	for _, a := range addrs {
		c.WriteMessage(wsc.TextMessage, append([]byte(simpleMessageTypePeer), []byte(a)...))
	}
}

// core handler 1
func handlePeersPeer(subCtx stdCtx.Context, peerAddr []byte) {
	var addrStr = string(bytes.TrimSpace(peerAddr))
	if checkPeerExists(addrStr) {
		return
	}

	go tryConnect(subCtx, addrStr)
}

func serializeKnownRequest(token string, expire int64) (buf []byte, err error) {
	var uid uuid.UUID
	uid, err = uuid.FromString(token)
	if err != nil {
		return nil, err
	}
	buf = make([]byte, 2+lenInt64+lenToken)
	copy(buf[0:2], []byte(simpleMessageTypeReq))
	binary.LittleEndian.PutUint64(buf[2:2+lenInt64], uint64(expire))
	fmt.Println(">>>", uid.Bytes())
	copy(buf[2+lenInt64:2+lenInt64+lenToken], uid.Bytes())
	return buf, nil
}

func parseKnowRequest(payload []byte) (token string, expire int64, ok bool) {
	ok = len(payload) == (lenInt64 + lenToken)
	// fmt.Println("length ok?", ok, len(payload))
	if !ok {
		return token, expire, ok
	}
	u, err := uuid.FromBytes(payload[lenInt64:])
	// fmt.Println("UUID", u, err)
	ok = err == nil
	if !ok {
		return token, expire, ok
	}
	expire = int64(binary.LittleEndian.Uint64(payload[:lenInt64]))
	token = u.String()
	ok = true
	return token, expire, ok
}

// func serializeInit(myIP string) []byte {
// 	return []byte(simpleMessageTypeInit + myIP)
// }

// core handler 2
func handleWhoKnowWhich(from string, payload []byte) {

	token, expire, ok := parseKnowRequest(payload)

	// fmt.Println("handle who know:", token, expire, ok, (payload))

	if !ok {
		return //ignore
	}
	//no need to lock

	model.GetRouteOwnershipStore().AddRouteOwnership(from, token, expire)

	fmt.Println(">>>", from, token, expire)
}

func tryConnect(ctx stdCtx.Context, peerAddr string) error {
	myself := []byte(configstore.GetAddressStore().GetServerAddress())
	url := configstore.WebSocketPrefix + peerAddr + RouteClusterSocket
	// fmt.Println("try join peer:", url)
	c, _, err := wsc.DefaultDialer.Dial(url, map[string][]string{headerClusterProtoVersion: []string{systemClusterProtocolVersion}})

	if err != nil {
		fmt.Println("[ERROR]", err)
		return err
	}
	//  else {
	fmt.Println("Peer connected", url)
	// }
	go startSocket(ctx, c, myself, false)
	return nil
}

// func wsIncoming(w http.ResponseWriter, r *http.Request) {
// 	c, err := upgrader.Upgrade(w, r, nil)
// 	if err != nil {
// 		log.Print("upgrade:", err)
// 		return
// 	}
// 	defer c.Close()
// 	for {
// 		mt, message, err := c.ReadMessage()
// 		if err != nil {
// 			log.Println("read:", err)
// 			break
// 		}
// 		log.Printf(handleWhoKnowWhich"recv: %s", message)
// 		err = c.WriteMessage(mt, message)
// 		if err != nil {
// 			log.Println("write:", err)
// 			break
// 		}
// 	}
// }

func handleIncomingTextMessage(ctx stdCtx.Context, from string, message []byte) {
	switch string(string(message[:2])) {
	case simpleMessageTypePeer:
		handlePeersPeer(ctx, message[2:])
	case simpleMessageTypeReq:
		handleWhoKnowWhich(from, message[2:])
	case simpleMessageTypeFwdReq:
		if len(message) > 2+lenInt64+lenToken {
			var fwdHostAddr = string(message[2+lenInt64+lenToken:])
			handleWhoKnowWhich(fwdHostAddr, message[2:2+lenInt64+lenToken])
		}
	case simpleMessageTypeKillPending:

	}
}

func forwardRouteOwnership(c *wsc.Conn, addr, token string, expire int64) error {
	buf, err := serializeKnownRequest(token, expire)
	if err != nil {
		panic(err) //assert never happen ; if format error , then always error and break connection
		return err //
	}

	buf = append(buf, []byte(addr)...)
	copy(buf[0:2], []byte(simpleMessageTypeFwdReq)) //change type to forward
	err = c.WriteMessage(wsc.TextMessage, buf)
	return err
}

func sendRebalanceRecord(c network.SendCloser, r *tmpRebalanceRoute) error {
	uid, err := uuid.FromString(r.token)
	if err != nil {
		return err
	}

	buf := make([]byte, 2+lenInt64+lenToken+len(r.data))
	copy(buf[:2], []byte(simpleMessageTypeRebalance))
	binary.LittleEndian.PutUint64(buf[2:2+lenInt64], uint64(r.expire))
	copy(buf[2+lenInt64:2+lenInt64+lenToken], uid.Bytes())
	copy(buf[2+lenInt64+lenToken:], r.data)

	err = c.WriteMessage(wsc.TextMessage, buf)
	if err != nil {
		return err
	}
	return nil
}

func welcomeNewJoiner(c *wsc.Conn) {
	whoKnows, whenExpire := model.GetRouteOwnershipStore().ExportTokenOwners()

	if whoKnows != nil && len(whoKnows) > 0 && whenExpire != nil && len(whenExpire) > 0 {
		//clone data
		var ct = time.Now().Add(time.Second).UnixNano() //ignore expiring data
		for xt, tks := range whenExpire {
			if xt < ct {
				continue //ignore
			}
			for _, tk := range tks {
				if addr, ok := whoKnows[tk]; ok {
					if err := forwardRouteOwnership(c, addr, tk, xt); err != nil {
						fmt.Println("ERROR", "forwardRouteOwnership", err)
						return //only err is connection already closed, skip other messages
					}
					fmt.Println("ownership sent", addr, tk, xt)
				}
			}
		}
	}
}

func sendMyAddress(c network.SendCloser, myHTTPAddress []byte) {
	buf := append([]byte(simpleMessageTypeAddr), myHTTPAddress...)
	c.WriteMessage(wsc.TextMessage, buf)
}

//startSocket !! blocking !!
func startSocket(ctx stdCtx.Context, c *wsc.Conn, myHTTPAddress []byte, isIncoming bool) {
	myCtx, cancelCtx := stdCtx.WithCancel(ctx)

	go func() {
		<-myCtx.Done()
		fmt.Println("!! Socket closing", c.RemoteAddr(), "<<<", c.LocalAddr(), "---", string(myHTTPAddress))
		if ctx.Err() != nil {
			//whole server stopping

			//all socket receive context message, no need to loop

			time.Sleep(time.Millisecond * 500)
			// c.WriteMessage(wsc.TextMessage, []byte(simpleMessageTypeDying))

		}
		fmt.Println("p", ctx.Err(), "sub", myCtx.Err())
		c.Close()
		cancelCtx()
		fmt.Println("!! Socket closed")
	}()

	var remoteAddr string
	var inited = false

	go sendMyAddress(c, myHTTPAddress)

	go welcomeNewJoiner(c)

	go publishMyPeer(c)

loop:
	for {
		if myCtx.Err() != nil {
			break
		}
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("Connection Closed:", err)
			break
		}
		if mt != wsc.TextMessage {
			continue
		}
		if len(message) < 2 {
			continue
		}

		if myCtx.Err() != nil {
			break
		}

		switch string(message[:2]) {
		case simpleMessageTypeAddr:
			remoteAddr = string(message[2:])
			if !inited {
				inited = true
				addPeer(isIncoming, remoteAddr, c)
			}
		case simpleMessageTypePeer:
			go handlePeersPeer(ctx, message[2:])
		case simpleMessageTypeReq:
			if !inited {
				continue loop
			}
			go handleWhoKnowWhich(remoteAddr, message[2:])
		case simpleMessageTypeFwdReq:
			if len(message) > 2+lenInt64+lenToken {
				var fwdHostAddr = string(message[2+lenInt64+lenToken:])
				go handleWhoKnowWhich(fwdHostAddr, message[2:2+lenInt64+lenToken])
			}
		case simpleMessageTypeKillPending:
			go handleRemoteKillPending(message[2:])
		case simpleMessageTypeRebalance:
			go handleRemoteRebalance(message[2:])
		}
	}

	disconnectPeer(remoteAddr)
	cancelCtx()
}

//GetWsHandler get Handler for websocket endpoint route setup
func GetWsHandler(serverCtx stdCtx.Context) context.Handler {

	var myHTTPAddress = []byte(configstore.GetAddressStore().GetServerAddress()) //must have config value already asserted at start ; only used as bytes

	var wsIncoming func(w http.ResponseWriter, r *http.Request)

	wsInitOnce.Do(func() {

		upgrader := wsc.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				ver := r.Header.Get(headerClusterProtoVersion)
				if ver != systemClusterProtocolVersion {
					return false
				}
				return configstore.CheckAddressAcceptable([]byte(r.RemoteAddr))
			},
		}
		wsIncoming = func(w http.ResponseWriter, r *http.Request) {

			c, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				log.Print("ERROR Upgrade:", err)
				return
			}

			fmt.Println("connection from ", r.RemoteAddr, "accepted")
			//blocking function
			startSocket(serverCtx, c, myHTTPAddress, true)
		}
	})
	return iris.FromStd(wsIncoming)
}

//JoinCluster try join peers known by config
func JoinCluster(ctx stdCtx.Context, knownPeers []interface{}) {
	for _, p := range knownPeers {
		if strPeerAddr, ok := p.(string); ok {
			err := tryConnect(ctx, strPeerAddr)
			if err != nil {
				fmt.Println("Peer not found", p, err, "(ignored now, but eventually will know every peers)")
			}
		}
	}
}
