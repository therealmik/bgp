package main

const (
	OPEN         = 1
	UPDATE       = 2
	NOTIFICATION = 3
	KEEPALIVE    = 4
)

type OpenMessage struct {
	Version            uint8
	AutonomousSystem   uint16
	HoldTime           int // This is a uint16 on the wire
	BGPIdentifier      uint32
	OptionalParameters []byte
}

func DecodeOpen(frame BGPFrame) (*OpenMessage, bool) {
	if len(frame.Body) < 10 {
		return nil, false
	}

	optParamLen := int(frame.Body[9])
	if optParamLen+10 != len(frame.Body) {
		return nil, false
	}

	msg := new(OpenMessage)
	msg.Version = uint8(frame.Body[0])
	msg.AutonomousSystem = (uint16(frame.Body[1]) << 8) | uint16(frame.Body[2])
	msg.HoldTime = (int(frame.Body[3]) << 8) | int(frame.Body[4])
	msg.BGPIdentifier = (uint32(frame.Body[5]) << 24) | (uint32(frame.Body[6]) << 16) | (uint32(frame.Body[7]) << 8) | uint32(frame.Body[8])

	return msg, true
}

func EncodeOpen(msg *OpenMessage) BGPFrame {
	body := make([]byte, 0, len(msg.OptionalParameters)+10)
	body = append(body,
		byte(msg.Version),
		byte(msg.AutonomousSystem>>8), byte(msg.AutonomousSystem),
		byte(msg.HoldTime>>8), byte(msg.HoldTime),
		byte(msg.BGPIdentifier>>24), byte(msg.BGPIdentifier>>16), byte(msg.BGPIdentifier>>8), byte(msg.BGPIdentifier),
		byte(len(msg.OptionalParameters)))
	body = append(body, msg.OptionalParameters...)
	return BGPFrame{OPEN, body}
}

func EncodeIPv4Update(withdrawen []Prefix, pathAttributes []PathAttr, nlri []Prefix) BGPFrame {
	wbuf := []byte(nil)
	for _, net := range withdrawen {
		wbuf = append(wbuf, net...)
	}

	pabuf := []byte(nil)
	for _, pa := range pathAttributes {
		pabuf = append(pabuf, pa.BGPEncode()...)
	}

	nlribuf := []byte(nil)
	for _, pfx := range nlri {
		nlribuf = append(nlribuf, pfx...)
	}

	buffer := make([]byte, len(wbuf)+len(pabuf)+len(nlribuf)+2)
	var i int

	buffer[i] = byte(len(wbuf) >> 8)
	i++

	buffer[i] = byte(len(wbuf))
	i++

	copy(buffer[i:], wbuf)
	i += len(wbuf)

	buffer[i] = byte(len(pabuf) >> 8)
	i++

	buffer[i] = byte(len(pabuf))
	i++

	copy(buffer[i:], pabuf)
	i += len(pabuf)

	copy(buffer[i:], nlribuf)

	return BGPFrame{Type: UPDATE, Body: buffer}
}
