package trie256p

import (
	"errors"
)

var (
	ErrWrongNibble = errors.New("key16 byte must be less than 0x0F")
	ErrEmpty       = errors.New("encoded key16 can't be empty")
	ErrWrongFormat = errors.New("encoded key16 wrong format")
)

// TODO WIP hex keys will be used to transparently optimize blake2b proof size

//
func unpackK16(dst, src []byte) []byte {
	for _, c := range src {
		dst = append(dst, c>>4, c&0x0F)
	}
	return dst
}

func packK16(dst, src []byte) ([]byte, error) {
	for i := 0; i < len(src); i += 2 {
		if src[i] > 0x0F {
			return nil, ErrWrongNibble
		}
		c := src[i] << 4
		if i+1 < len(src) {
			c |= src[i+1]
		}
		dst = append(dst, c)
	}
	return dst, nil
}

func encodeK16(k16 []byte) ([]byte, error) {
	ret := make([]byte, 0, len(k16)%2+1)
	if len(k16)%2 == 0 {
		ret = append(ret, 0x00)
	} else {
		ret = append(ret, 0xFF)
	}
	return packK16(ret, k16)
}

func decodeK16(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, ErrEmpty
	}
	if data[0] != 0 && data[0] != 0xFF {
		return nil, ErrWrongFormat
	}
	ret := make([]byte, 0, len(data)*2)
	ret = unpackK16(ret, data[1:])
	if data[0] == 0xFF {
		if ret[len(ret)-1] != 0 {
			return nil, ErrWrongFormat
		}
		ret = ret[:len(ret)-1]
	}
	return ret, nil
}
