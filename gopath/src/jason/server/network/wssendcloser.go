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

type msg struct {
	t int
	b []byte
}

//ProtectedSocket is a SendCloser wrapper with mutex protection
type ProtectedSocket struct {
	ch chan *msg
}

//NewProtectedSocket create a ProtectedSocket from the original SendCloser
func NewProtectedSocket(s SendCloser) *ProtectedSocket {
	ch := make(chan *msg)
	go func() {
		for {
			m, ok := <-ch
			if !ok {
				break
			}
			s.WriteMessage(m.t, m.b)
		}
		s.Close()
	}()
	return &ProtectedSocket{ch}
}

//WriteMessage write through the underlineing SendCloser
func (ps *ProtectedSocket) WriteMessage(t int, b []byte) (err error) {

	ps.ch <- &msg{t, b}

	return err
}

//Close close the underlineing SendCloser
func (ps *ProtectedSocket) Close() error {
	return ps.Close()
}
