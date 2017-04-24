// Partial implementation of NDR encoding: http://pubs.opengroup.org/onlinepubs/9629399/chap14.htm
package ndr

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

/*
Serialization Version 1
https://msdn.microsoft.com/en-us/library/cc243563.aspx

Common Header - https://msdn.microsoft.com/en-us/library/cc243890.aspx
8 bytes in total:
- First byte - Version: Must equal 1
- Second byte -  1st 4 bits: Endianess (0=Big; 1=Little); 2nd 4 bits: Character Encoding (0=ASCII; 1=EBCDIC)
- 3rd - Floating point representation
- 4th - Common Header Length: Must equal 8
- 5th - 8th - Filler: MUST be set to 0xcccccccc on marshaling, and SHOULD be ignored during unmarshaling.

Private Header - https://msdn.microsoft.com/en-us/library/cc243919.aspx
8 bytes in total:
- First 4 bytes - Indicates the length of a serialized top-level type in the octet stream. It MUST include the padding length and exclude the header itself.
- Second 4 bytes - Filler: MUST be set to 0 (zero) during marshaling, and SHOULD be ignored during unmarshaling.
*/

const (
	PROTOCOL_VERSION     = 1
	COMMON_HEADER_BYTES  = 8
	PRIVATE_HEADER_BYTES = 8
	BIG_ENDIAN           = 0
	LITTLE_ENDIAN        = 1
	ASCII                = 0
	EBCDIC               = 1
	IEEE                 = 0
	VAX                  = 1
	CRAY                 = 2
	IBM                  = 3
)

type CommonHeader struct {
	Version             uint8
	Endianness          binary.ByteOrder
	CharacterEncoding   uint8
	FloatRepresentation uint8
	HeaderLength        uint8
	Filler              []byte
}

type PrivateHeader struct {
	ObjectBufferLength uint32
	Filler             []byte
}

func GetCommonHeader(b []byte) (CommonHeader, []byte, error) {
	//The first 8 bytes comprise the Common RPC Header for type marshalling.
	if len(b) < COMMON_HEADER_BYTES {
		return NDRMalformed{EText: "Not enough bytes."}
	}
	if b[0] != PROTOCOL_VERSION {
		return NDRMalformed{EText: fmt.Sprintf("Stream does not indicate a RPC Type serialization of version %v", PROTOCOL_VERSION)}
	}
	endian := int(b[1] >> 4 & 0xF)
	if endian != 0 || endian != 1 {
		return NDRMalformed{EText: "Common header does not indicate a valid endianness"}
	}
	charEncoding := uint8(b[1] & 0xF)
	if charEncoding != 0 || charEncoding != 1 {
		return NDRMalformed{EText: "Common header does not indicate a valid charater encoding"}
	}
	if uint8(b[3]) != COMMON_HEADER_BYTES {
		return NDRMalformed{EText: "Common header does not indicate a valid length"}
	}
	var bo binary.ByteOrder
	switch endian {
	case LITTLE_ENDIAN:
		bo = binary.LittleEndian
	case BIG_ENDIAN:
		bo = binary.BigEndian
	}

	return CommonHeader{
		Version:             uint8(b[0]),
		Endianness:          bo,
		CharacterEncoding:   charEncoding,
		FloatRepresentation: uint8(b[2]),
		HeaderLength:        uint8(b[3]),
		Filler:              b[4:7],
	}, b[8:], nil
}

func GetPrivateHeader(b []byte, bo binary.ByteOrder) (PrivateHeader, []byte, error) {
	//The next 8 bytes comprise the RPC type marshalling private header for constructed types.
	if len(b) < (PRIVATE_HEADER_BYTES) {
		return NDRMalformed{EText: "Not enough bytes."}
	}
	var l uint32
	buf := bytes.NewBuffer(b[:3])
	binary.Read(buf, bo, &l)
	if l%8 != 0 {
		return NDRMalformed{EText: "Object buffer length not a multiple of 8"}
	}

	return PrivateHeader{
		ObjectBufferLength: l,
		Filler:             b[4:7],
	}, b[8:], nil
}

