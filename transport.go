package bgp

import (
	"fmt"
	"net"
	"time"
)

type Error string

func (e Error) Error() string {
	return string(e)
}

type BGPFrame struct {
	Type byte
	Body []byte
}

type Connection struct {
	Conn        net.Conn
	RecvChannel chan BGPFrame
	SendChannel chan BGPFrame
}

func (self *Connection) ReadProc() {
	header := make([]byte, 19)

ReadLoop:
	for {

		n := 0
		for n < 19 {
			m, err := self.Conn.Read(header[n:19])
			if err != nil {
				fmt.Println("Error reading from peer:", err)
				break ReadLoop
			}
			n += m
		}
		length := (int(header[16]) << 8) | int(header[17])
		if length < 19 || length > 4096 {
			fmt.Println("Message too long:", length)
			self.SendChannel <- Notification(1, 1, nil)
			break ReadLoop
		}

		length -= 19
		body := make([]byte, length)
		n = 0
		for n < length {
			m, err := self.Conn.Read(body[n:length])
			if err != nil {
				fmt.Println("Error reading from peer:", err)
				break ReadLoop
			}
			n += m
		}
		self.RecvChannel <- BGPFrame{header[18], body}
	}
	close(self.RecvChannel)
	self.Conn.Close()
}

func (self *Connection) Write(frame BGPFrame) error {
	length := len(frame.Body) + 19

	buffer := make([]byte, length)
	for i := 0; i < 16; i++ {
		buffer[i] = 0xff
	}

	if length > 4096 || length < 0 {
		return Error("Cannot send message > 4096 octets")
	}

	buffer[16] = byte(length >> 8)
	buffer[17] = byte(length & 0xff)
	buffer[18] = frame.Type

	copy(buffer[19:], frame.Body)

	n := 0
	for n < length {
		m, err := self.Conn.Write(buffer[n:length])
		if err != nil {
			return err
		}
		n += m
	}

	return nil
}

func (self *Connection) WriteProc(keepaliveInterval time.Duration) {
	keepaliveFrame := BGPFrame{Type: KEEPALIVE, Body: []byte(nil)}
	self.Write(keepaliveFrame)

	for {
		timeout := make(chan bool, 1)
		go func() {
			time.Sleep(keepaliveInterval)
			timeout <- true
		}()

		var err error

		select {
		case frame, ok := <-self.SendChannel:
			if !ok {
				break
			}
			err = self.Write(frame)
			if frame.Type == NOTIFICATION {
				break
			}
		case <-timeout:
			err = self.Write(keepaliveFrame)
		}
		if err != nil {
			fmt.Println("Error sending message: ", err)
			break
		}
	}
	self.Conn.Close()
}

func NewConnection(conn net.Conn) *Connection {
	self := &Connection{
		Conn:        conn,
		SendChannel: make(chan BGPFrame),
		RecvChannel: make(chan BGPFrame),
	}

	go self.ReadProc()

	return self
}
