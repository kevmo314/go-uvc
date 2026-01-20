package transfers

import (
	"io"
	"testing"
)

func TestPayloadUnmarshalBinary_MinimalHeader(t *testing.T) {
	// Minimal header: 2 bytes (length + bitmask), no PTS/SCR
	// buf[0] = header length (2)
	// buf[1] = bitmask (0x80 = EndOfHeader only)
	buf := []byte{2, 0x80, 0xDE, 0xAD, 0xBE, 0xEF}

	p := &Payload{}
	if err := p.UnmarshalBinary(buf); err != nil {
		t.Fatalf("UnmarshalBinary failed: %v", err)
	}

	if p.HeaderInfoBitmask != 0x80 {
		t.Errorf("HeaderInfoBitmask = %02x, want %02x", p.HeaderInfoBitmask, 0x80)
	}
	if p.HasPTS() {
		t.Error("HasPTS() = true, want false")
	}
	if p.HasSCR() {
		t.Error("HasSCR() = true, want false")
	}
	if !p.EndOfHeader() {
		t.Error("EndOfHeader() = false, want true")
	}
	if len(p.Data) != 4 {
		t.Errorf("Data length = %d, want 4", len(p.Data))
	}
	if p.Data[0] != 0xDE || p.Data[1] != 0xAD {
		t.Errorf("Data = %x, want DEADBEEF", p.Data)
	}
}

func TestPayloadUnmarshalBinary_WithPTS(t *testing.T) {
	// Header with PTS: 6 bytes (length + bitmask + 4 bytes PTS)
	// buf[0] = header length (6)
	// buf[1] = bitmask (0x04 = HasPTS)
	// buf[2:6] = PTS (little endian)
	buf := []byte{
		6,                      // header length
		0x04,                   // bitmask: HasPTS
		0x01, 0x02, 0x03, 0x04, // PTS = 0x04030201
		0xAA, 0xBB, // payload data
	}

	p := &Payload{}
	if err := p.UnmarshalBinary(buf); err != nil {
		t.Fatalf("UnmarshalBinary failed: %v", err)
	}

	if !p.HasPTS() {
		t.Error("HasPTS() = false, want true")
	}
	if p.HasSCR() {
		t.Error("HasSCR() = true, want false")
	}
	if p.PTS != 0x04030201 {
		t.Errorf("PTS = %08x, want %08x", p.PTS, 0x04030201)
	}
	if len(p.Data) != 2 {
		t.Errorf("Data length = %d, want 2", len(p.Data))
	}
}

func TestPayloadUnmarshalBinary_WithSCR(t *testing.T) {
	// Header with SCR: 8 bytes (length + bitmask + 4 bytes STC + 2 bytes token)
	// buf[0] = header length (8)
	// buf[1] = bitmask (0x08 = HasSCR)
	buf := []byte{
		8,                      // header length
		0x08,                   // bitmask: HasSCR
		0x11, 0x22, 0x33, 0x44, // STC = 0x44332211
		0x55, 0x66, // TokenCounter = 0x6655
		0xCC, 0xDD, // payload data
	}

	p := &Payload{}
	if err := p.UnmarshalBinary(buf); err != nil {
		t.Fatalf("UnmarshalBinary failed: %v", err)
	}

	if p.HasPTS() {
		t.Error("HasPTS() = true, want false")
	}
	if !p.HasSCR() {
		t.Error("HasSCR() = false, want true")
	}
	if p.SCR.SourceTimeClock != 0x44332211 {
		t.Errorf("SCR.SourceTimeClock = %08x, want %08x", p.SCR.SourceTimeClock, 0x44332211)
	}
	if p.SCR.TokenCounter != 0x6655 {
		t.Errorf("SCR.TokenCounter = %04x, want %04x", p.SCR.TokenCounter, 0x6655)
	}
	if len(p.Data) != 2 {
		t.Errorf("Data length = %d, want 2", len(p.Data))
	}
}

