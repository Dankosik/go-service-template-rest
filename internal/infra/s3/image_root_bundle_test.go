package s3

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"math/big"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestImageRootBundleLoaderIsStrictAndBounded(t *testing.T) {
	ca := testCACertificate(t, 1, true)

	t.Run("valid", func(t *testing.T) {
		bundle, err := loadImageRootBundle(memoryImageRootSource(ca, 0, nil))
		if err != nil {
			t.Fatalf("loadImageRootBundle() error = %v", err)
		}
		if bundle.pool == nil || bundle.bytes != int64(len(ca)) || bundle.roots != 1 {
			t.Fatalf("bundle = pool:%p bytes:%d roots:%d", bundle.pool, bundle.bytes, bundle.roots)
		}
	})

	t.Run("byte limit", func(t *testing.T) {
		exact := append(bytes.Clone(ca), bytes.Repeat([]byte{'\n'}, maxImageRootBundleBytes-len(ca))...)
		if _, err := loadImageRootBundle(memoryImageRootSource(exact, 0, nil)); err != nil {
			t.Fatalf("exact byte limit error = %v", err)
		}
		if _, err := loadImageRootBundle(memoryImageRootSource(append(exact, '\n'), 0, nil)); err == nil {
			t.Fatal("oversized bundle error = nil")
		}
	})

	t.Run("certificate limit", func(t *testing.T) {
		var roots []byte
		for serial := int64(1); serial <= maxImageRootCertificates+1; serial++ {
			roots = append(roots, testCACertificate(t, serial, true)...)
		}
		limit := bytes.LastIndex(roots, []byte("-----BEGIN CERTIFICATE-----"))
		if _, err := loadImageRootBundle(memoryImageRootSource(roots[:limit], 0, nil)); err != nil {
			t.Fatalf("exact certificate limit error = %v", err)
		}
		if _, err := loadImageRootBundle(memoryImageRootSource(roots, 0, nil)); err == nil {
			t.Fatal("excess certificate count error = nil")
		}
	})

	invalid := []struct {
		name string
		data []byte
		mode fs.FileMode
	}{
		{name: "empty"},
		{name: "not regular", data: ca, mode: fs.ModeDir},
		{name: "duplicate", data: append(bytes.Clone(ca), ca...)},
		{name: "non CA", data: testCACertificate(t, 2, false)},
		{name: "wrong PEM type", data: pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("not a key")})},
		{name: "PEM headers", data: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Headers: map[string]string{"Proc-Type": "4,ENCRYPTED"}, Bytes: []byte("invalid")})},
		{name: "invalid DER", data: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("invalid")})},
		{name: "trailing data", data: append(bytes.Clone(ca), []byte("not PEM")...)},
	}
	for _, test := range invalid {
		t.Run(test.name, func(t *testing.T) {
			if _, err := loadImageRootBundle(memoryImageRootSource(test.data, test.mode, nil)); err == nil {
				t.Fatal("loadImageRootBundle() error = nil")
			}
		})
	}

	t.Run("source errors", func(t *testing.T) {
		if _, err := loadImageRootBundle(func() (fs.File, error) { return nil, errors.New("open") }); err == nil {
			t.Fatal("open error was not returned")
		}
		if _, err := loadImageRootBundle(memoryImageRootSource(ca, 0, errors.New("stat"))); err == nil {
			t.Fatal("stat error was not returned")
		}
		if _, err := loadImageRootBundle(func() (fs.File, error) {
			return readErrorImageRootFile{info: memoryImageRootInfo{size: int64(len(ca))}}, nil
		}); err == nil {
			t.Fatal("read error was not returned")
		}
	})

	t.Run("source changes", func(t *testing.T) {
		if _, err := loadImageRootBundle(func() (fs.File, error) {
			return &memoryImageRootFile{Reader: bytes.NewReader(ca), info: memoryImageRootInfo{size: int64(len(ca) + 1)}}, nil
		}); err == nil {
			t.Fatal("changed source error = nil")
		}
	})
}

