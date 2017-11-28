package configstore

//XAPIKeyStore store external API keys
type XAPIKeyStore struct {
	gmapKey string
}

var sharedKeyStore XAPIKeyStore

//GetXKeyStore get key store for exteranl API keys
func GetXKeyStore() *XAPIKeyStore {
	return &sharedKeyStore
}

//GetGMapKey get google map API key
func (x *XAPIKeyStore) GetGMapKey() string {
	return x.gmapKey
}

//SetGMApKey set google map API key
func (x *XAPIKeyStore) SetGMApKey(k string) {
	x.gmapKey = k
}
