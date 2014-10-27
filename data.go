package bgp

import "net"

type Prefix []byte

const (
	OPTIONAL        = (1 << 0)
	TRANSITIVE      = (1 << 1)
	PARTIAL         = (1 << 2)
	EXTENDED_LENGTH = (1 << 3)

	ORIGIN           = 1
	AS_PATH          = 2
	NEXT_HOP         = 3
	MULTI_EXIT_DISC  = 4
	LOCAL_PREF       = 5
	ATOMIC_AGGREGATE = 6
	AGGREGATOR       = 7

	// ORIGIN codes
	IGP        = 0
	EGP        = 1
	INCOMPLETE = 2

	// AS_PATH types
	AS_SET      = 1
	AS_SEQUENCE = 2
)

type PathAttr struct {
	Flags uint8
	Code  uint8
	Body  []byte
}

func (attr PathAttr) BGPEncode() []byte {
	attr.Flags |= EXTENDED_LENGTH
	if len(attr.Body) < 256 {
		attr.Flags ^= EXTENDED_LENGTH
	}
	bodyOffset := 3 + int((attr.Flags&EXTENDED_LENGTH)>>3)
	buf := make([]byte, bodyOffset+len(attr.Body))

	buf[0] = attr.Flags
	buf[1] = attr.Code
	if (attr.Flags & EXTENDED_LENGTH) == 0 {
		buf[2] = byte(len(attr.Body) >> 8)
		buf[3] = byte(len(attr.Body))
	} else {
		buf[2] = byte(len(attr.Body))
	}
	copy(buf[bodyOffset:], attr.Body)

	return buf
}

func DecodeIPv4Prefix(buf Prefix) (net.IPNet, int) {
	prefixLen := int(buf[0])
	prefixBytes := (prefixLen + 7) / 8
	prefix := make([]byte, 4)
	copy(prefix, buf[1:prefixBytes+1])
	return net.IPNet{IP: net.IP(prefix), Mask: net.CIDRMask(prefixLen, 32)}, prefixBytes + 1
}

func EncodeIPv4Prefix(prefix net.IPNet) Prefix {
	prefixLen, _ := prefix.Mask.Size()
	prefixBytes := (prefixLen + 7) / 8
	ret := make([]byte, 0, prefixBytes+1)
	ret = append(ret, byte(prefixLen))
	return Prefix(append(ret, []byte(prefix.IP.To4())[:prefixBytes]...))
}

func Notification(code, subcode uint8, data []byte) BGPFrame {
	buffer := make([]byte, len(data)+2)
	buffer[0] = code
	buffer[1] = subcode
	copy(buffer[2:], data)

	return BGPFrame{Type: NOTIFICATION, Body: buffer}
}
