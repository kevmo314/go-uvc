package descriptors

type FormatDescriptor interface {
	isStreamingInterface()
	isFormatDescriptor()
}

type FrameDescriptor interface {
	isStreamingInterface()
	isFrameDescriptor()
}
