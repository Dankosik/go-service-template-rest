package postgreswebhook

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"slices"
)

func canonicalRecord(tag string, fields ...[]byte) ([]byte, error) {
	size := len(tag) + 1
	for _, field := range fields {
		size += 4 + len(field)
	}
	out := make([]byte, 0, size)
	out = append(out, tag...)
	out = append(out, 0)
	for _, field := range fields {
		length, err := uint32Value(len(field))
		if err != nil {
			return nil, err
		}
		out = binary.BigEndian.AppendUint32(out, length)
		out = append(out, field...)
	}
	return out, nil
}

func canonicalList(items [][]byte) ([]byte, error) {
	items = slices.Clone(items)
	slices.SortFunc(items, slices.Compare)
	for i := 1; i < len(items); i++ {
		if slices.Equal(items[i-1], items[i]) {
			return nil, errors.New("canonical list contains duplicate item")
		}
	}
	count, err := uint32Value(len(items))
	if err != nil {
		return nil, err
	}
	out := binary.BigEndian.AppendUint32(nil, count)
	for _, item := range items {
		length, err := uint32Value(len(item))
		if err != nil {
			return nil, err
		}
		out = binary.BigEndian.AppendUint32(out, length)
		out = append(out, item...)
	}
	return out, nil
}

func canonicalDigest(data []byte) [sha256.Size]byte { return sha256.Sum256(data) }
