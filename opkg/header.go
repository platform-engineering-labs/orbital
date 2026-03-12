package opkg

/*
	Versions: 0
	Compression: 0 zstd
*/

const (
	Magic       string = "opkg79"
	Version     uint8  = 0
	Compression uint8  = 0
)

type Header struct {
	Magic       [6]byte `struc:"little"`
	Version     uint8
	Compression uint8

	ManifestLength uint32
}

func NewHeader(version uint8, compression uint8) *Header {
	header := &Header{Version: version, Compression: compression}
	copy(header.Magic[:], Magic)
	return header
}
