package descriptors

type ClassCode byte

const (
	ClassCodeVideo ClassCode = 0x0E
)

type SubclassCode byte

const (
	SubclassCodeUndefined                SubclassCode = 0x00
	SubclassCodeVideoControl             SubclassCode = 0x01
	SubclassCodeVideoStreaming           SubclassCode = 0x02
	SubclassCodeVideoInterfaceCollection SubclassCode = 0x03
)

type ProtocolCode byte

const (
	ProtocolCodeUndefined ProtocolCode = 0x00
	ProtocolCode15        ProtocolCode = 0x01
)

type ClassSpecificDescriptorType int

const (
	ClassSpecificDescriptorTypeUndefined     ClassSpecificDescriptorType = 0x20
	ClassSpecificDescriptorTypeDevice        ClassSpecificDescriptorType = 0x21
	ClassSpecificDescriptorTypeConfiguration ClassSpecificDescriptorType = 0x22
	ClassSpecificDescriptorTypeString        ClassSpecificDescriptorType = 0x23
	ClassSpecificDescriptorTypeInterface     ClassSpecificDescriptorType = 0x24
	ClassSpecificDescriptorTypeEndpoint      ClassSpecificDescriptorType = 0x25
)
