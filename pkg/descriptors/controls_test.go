package descriptors

import (
	"bytes"
	"testing"
	"time"
)

func TestVideoProbeCommitControl_RoundTrip(t *testing.T) {
	original := &VideoProbeCommitControl{
		HintBitmask:            0x0001,
		FormatIndex:            1,
		FrameIndex:             2,
		FrameInterval:          33333300 * time.Nanosecond, // ~30fps
		KeyFrameRate:           30,
		PFrameRate:             1,
		CompQuality:            5000,
		CompWindowSize:         1000,
		Delay:                  100,
		MaxVideoFrameSize:      1920 * 1080 * 2,
		MaxPayloadTransferSize: 3072,
		ClockFrequency:         48000000,
		FramingInfoBitmask:     0x01,
		PreferedVersion:        0x01,
		MinVersion:             0x00,
		MaxVersion:             0x01,
	}

	// Marshal
	data, err := original.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}

	// Unmarshal
	decoded := &VideoProbeCommitControl{}
	if err := decoded.UnmarshalBinary(data); err != nil {
		t.Fatalf("UnmarshalBinary failed: %v", err)
	}

	// Compare fields
	if decoded.HintBitmask != original.HintBitmask {
		t.Errorf("HintBitmask = %d, want %d", decoded.HintBitmask, original.HintBitmask)
	}
	if decoded.FormatIndex != original.FormatIndex {
		t.Errorf("FormatIndex = %d, want %d", decoded.FormatIndex, original.FormatIndex)
	}
	if decoded.FrameIndex != original.FrameIndex {
		t.Errorf("FrameIndex = %d, want %d", decoded.FrameIndex, original.FrameIndex)
	}
	if decoded.FrameInterval != original.FrameInterval {
		t.Errorf("FrameInterval = %v, want %v", decoded.FrameInterval, original.FrameInterval)
	}
	if decoded.MaxVideoFrameSize != original.MaxVideoFrameSize {
		t.Errorf("MaxVideoFrameSize = %d, want %d", decoded.MaxVideoFrameSize, original.MaxVideoFrameSize)
	}
	if decoded.MaxPayloadTransferSize != original.MaxPayloadTransferSize {
		t.Errorf("MaxPayloadTransferSize = %d, want %d", decoded.MaxPayloadTransferSize, original.MaxPayloadTransferSize)
	}
	if decoded.ClockFrequency != original.ClockFrequency {
		t.Errorf("ClockFrequency = %d, want %d", decoded.ClockFrequency, original.ClockFrequency)
	}
}

func TestVideoProbeCommitControl_UnmarshalBinary_UVC10(t *testing.T) {
	// UVC 1.0 format: 26 bytes
	buf := make([]byte, 26)
	buf[2] = 1                                       // FormatIndex
	buf[3] = 2                                       // FrameIndex
	buf[4], buf[5], buf[6], buf[7] = 0x15, 0xF9, 0x00, 0x00 // FrameInterval = 333333 (30fps in 100ns units)
	buf[18], buf[19], buf[20], buf[21] = 0x00, 0x00, 0x10, 0x00 // MaxVideoFrameSize = 1048576

	vpcc := &VideoProbeCommitControl{}
	if err := vpcc.UnmarshalBinary(buf); err != nil {
		t.Fatalf("UnmarshalBinary failed: %v", err)
	}

	if vpcc.FormatIndex != 1 {
		t.Errorf("FormatIndex = %d, want 1", vpcc.FormatIndex)
	}
	if vpcc.FrameIndex != 2 {
		t.Errorf("FrameIndex = %d, want 2", vpcc.FrameIndex)
	}
	if vpcc.MaxVideoFrameSize != 1048576 {
		t.Errorf("MaxVideoFrameSize = %d, want 1048576", vpcc.MaxVideoFrameSize)
	}
}

