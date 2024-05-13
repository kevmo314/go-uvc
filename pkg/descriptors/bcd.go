package descriptors

type BinaryCodedDecimal uint16

func (bcd BinaryCodedDecimal) Uint16Value() uint16 {
	// read as little endian bcd
	return ((uint16(bcd&0x00f0) >> 4) * 1000) + (uint16(bcd&0x000f) * 100) + ((uint16(bcd&0xf000) >> 12) * 10) + (uint16(bcd&0x0f00) >> 8)
}
