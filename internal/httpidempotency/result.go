package httpidempotency

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net/http"
	"slices"
	"strings"
)

// Result is the bounded semantic response an operation may retain and replay.
type Result struct {
	Status    int
	MediaType string
	Codec     string
	Headers   http.Header
	Payload   []byte
}

type encodedHeader struct {
	name   string
	values []string
}

// EncodeResult returns the one versioned representation the Store may retain.
func EncodeResult(contract Contract, result Result) ([]byte, error) {
	if err := contract.Validate(); err != nil {
		return nil, err
	}
	if result.Status < 200 || result.Status >= 300 || !slices.Contains(contract.ReplayStatuses, result.Status) {
		return nil, fmt.Errorf("idempotency result: status %d is not replayable", result.Status)
	}
	if strings.TrimSpace(result.MediaType) == "" || strings.TrimSpace(result.Codec) == "" {
		return nil, errors.New("idempotency result: media type and codec are required")
	}
	if !slices.Contains(contract.ResultCodecs, result.Codec) {
		return nil, fmt.Errorf("idempotency result: codec %q is not declared", result.Codec)
	}
	headers, err := encodeResultHeaders(contract, result.Headers)
	if err != nil {
		return nil, err
	}

	encoded := make([]byte, 0, len(result.Payload)+128)
	encoded = append(encoded, "http-idempotency.result.v1"...)
	encoded = append(encoded, 0)
	var status [2]byte
	binary.BigEndian.PutUint16(status[:], uint16(result.Status))
	encoded = append(encoded, status[:]...)
	encoded, err = appendBytes(encoded, result.MediaType)
	if err != nil {
		return nil, err
	}
	encoded, err = appendBytes(encoded, result.Codec)
	if err != nil {
		return nil, err
	}
	var count [2]byte
	binary.BigEndian.PutUint16(count[:], uint16(len(headers))) // #nosec G115 -- header count is rejected above math.MaxUint16.
	encoded = append(encoded, count[:]...)
	for _, header := range headers {
		encoded, err = appendBytes(encoded, header.name)
		if err != nil {
			return nil, err
		}
		binary.BigEndian.PutUint16(count[:], uint16(len(header.values))) // #nosec G115 -- header value count is rejected above math.MaxUint16.
		encoded = append(encoded, count[:]...)
		for _, value := range header.values {
			encoded, err = appendBytes(encoded, value)
			if err != nil {
				return nil, err
			}
		}
	}
	encoded, err = appendBytes(encoded, string(result.Payload))
	if err != nil {
		return nil, err
	}
	if len(encoded) > contract.ResultMaxBytes {
		return nil, fmt.Errorf("idempotency result: encoded result exceeds %d bytes", contract.ResultMaxBytes)
	}
	return encoded, nil
}

func encodeResultHeaders(contract Contract, resultHeaders http.Header) ([]encodedHeader, error) {
	allowed := make(map[string]struct{}, len(contract.StableHeaders))
	for _, header := range contract.StableHeaders {
		allowed[header] = struct{}{}
	}
	headers := make([]encodedHeader, 0, len(resultHeaders))
	seen := make(map[string]struct{}, len(resultHeaders))
	for name, values := range resultHeaders {
		lowerName := strings.ToLower(name)
		if _, ok := allowed[lowerName]; !ok || forbiddenResultHeader(lowerName) || !validToken(lowerName) {
			return nil, fmt.Errorf("idempotency result: header %q is not replayable", name)
		}
		if _, duplicate := seen[lowerName]; duplicate {
			return nil, fmt.Errorf("idempotency result: header %q is duplicated", name)
		}
		seen[lowerName] = struct{}{}
		if len(values) == 0 || len(values) > math.MaxUint16 {
			return nil, fmt.Errorf("idempotency result: header %q has invalid values", name)
		}
		for _, value := range values {
			if !validHeaderValue(value) {
				return nil, fmt.Errorf("idempotency result: header %q has invalid value", name)
			}
		}
		headers = append(headers, encodedHeader{name: lowerName, values: values})
	}
	if len(headers) > math.MaxUint16 {
		return nil, errors.New("idempotency result: too many headers")
	}
	slices.SortFunc(headers, func(a, b encodedHeader) int { return strings.Compare(a.name, b.name) })
	return headers, nil
}

