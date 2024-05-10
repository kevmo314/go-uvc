package descriptors

type FormatDescriptor struct {
	DVFormatDescriptor           DVFormatDescriptor
	StreamBasedFormatDescriptor  StreamBasedFormatDescriptor
	FrameBasedFormatDescriptor   FrameBasedFormatDescriptor
	H264FormatDescriptor         H264FormatDescriptor
	VP8FormatDescriptor          VP8FormatDescriptor
	MJPEGFormatDescriptor        MJPEGFormatDescriptor
	UncompressedFormatDescriptor UncompressedFormatDescriptor
}

type FrameDescriptor struct {
	FrameBasedFrameDescriptor   FrameBasedFrameDescriptor
	H264FrameDescriptor         H264FrameDescriptor
	VP8FrameDescriptor          VP8FrameDescriptor
	MJPEGFrameDescriptor        MJPEGFrameDescriptor
	UncompressedFrameDescriptor UncompressedFrameDescriptor
}
