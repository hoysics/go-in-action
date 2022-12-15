package compress

import (
	"bytes"
	"compress/gzip"
	"io"
)

var _ Compressor = &GZip{}

type GZip struct {
	buf    *bytes.Buffer
	writer *gzip.Writer
	//reader *gzip.Reader
}

func (g *GZip) Code() byte {
	return 1
}
func (g *GZip) Compress(originBuff []byte) ([]byte, error) {
	g.writer.Reset(g.buf)
	defer func() {
		g.buf.Reset()
	}()

	leng, err := g.writer.Write(originBuff)
	if err != nil || leng == 0 {
		return nil, err
	}
	err = g.writer.Flush()
	if err != nil {
		return nil, err
	}
	err = g.writer.Close()
	if err != nil {
		return nil, err
	}
	return g.buf.Bytes(), nil
}

func (g *GZip) Uncompress(originBuff []byte) ([]byte, error) {
	buf := bytes.NewBuffer(originBuff)
	r, err := gzip.NewReader(buf)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(r)
}
