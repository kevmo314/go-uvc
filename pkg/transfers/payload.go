package transfers

import (
	"encoding/binary"
	"io"
)

type PayloadReader interface {
	io.Closer
	ReadPayload() (*Payload, error)
}

type Payload struct {
	HeaderInfoBitmask uint8
	PTS               uint32
	SCR               struct {
		SourceTimeClock uint32
		TokenCounter    uint16
	}
	Data []byte
}

func (f *Payload) FrameID() bool {
	return f.HeaderInfoBitmask&0b00000001 != 0
}

func (f *Payload) EndOfFrame() bool {
	return f.HeaderInfoBitmask&0b00000010 != 0
}

func (f *Payload) HasPTS() bool {
	return f.HeaderInfoBitmask&0b00000100 != 0
}

func (f *Payload) HasSCR() bool {
	return f.HeaderInfoBitmask&0b00001000 != 0
}

func (f *Payload) PayloadSpecificBit() bool {
	return f.HeaderInfoBitmask&0b00010000 != 0
}

func (f *Payload) StillImage() bool {
	return f.HeaderInfoBitmask&0b00100000 != 0
}

func (f *Payload) Error() bool {
	return f.HeaderInfoBitmask&0b01000000 != 0
}

func (f *Payload) EndOfHeader() bool {
	return f.HeaderInfoBitmask&0b10000000 != 0
}

func (f *Payload) UnmarshalBinary(buf []byte) error {
	if len(buf) < int(buf[0]) {
		return io.ErrShortBuffer
	}
	f.HeaderInfoBitmask = buf[1]
	offset := 2
	if f.HasPTS() {
		f.PTS = binary.LittleEndian.Uint32(buf[offset : offset+4])
		offset += 4
	}
	if f.HasSCR() {
		f.SCR.SourceTimeClock = binary.LittleEndian.Uint32(buf[offset : offset+4])
		offset += 4
		f.SCR.TokenCounter = binary.LittleEndian.Uint16(buf[offset : offset+2])
		offset += 2
	}
	f.Data = buf[offset:]
	return nil
}
