
package packet

import (
	"compress/bzip2"
	"compress/flate"
	"compress/zlib"
	"golang.org/x/crypto/openpgp/errors"
	"io"
	"strconv"
)

type Compressed struct {
	Body io.Reader
}

const (
	NoCompression      = flate.NoCompression
	BestSpeed          = flate.BestSpeed
	BestCompression    = flate.BestCompression
	DefaultCompression = flate.DefaultCompression
)

type CompressionConfig struct {

	Level int
}

func (c *Compressed) parse(r io.Reader) error {
	var buf [1]byte
	_, err := readFull(r, buf[:])
	if err != nil {
		return err
	}

	switch buf[0] {
	case 1:
		c.Body = flate.NewReader(r)
	case 2:
		c.Body, err = zlib.NewReader(r)
	case 3:
		c.Body = bzip2.NewReader(r)
	default:
		err = errors.UnsupportedError("unknown compression algorithm: " + strconv.Itoa(int(buf[0])))
	}

	return err
}

type compressedWriteCloser struct {
	sh io.Closer      
	c  io.WriteCloser 
}

func (cwc compressedWriteCloser) Write(p []byte) (int, error) {
	return cwc.c.Write(p)
}

func (cwc compressedWriteCloser) Close() (err error) {
	err = cwc.c.Close()
	if err != nil {
		return err
	}

	return cwc.sh.Close()
}

func SerializeCompressed(w io.WriteCloser, algo CompressionAlgo, cc *CompressionConfig) (literaldata io.WriteCloser, err error) {
	compressed, err := serializeStreamHeader(w, packetTypeCompressed)
	if err != nil {
		return
	}

	_, err = compressed.Write([]byte{uint8(algo)})
	if err != nil {
		return
	}

	level := DefaultCompression
	if cc != nil {
		level = cc.Level
	}

	var compressor io.WriteCloser
	switch algo {
	case CompressionZIP:
		compressor, err = flate.NewWriter(compressed, level)
	case CompressionZLIB:
		compressor, err = zlib.NewWriterLevel(compressed, level)
	default:
		s := strconv.Itoa(int(algo))
		err = errors.UnsupportedError("Unsupported compression algorithm: " + s)
	}
	if err != nil {
		return
	}

	literaldata = compressedWriteCloser{compressed, compressor}

	return
}