func TestVideoProbeCommitControl_MarshalInto(t *testing.T) {
	vpcc := &VideoProbeCommitControl{
		FormatIndex:      1,
		FrameIndex:       3,
		MaxVideoFrameSize: 1024,
	}

	// Test marshaling into a 26-byte buffer (UVC 1.0)
	buf26 := make([]byte, 26)
	if err := vpcc.MarshalInto(buf26); err != nil {
		t.Fatalf("MarshalInto(26) failed: %v", err)
	}
	if buf26[2] != 1 {
		t.Errorf("buf26[2] (FormatIndex) = %d, want 1", buf26[2])
	}
	if buf26[3] != 3 {
		t.Errorf("buf26[3] (FrameIndex) = %d, want 3", buf26[3])
	}

	// Test marshaling into a 34-byte buffer (UVC 1.1)
	vpcc.ClockFrequency = 48000000
	vpcc.PreferedVersion = 0x01
	buf34 := make([]byte, 34)
	if err := vpcc.MarshalInto(buf34); err != nil {
		t.Fatalf("MarshalInto(34) failed: %v", err)
	}
	// ClockFrequency should be at bytes 26-29
	if buf34[31] != 0x01 {
		t.Errorf("buf34[31] (PreferedVersion) = %d, want 1", buf34[31])
	}
}

func TestVideoProbeCommitControl_FrameIntervalConversion(t *testing.T) {
	// Test that frame interval is correctly converted to/from 100ns units
	vpcc := &VideoProbeCommitControl{
		FrameInterval: 33333300 * time.Nanosecond, // ~30fps
	}

	data, _ := vpcc.MarshalBinary()

	decoded := &VideoProbeCommitControl{}
	decoded.UnmarshalBinary(data)

	// Should be within 100ns precision
	diff := vpcc.FrameInterval - decoded.FrameInterval
	if diff < 0 {
		diff = -diff
	}
	if diff > 100*time.Nanosecond {
		t.Errorf("FrameInterval precision loss: original=%v, decoded=%v", vpcc.FrameInterval, decoded.FrameInterval)
	}
}

func TestVideoProbeCommitControl_MarshalBinary_Length(t *testing.T) {
	vpcc := &VideoProbeCommitControl{}
	data, err := vpcc.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}
	// Default marshal should produce 48 bytes (UVC 1.5 full format)
	if len(data) != 48 {
		t.Errorf("MarshalBinary length = %d, want 48", len(data))
	}
}

func TestVideoProbeCommitControl_ZeroValues(t *testing.T) {
	// Test that zero-initialized struct marshals and unmarshals correctly
	original := &VideoProbeCommitControl{}
	data, _ := original.MarshalBinary()

	decoded := &VideoProbeCommitControl{}
	if err := decoded.UnmarshalBinary(data); err != nil {
		t.Fatalf("UnmarshalBinary failed: %v", err)
	}

	// All fields should be zero
	if decoded.FormatIndex != 0 || decoded.FrameIndex != 0 {
		t.Error("Zero-initialized struct should unmarshal to zero values")
	}
}

func TestVideoProbeCommitControl_ByteOrder(t *testing.T) {
	// Verify little-endian byte order
	vpcc := &VideoProbeCommitControl{
		HintBitmask:       0x1234,
		MaxVideoFrameSize: 0xDEADBEEF,
	}

	data, _ := vpcc.MarshalBinary()

	// HintBitmask at bytes 0-1 (little endian: 0x34, 0x12)
	if data[0] != 0x34 || data[1] != 0x12 {
		t.Errorf("HintBitmask bytes = [%02x, %02x], want [34, 12]", data[0], data[1])
	}

	// MaxVideoFrameSize at bytes 18-21 (little endian: EF, BE, AD, DE)
	if !bytes.Equal(data[18:22], []byte{0xEF, 0xBE, 0xAD, 0xDE}) {
		t.Errorf("MaxVideoFrameSize bytes = %x, want EFBEADDE", data[18:22])
	}
}