func TestPayloadUnmarshalBinary_WithPTSAndSCR(t *testing.T) {
	// Header with both PTS and SCR: 12 bytes
	buf := []byte{
		12,                     // header length
		0x0C,                   // bitmask: HasPTS | HasSCR
		0x01, 0x02, 0x03, 0x04, // PTS
		0x11, 0x22, 0x33, 0x44, // STC
		0x55, 0x66, // TokenCounter
		0xEE, 0xFF, // payload data
	}

	p := &Payload{}
	if err := p.UnmarshalBinary(buf); err != nil {
		t.Fatalf("UnmarshalBinary failed: %v", err)
	}

	if !p.HasPTS() {
		t.Error("HasPTS() = false, want true")
	}
	if !p.HasSCR() {
		t.Error("HasSCR() = false, want true")
	}
	if p.PTS != 0x04030201 {
		t.Errorf("PTS = %08x, want %08x", p.PTS, 0x04030201)
	}
	if p.SCR.SourceTimeClock != 0x44332211 {
		t.Errorf("SCR.SourceTimeClock = %08x, want %08x", p.SCR.SourceTimeClock, 0x44332211)
	}
	if len(p.Data) != 2 {
		t.Errorf("Data length = %d, want 2", len(p.Data))
	}
}

func TestPayloadUnmarshalBinary_ShortBuffer(t *testing.T) {
	// Buffer claims to be 10 bytes but only has 5
	buf := []byte{10, 0x00, 0x01, 0x02, 0x03}

	p := &Payload{}
	err := p.UnmarshalBinary(buf)
	if err != io.ErrShortBuffer {
		t.Errorf("UnmarshalBinary error = %v, want io.ErrShortBuffer", err)
	}
}

func TestPayloadBitfieldAccessors(t *testing.T) {
	tests := []struct {
		bitmask  uint8
		name     string
		accessor func(*Payload) bool
		want     bool
	}{
		{0b00000001, "FrameID(1)", (*Payload).FrameID, true},
		{0b00000000, "FrameID(0)", (*Payload).FrameID, false},
		{0b00000010, "EndOfFrame(1)", (*Payload).EndOfFrame, true},
		{0b00000000, "EndOfFrame(0)", (*Payload).EndOfFrame, false},
		{0b00000100, "HasPTS(1)", (*Payload).HasPTS, true},
		{0b00000000, "HasPTS(0)", (*Payload).HasPTS, false},
		{0b00001000, "HasSCR(1)", (*Payload).HasSCR, true},
		{0b00000000, "HasSCR(0)", (*Payload).HasSCR, false},
		{0b00010000, "PayloadSpecificBit(1)", (*Payload).PayloadSpecificBit, true},
		{0b00000000, "PayloadSpecificBit(0)", (*Payload).PayloadSpecificBit, false},
		{0b00100000, "StillImage(1)", (*Payload).StillImage, true},
		{0b00000000, "StillImage(0)", (*Payload).StillImage, false},
		{0b01000000, "Error(1)", (*Payload).Error, true},
		{0b00000000, "Error(0)", (*Payload).Error, false},
		{0b10000000, "EndOfHeader(1)", (*Payload).EndOfHeader, true},
		{0b00000000, "EndOfHeader(0)", (*Payload).EndOfHeader, false},
		{0b11111111, "AllBits", (*Payload).FrameID, true},
	}

	for _, tt := range tests {
		p := &Payload{HeaderInfoBitmask: tt.bitmask}
		if got := tt.accessor(p); got != tt.want {
			t.Errorf("%s with bitmask %08b = %v, want %v", tt.name, tt.bitmask, got, tt.want)
		}
	}
}

func TestPayloadUnmarshalBinary_EndOfFrameAndFrameID(t *testing.T) {
	// Test frame boundary detection flags
	buf := []byte{2, 0x03, 0x00} // FrameID=1, EndOfFrame=1

	p := &Payload{}
	if err := p.UnmarshalBinary(buf); err != nil {
		t.Fatalf("UnmarshalBinary failed: %v", err)
	}

	if !p.FrameID() {
		t.Error("FrameID() = false, want true")
	}
	if !p.EndOfFrame() {
		t.Error("EndOfFrame() = false, want true")
	}
}
