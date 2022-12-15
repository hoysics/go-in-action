package compress

import (
	"github.com/pierrec/lz4/v4"
)

var _ Compressor = &Lz4{}

type Lz4 struct {
	lz4.Compressor
	size int
}

func (lc *Lz4) Code() byte {
	return 1
}
func (lc *Lz4) Compress(originBuff []byte) ([]byte, error) {
	buf := make([]byte, lz4.CompressBlockBound(len(originBuff)))
	n, err := lc.CompressBlock(originBuff, buf)
	if err != nil {
		return nil, err
	}
	lc.size = len(originBuff)
	return buf[:n], nil

}

func (lc *Lz4) UnCompress(originBuff []byte) ([]byte, error) {
	val := make([]byte, lc.size)
	n, err := lz4.UncompressBlock(originBuff, val)
	if err != nil {
		return nil, err
	}
	return val[:n], nil

}
