package descriptors

type FormatDescriptor interface {
	isStreamingInterface()
	isFormatDescriptor()
	// Index returns the index of the format descriptor in the format descriptor array.
	// This is identical to retrieving FormatIndex from the descriptor but is provided
	// for convenience.
	Index() uint8
}

type FrameDescriptor interface {
	isStreamingInterface()
	isFrameDescriptor()
	// Index returns the index of the frame descriptor in the frame descriptor array.
	// This is identical to retrieving FrameIndex from the descriptor but is provided
	// for convenience.
	Index() uint8
}