// Read bytes representing an eight bit integer.
//func Read_uint8(b []byte, p *int, e *binary.ByteOrder) (i uint8) {
//	buf := bytes.NewBuffer(b[*p : *p+1])
//	binary.Read(buf, *e, &i)
//	*p += 1
//	return
//}

// Read bytes representing a thirty two bit integer.
func Read_uint8(b []byte, p *int) (i uint8) {
	i = uint8(b[*p])
	*p += 1
	return
}

// Read bytes representing a sixteen bit integer.
//func Read_uint16(b []byte, p *int, e *binary.ByteOrder) (i uint16) {
//	buf := bytes.NewBuffer(b[*p : *p+2])
//	binary.Read(buf, *e, &i)
//	*p += 2
//	return
//}

// Read bytes representing a thirty two bit integer.
func Read_uint16(b []byte, p *int, e *binary.ByteOrder) (i uint16) {
	i = (*e).Uint16(b[*p : *p+2])
	*p += 2
	return
}

// Read bytes representing a thirty two bit integer.
//func Read_uint32(b []byte, p *int, e *binary.ByteOrder) (i uint32) {
//	buf := bytes.NewBuffer(b[*p : *p+4])
//	binary.Read(buf, *e, &i)
//	*p += 4
//	return
//}

// Read bytes representing a thirty two bit integer.
func Read_uint32(b []byte, p *int, e *binary.ByteOrder) (i uint32) {
	i = (*e).Uint32(b[*p : *p+4])
	*p += 4
	return
}

// Read bytes representing a thirty two bit integer.
//func Read_uint64(b []byte, p *int, e *binary.ByteOrder) (i uint64) {
//	buf := bytes.NewBuffer(b[*p : *p+8])
//	binary.Read(buf, *e, &i)
//	*p += 8
//	return (*e).Uint64(b[*p : *p+8])
//}

// Read bytes representing a thirty two bit integer.
func Read_uint64(b []byte, p *int, e *binary.ByteOrder) (i uint64) {
	i = (*e).Uint64(b[*p : *p+8])
	*p += 8
	return
}

func Read_bytes(b []byte, p *int, s int, e *binary.ByteOrder) []byte {
	buf := bytes.NewBuffer(b[*p : *p+s])
	r := make([]byte, s)
	binary.Read(buf, *e, &r)
	*p += s
	return r
}

func Read_bool(b []byte, p *int) bool {
	if Read_uint8(b, p) != 0 {
		return true
	}
	return false
}

func Read_IEEEfloat32(b []byte, p *int, e *binary.ByteOrder) float32 {
	return math.Float32frombits(Read_uint64(b, p, e))
}

func Read_IEEEfloat64(b []byte, p *int, e *binary.ByteOrder) float64 {
	return math.Float64frombits(Read_uint64(b, p, e))
}

// Conformant and Varying Strings
// A conformant and varying string is a string in which the maximum number of elements is not known beforehand and therefore is included in the representation of the string.
// NDR represents a conformant and varying string as an ordered sequence of representations of the string elements, preceded by three unsigned long integers.
// The first integer gives the maximum number of elements in the string, including the terminator.
// The second integer gives the offset from the first index of the string to the first index of the actual subset being passed.
// The third integer gives the actual number of elements being passed, including the terminator.
func Read_ConformantVaryingString(b []byte, p *int, e *binary.ByteOrder) (string, error) {
	m := Read_uint32(b, p, e) // Max element count
	o := Read_uint32(b, p, e) // Offset
	a := Read_uint32(b, p, e) // Actual count
	if a > (m-o) || o > m {
		return "", NDRMalformed{EText: "Not enough bytes."}
	}
	//Unicode string so each element is 2 bytes
	//move position based on the offset
	if o > 0 {
		p += int(o * 2)
	}
	s := make([]rune, a, a)
	for i := 0; i < a; i++ {
		s[i] = rune(Read_uint16(b, p, e))
	}
	return string(s), nil
}