// DecodeResult checks and decodes a retained result before HTTP re-renders it.
func DecodeResult(contract Contract, encoded []byte) (Result, error) {
	if err := contract.Validate(); err != nil {
		return Result{}, err
	}
	if len(encoded) > contract.ResultMaxBytes || !strings.HasPrefix(string(encoded), "http-idempotency.result.v1\x00") {
		return Result{}, errors.New("idempotency result: invalid envelope")
	}
	reader := resultReader{data: encoded[len("http-idempotency.result.v1")+1:]}
	status, err := reader.uint16()
	if err != nil {
		return Result{}, err
	}
	mediaType, err := reader.string()
	if err != nil {
		return Result{}, err
	}
	codec, err := reader.string()
	if err != nil {
		return Result{}, err
	}
	headers, err := decodeResultHeaders(contract, &reader)
	if err != nil {
		return Result{}, err
	}
	payload, err := reader.bytes()
	if err != nil || !reader.done() {
		return Result{}, errors.New("idempotency result: invalid retained payload")
	}
	result := Result{Status: int(status), MediaType: mediaType, Codec: codec, Headers: headers, Payload: payload}
	if _, err := EncodeResult(contract, result); err != nil {
		return Result{}, err
	}
	return result, nil
}

func decodeResultHeaders(contract Contract, reader *resultReader) (http.Header, error) {
	count, err := reader.uint16()
	if err != nil {
		return nil, err
	}
	headers := make(http.Header, count)
	seen := make(map[string]struct{}, count)
	for range count {
		name, err := reader.string()
		if err != nil {
			return nil, err
		}
		if name != strings.ToLower(name) || !slices.Contains(contract.StableHeaders, name) || forbiddenResultHeader(name) || !validToken(name) {
			return nil, errors.New("idempotency result: invalid retained header")
		}
		if _, duplicate := seen[name]; duplicate {
			return nil, errors.New("idempotency result: duplicate retained header")
		}
		valuesCount, err := reader.uint16()
		if err != nil || valuesCount == 0 {
			return nil, errors.New("idempotency result: invalid retained header values")
		}
		values := make([]string, 0, valuesCount)
		for range valuesCount {
			value, err := reader.string()
			if err != nil || !validHeaderValue(value) {
				return nil, errors.New("idempotency result: invalid retained header value")
			}
			values = append(values, value)
		}
		seen[name] = struct{}{}
		headers[http.CanonicalHeaderKey(name)] = values
	}
	return headers, nil
}

func appendBytes(dst []byte, value string) ([]byte, error) {
	if len(value) > math.MaxUint32 {
		return nil, errors.New("idempotency result: field exceeds uint32 length")
	}
	var length [4]byte
	binary.BigEndian.PutUint32(length[:], uint32(len(value))) // #nosec G115 -- field length is rejected above math.MaxUint32.
	dst = append(dst, length[:]...)
	return append(dst, value...), nil
}

type resultReader struct {
	data []byte
}

func (r *resultReader) uint16() (uint16, error) {
	if len(r.data) < 2 {
		return 0, errors.New("idempotency result: truncated envelope")
	}
	value := binary.BigEndian.Uint16(r.data[:2])
	r.data = r.data[2:]
	return value, nil
}

func (r *resultReader) bytes() ([]byte, error) {
	if len(r.data) < 4 {
		return nil, errors.New("idempotency result: truncated envelope")
	}
	length := binary.BigEndian.Uint32(r.data[:4])
	r.data = r.data[4:]
	if uint64(length) > uint64(len(r.data)) {
		return nil, errors.New("idempotency result: truncated envelope")
	}
	value := append([]byte(nil), r.data[:length]...)
	r.data = r.data[length:]
	return value, nil
}

func (r *resultReader) string() (string, error) {
	value, err := r.bytes()
	return string(value), err
}

func (r *resultReader) done() bool {
	return len(r.data) == 0
}

func validHeaderValue(value string) bool {
	return !strings.ContainsAny(value, "\r\n\x00")
}
