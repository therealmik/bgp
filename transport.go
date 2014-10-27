package main

import (
	"fmt"
	"net"
)

type BGPFrame struct {
	Type byte
	Body []byte
}

func ReadProc(conn net.Conn, ch chan<- BGPFrame) {
	header := make([]byte, 19)

	for {
		n := 0
		for n < 19 {
			m, err := conn.Read(header[n:19])
			if err != nil {
				fmt.Println("Error reading from peer:", err)
				close(ch)
				return
			}
			n += m
		}
		length := (int(header[16]) << 8) | int(header[17])
		if length < 19 || length > 4096 {
			fmt.Println("Message too long:", length)
			close(ch)
			return
		}

		length -= 19
		body := make([]byte, length)
		n = 0
		for n < length {
			m, err := conn.Read(body[n:length])
			if err != nil {
				fmt.Println("Error reading from peer:", err)
				close(ch)
				return
			}
			n += m
		}
		ch <- BGPFrame{header[18], body}
	}
}

func WriteProc(conn net.Conn, ch <-chan BGPFrame) {
	buffer := make([]byte, 4096)
	for i := 0; i < 16; i++ {
		buffer[i] = 0xff
	}

	for msg := range ch {
		length := len(msg.Body) + 19
		if length > 4096 {
			fmt.Println("Warning: not attempting to send message > 4096 octets")
		}
		if length > 65535 || length < 0 {
			fmt.Println("Error: cannot send message > 65535 bytes, killing connection!")
			break
		}
		buffer[16] = byte(length >> 8)
		buffer[17] = byte(length & 0xff)
		buffer[18] = msg.Type

		copy(buffer[19:], msg.Body)

		n := 0
		for n < length {
			m, err := conn.Write(buffer[n:length])
			if err != nil {
				fmt.Println("Error writing to peer:", err)
				break
			}
			n += m
		}
	}
	conn.Close()
}

func startTransport(conn net.Conn) (chan BGPFrame, chan BGPFrame) {
	readCh := make(chan BGPFrame)
	writeCh := make(chan BGPFrame)

	go ReadProc(conn, readCh)
	go WriteProc(conn, writeCh)

	return readCh, writeCh
}

func Neighbor(conn net.Conn, autonomousSystem uint16, holdTime int, bgpIdentifier uint32, optionalParameters []byte) *BGPProc {
	readCh, writeCh := startTransport(conn)

	writeCh <- EncodeOpen(&OpenMessage{
		Version:            4,
		AutonomousSystem:   autonomousSystem,
		HoldTime:           holdTime,
		BGPIdentifier:      bgpIdentifier,
		OptionalParameters: optionalParameters,
	})

	msg := <-readCh
	if msg.Type != OPEN {
		fmt.Println("Got unexpected message type as first message: ", msg.Type)
		close(writeCh)
		return
	}

}
