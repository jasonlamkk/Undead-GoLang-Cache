package network

//KeyValueInjecter provide a interface for inject key, value and expire
type KeyValueInjecter interface {
	InjectResult(key string, output []byte, expire int64)
}
