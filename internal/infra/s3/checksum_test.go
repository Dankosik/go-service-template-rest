package s3

import "testing"

func TestCRC64NVMEKnownAnswer(t *testing.T) {
	t.Parallel()
	checksum := crc64NVME()
	_, _ = checksum.Write([]byte("123456789"))
	if got := crc64NVMEBase64(checksum); got != "rosUhgp5mIg=" {
		t.Fatalf("CRC64NVME(123456789) = %q, want %q", got, "rosUhgp5mIg=")
	}
}
