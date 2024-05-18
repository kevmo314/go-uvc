package descriptors

func copyGUID(dst []byte, src []byte) {
	// copy according to the GUID format defined in UVC spec 1.5, section 2.9.
	dst[0] = src[3]
	dst[1] = src[2]
	dst[2] = src[1]
	dst[3] = src[0]
	dst[4] = src[5]
	dst[5] = src[4]
	dst[6] = src[7]
	dst[7] = src[6]
	dst[8] = src[8]
	dst[9] = src[9]
	dst[10] = src[10]
	dst[11] = src[11]
	dst[12] = src[12]
	dst[13] = src[13]
	dst[14] = src[14]
	dst[15] = src[15]
}
