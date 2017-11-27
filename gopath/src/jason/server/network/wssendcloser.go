package network

//Sender is a interface with WriteMessage function
type Sender interface {
	WriteMessage(int, []byte) error
}

//Closer is a interface with Close function
type Closer interface {
	Close() error
}

//SendCloser is a interface with WriteMessage and Close function
type SendCloser interface {
	WriteMessage(int, []byte) error
	Close() error
}
