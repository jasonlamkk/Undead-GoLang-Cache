package configstore

import "fmt"

//AddressStore store and format IP and port
type AddressStore struct {
	ip        string
	portShift int
}

const (
	portBase = 80
	//WebSocketPrefix const string for ws:// protcol
	WebSocketPrefix = "ws://"
)

var sharedAddressStore AddressStore

//GetAddressStore get singleton AddressStore object
func GetAddressStore() *AddressStore {
	return &sharedAddressStore
}

//SetAddress set ip and port shift
func (a *AddressStore) SetAddress(sIP string, iShift int) {
	if portBase+iShift > 65535 || iShift < 0 {
		fmt.Println("invalid port shift range ", 0, "-", 65535-portBase)
		panic("config error")
	}
	a.ip = sIP
	a.portShift = iShift
}

//GetServerAddress get server address
func (a *AddressStore) GetServerAddress() string {
	return fmt.Sprintf("%s:%d", a.ip, portBase+a.portShift)
}

//GetHTTPAddress get HTTP server address
func (a *AddressStore) GetHTTPAddress() string {
	return "http://" + a.GetServerAddress()
}

//GetWebSocketClusterAddress get web socket server address
func (a *AddressStore) GetWebSocketClusterAddress() string {
	return WebSocketPrefix + a.GetServerAddress()
}