func TestFinalImageRootBundleReceipt(t *testing.T) {
	path := os.Getenv("S3_IMAGE_ROOT_BUNDLE_RECEIPT_PATH")
	if path == "" {
		t.Skip("S3_IMAGE_ROOT_BUNDLE_RECEIPT_PATH is not set")
	}

	wantBytes := receiptInt64(t, "S3_IMAGE_ROOT_BUNDLE_RECEIPT_BYTES")
	wantRoots := int(receiptInt64(t, "S3_IMAGE_ROOT_BUNDLE_RECEIPT_ROOTS"))
	wantSHA256 := os.Getenv("S3_IMAGE_ROOT_BUNDLE_RECEIPT_SHA256")
	if len(wantSHA256) != sha256.Size*2 {
		t.Fatalf("S3_IMAGE_ROOT_BUNDLE_RECEIPT_SHA256 = %q", wantSHA256)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(data)) != wantBytes {
		t.Fatalf("bundle bytes = %d, want %d", len(data), wantBytes)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != wantSHA256 {
		t.Fatalf("bundle SHA-256 = %s, want %s", got, wantSHA256)
	}

	bundle, err := loadImageRootBundle(memoryImageRootSource(data, 0, nil))
	if err != nil {
		t.Fatalf("loadImageRootBundle() error = %v", err)
	}
	if bundle.bytes != wantBytes || bundle.roots != wantRoots {
		t.Fatalf("strict loader = bytes:%d roots:%d, want bytes:%d roots:%d", bundle.bytes, bundle.roots, wantBytes, wantRoots)
	}
}

func receiptInt64(t *testing.T, key string) int64 {
	t.Helper()
	value, err := strconv.ParseInt(os.Getenv(key), 10, 64)
	if err != nil || value <= 0 {
		t.Fatalf("%s = %q", key, os.Getenv(key))
	}
	return value
}

func testImageRootSource(t *testing.T) imageRootSource {
	t.Helper()
	return memoryImageRootSource(testCACertificate(t, 1, true), 0, nil)
}

func testCACertificate(t *testing.T, serial int64, isCA bool) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(serial),
		Subject:               pkix.Name{CommonName: "T9A test root"},
		NotBefore:             time.Unix(0, 0),
		NotAfter:              time.Unix(1<<31, 0),
		BasicConstraintsValid: true,
		IsCA:                  isCA,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func envelopeRootBundle(t *testing.T) (bundle []byte, root *x509.Certificate, signer crypto.Signer) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	for serial := int64(1); serial < maxImageRootCertificates-2; serial++ {
		if serial == 1 {
			certificate, pemBytes := envelopeRootCertificate(t, key, serial, nil)
			root = certificate
			bundle = append(bundle, pemBytes...)
			continue
		}
		bundle = append(bundle, envelopeRootPEM(t, key, serial, nil)...)
	}

	target := maxImageRootBundleBytes - len(bundle)
	first := envelopeRootPEM(t, key, maxImageRootCertificates-2, []byte{'x'})
	second := envelopeRootPEM(t, key, maxImageRootCertificates-1, []byte{'x'})
	last := envelopeRootPEMAtMost(t, key, maxImageRootCertificates, target-len(first)-len(second))
	bundle = append(bundle, first...)
	bundle = append(bundle, second...)
	bundle = append(bundle, last...)
	if len(bundle) != maxImageRootBundleBytes {
		t.Fatalf("envelope bundle bytes = %d, want %d", len(bundle), maxImageRootBundleBytes)
	}
	if root == nil {
		t.Fatal("envelope bundle root is nil")
	}
	return bundle, root, key
}

func envelopeRootPEMAtMost(t *testing.T, signer crypto.Signer, serial int64, target int) []byte {
	t.Helper()
	if target <= 0 {
		t.Fatal("envelope PEM target must be positive")
	}
	low, high := 0, target
	var best []byte
	for low <= high {
		payload := low + (high-low)/2
		pemBytes := envelopeRootPEM(t, signer, serial, bytes.Repeat([]byte{'x'}, payload))
		if len(pemBytes) <= target {
			best = pemBytes
			low = payload + 1
			continue
		}
		high = payload - 1
	}
	if best == nil {
		t.Fatal("cannot create a bounded envelope PEM")
	}
	return best
}

func envelopeRootCertificate(t *testing.T, signer crypto.Signer, serial int64, payload []byte) (*x509.Certificate, []byte) {
	t.Helper()
	der := envelopeRootDER(t, signer, serial, payload)
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return certificate, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func envelopeRootPEM(t *testing.T, signer crypto.Signer, serial int64, payload []byte) []byte {
	t.Helper()
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: envelopeRootDER(t, signer, serial, payload)})
}

func envelopeRootDER(t *testing.T, signer crypto.Signer, serial int64, payload []byte) []byte {
	t.Helper()
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(serial),
		Subject:               pkix.Name{CommonName: "T9 envelope root"},
		NotBefore:             time.Unix(0, 0),
		NotAfter:              time.Unix(1<<31, 0),
		BasicConstraintsValid: true,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	if len(payload) != 0 {
		template.ExtraExtensions = []pkix.Extension{{Id: []int{1, 2, 3, 4, 5}, Value: payload}}
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, signer.Public(), signer)
	if err != nil {
		t.Fatal(err)
	}
	return der
}

func memoryImageRootSource(data []byte, mode fs.FileMode, statErr error) imageRootSource {
	return func() (fs.File, error) {
		return &memoryImageRootFile{
			Reader:  bytes.NewReader(data),
			info:    memoryImageRootInfo{size: int64(len(data)), mode: mode},
			statErr: statErr,
		}, nil
	}
}

type memoryImageRootFile struct {
	*bytes.Reader

	info    memoryImageRootInfo
	statErr error
}

func (f *memoryImageRootFile) Close() error { return nil }

func (f *memoryImageRootFile) Stat() (fs.FileInfo, error) { return f.info, f.statErr }

type memoryImageRootInfo struct {
	size int64
	mode fs.FileMode
}

func (i memoryImageRootInfo) Name() string       { return "ca-certificates.crt" }
func (i memoryImageRootInfo) Size() int64        { return i.size }
func (i memoryImageRootInfo) Mode() fs.FileMode  { return i.mode }
func (i memoryImageRootInfo) ModTime() time.Time { return time.Time{} }
func (i memoryImageRootInfo) IsDir() bool        { return false }
func (i memoryImageRootInfo) Sys() any           { return nil }

type readErrorImageRootFile struct{ info memoryImageRootInfo }

func (readErrorImageRootFile) Read([]byte) (int, error)     { return 0, errors.New("read") }
func (readErrorImageRootFile) Close() error                 { return nil }
func (f readErrorImageRootFile) Stat() (fs.FileInfo, error) { return f.info, nil }
