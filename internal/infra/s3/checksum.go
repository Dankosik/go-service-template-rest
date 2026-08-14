package s3

import (
	"encoding/base64"
	"encoding/binary"
	"hash"
	"hash/crc64"
	"io"
)

const crc64NVMEPolynomial = 0x9a6c9329ac4bc9b5

var crc64NVMETable = crc64.MakeTable(crc64NVMEPolynomial)

type exactChecksumReader struct {
	source    io.Reader
	remaining int64
	hash      io.Writer
}

func newExactChecksumReader(source io.Reader, length int64, checksum io.Writer) *exactChecksumReader {
	return &exactChecksumReader{source: source, remaining: length, hash: checksum}
}

func (r *exactChecksumReader) Read(p []byte) (int, error) {
	if r.remaining == 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > r.remaining {
		p = p[:int(r.remaining)]
	}
	n, err := r.source.Read(p)
	if n > 0 {
		r.remaining -= int64(n)
		_, _ = r.hash.Write(p[:n])
	}
	if n == 0 && err == nil {
		return 0, io.ErrNoProgress
	}
	return n, err //nolint:wrapcheck // Reader callers require the source EOF identity.
}

func (r *exactChecksumReader) complete() error {
	if r.remaining != 0 {
		return io.ErrUnexpectedEOF
	}
	return nil
}

func crc64NVME() hash.Hash64 {
	return crc64.New(crc64NVMETable)
}

func crc64NVMEBase64(checksum hash.Hash64) string {
	var raw [8]byte
	binary.BigEndian.PutUint64(raw[:], checksum.Sum64())
	return base64.StdEncoding.EncodeToString(raw[:])
}

func matchingCRC64NVME(expected string, actual *string) bool {
	return actual != nil && *actual == expected
}
