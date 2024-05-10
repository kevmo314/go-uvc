package formats

import "github.com/google/uuid"

type CompressionFormat [16]byte

var (
	CompressionFormatYUY2 = CompressionFormat(uuid.MustParse("32595559-0000-0010-8000-00AA00389B71"))
	CompressionFormatNV12 = CompressionFormat(uuid.MustParse("3231564E-0000-0010-8000-00AA00389B71"))
	CompressionFormatM420 = CompressionFormat(uuid.MustParse("3032344D-0000-0010-8000-00AA00389B71"))
	CompressionFormatI420 = CompressionFormat(uuid.MustParse("30323449-0000-0010-8000-00AA00389B71"))
)
