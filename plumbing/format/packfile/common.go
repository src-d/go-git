package packfile

const (
	// VersionSupported is the packfile version supported by this parser.
	VersionSupported uint32 = 2
)

const (
	maskType = uint8(112) // 0111 0000
)

var signature = []byte{'P', 'A', 'C', 'K'}

const (
	lengthBits      = uint8(7)   // subsequent bytes has 7 bits to store the length
	firstLengthBits = uint8(4)   // the first byte has 4 bits to store the length
	maskFirstLength = 15         // 0000 1111
	maskContinue    = 0x80       // 1000 0000
	maskLength      = uint8(127) // 0111 1111
)
