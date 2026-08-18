package s3

import (
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
)

const (
	imageRootBundlePath      = "/etc/ssl/certs/ca-certificates.crt"
	maxImageRootBundleBytes  = 448 << 10
	maxImageRootCertificates = 288
)

type imageRootSource func() (fs.File, error)

type imageRootBundle struct {
	pool  *x509.CertPool
	bytes int64
	roots int
}

func productionImageRootSource() (fs.File, error) {
	file, err := os.Open(imageRootBundlePath)
	if err != nil {
		return nil, fmt.Errorf("open image root bundle: %w", err)
	}
	return file, nil
}

//nolint:cyclop // Each branch is a required fail-closed image-bundle validation.
func loadImageRootBundle(source imageRootSource) (imageRootBundle, error) {
	file, err := source()
	if err != nil {
		return imageRootBundle{}, fmt.Errorf("open S3 image root bundle: %w", err)
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return imageRootBundle{}, fmt.Errorf("stat S3 image root bundle: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxImageRootBundleBytes {
		return imageRootBundle{}, errors.New("load S3 image root bundle: source must be one non-empty bounded regular file")
	}

	data, err := io.ReadAll(io.LimitReader(file, maxImageRootBundleBytes+1))
	if err != nil {
		return imageRootBundle{}, fmt.Errorf("read S3 image root bundle: %w", err)
	}
	if int64(len(data)) != info.Size() || len(data) > maxImageRootBundleBytes {
		return imageRootBundle{}, errors.New("load S3 image root bundle: source changed or exceeded the byte limit")
	}

	pool := x509.NewCertPool()
	seen := make(map[[sha256.Size]byte]struct{})
	rest := data
	for {
		rest = bytes.TrimLeft(rest, " \t\r\n")
		if len(rest) == 0 {
			break
		}
		if !bytes.HasPrefix(rest, []byte("-----BEGIN CERTIFICATE-----")) {
			return imageRootBundle{}, errors.New("load S3 image root bundle: non-certificate data is present")
		}
		block, remaining := pem.Decode(rest)
		if block == nil || block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
			return imageRootBundle{}, errors.New("load S3 image root bundle: invalid PEM certificate block")
		}
		certificate, parseErr := x509.ParseCertificate(block.Bytes)
		if parseErr != nil || !certificate.BasicConstraintsValid || !certificate.IsCA {
			return imageRootBundle{}, errors.New("load S3 image root bundle: invalid CA certificate")
		}
		identity := sha256.Sum256(certificate.Raw)
		if _, duplicate := seen[identity]; duplicate {
			return imageRootBundle{}, errors.New("load S3 image root bundle: duplicate CA certificate")
		}
		seen[identity] = struct{}{}
		if len(seen) > maxImageRootCertificates {
			return imageRootBundle{}, errors.New("load S3 image root bundle: certificate count exceeds the limit")
		}
		pool.AddCert(certificate)
		rest = remaining
	}
	if len(seen) == 0 {
		return imageRootBundle{}, errors.New("load S3 image root bundle: no CA certificate")
	}

	return imageRootBundle{pool: pool, bytes: int64(len(data)), roots: len(seen)}, nil
}
