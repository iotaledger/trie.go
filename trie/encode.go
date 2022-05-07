package trie

import (
	"errors"
	"github.com/iotaledger/trie.go"
)

var (
	ErrWrongNibble      = errors.New("key16 byte must be less than 0x0F")
	ErrEmpty            = errors.New("encoded key16 can't be empty")
	ErrWrongFormat      = errors.New("encoded key16 wrong format")
	ErrWrongBinaryValue = errors.New("key2 byte must be 1 or 0")
	ErrWrongArity       = errors.New("arity value must be 1, 15 or 255")
)

// unpack16 src places each 4 bit nibble into separate byte
func unpack16(dst, src []byte) []byte {
	for _, c := range src {
		dst = append(dst, c>>4, c&0x0F)
	}
	return dst
}

// pack16 places to 4 bit nibbles into one byte. I case of odd number of nibbles,
// the 4 lower bit in the last byte remains 0
func pack16(dst, src []byte) ([]byte, error) {
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

// encode16 packs nibbles and prefixes it with number of excess bytes (0 or 1)
func encode16(k16 []byte) ([]byte, error) {
	ret := append(make([]byte, 0, len(k16)/2+1), byte(len(k16)%2))
	return pack16(ret, k16)
}

func decode16(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, ErrEmpty
	}
	if data[0] > 1 {
		return nil, ErrWrongFormat
	}
	ret := make([]byte, 0, len(data)*2)
	ret = unpack16(ret, data[1:])
	if data[0] == 1 && ret[len(ret)-1] != 0 {
		// enforce padding with 0
		return nil, ErrWrongFormat
	}
	// cut the excess byte
	ret = ret[:len(ret)-int(data[0])]
	return ret, nil
}

// unpack2 src places each bit into separate byte. Bigendian
func unpack2(dst, src []byte) []byte {
	for _, c := range src {
		for i := 7; i >= 0; i-- {
			m := byte(0x01) << i
			if c&(m) != 0 {
				dst = append(dst, 1)
			} else {
				dst = append(dst, 0)
			}
		}
	}
	return dst
}

// pack2 places each 8 bytes with 0/1 into byte (bigendian). The last are padded with 0 if necessary
func pack2(dst, src []byte) ([]byte, error) {
	for i := 0; i < len(src); i += 8 {
		c := byte(0)
		for j := 0; j < 8; j++ {
			if i+j >= len(src) {
				break
			}
			switch src[i+j] {
			case 1:
				c |= 0x80 >> j
			case 0:
			default:
				return nil, ErrWrongBinaryValue
			}
		}
		dst = append(dst, c)
	}
	return dst, nil
}

// encode2 packs binary values and prefixes it with number of padded bits
func encode2(k2 []byte) ([]byte, error) {
	padded := byte(len(k2) % 8)
	if padded != 0 {
		padded = 8 - padded
	}
	ret := append(make([]byte, 0, len(k2)/8+1), padded)
	return pack2(ret, k2)
}

// decode2 decodes to bit array
func decode2(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, ErrEmpty
	}
	if data[0] > 7 {
		return nil, ErrWrongFormat
	}
	ret := make([]byte, 0, len(data)*8)
	ret = unpack2(ret, data[1:])
	trie_go.Assert(len(ret) >= int(data[0]), "len(ret) >= int(data[0])")
	// enforce the last data[0] elements are 0
	for j := len(ret) - int(data[0]); j < len(ret); j++ {
		if ret[j] != 0 {
			return nil, ErrWrongFormat
		}
	}
	ret = ret[:len(ret)-int(data[0])]
	return ret, nil
}

func UnpackBytes(src []byte, arity PathArity) []byte {
	switch arity {
	case PathArity256:
		return src
	case PathArity16:
		return unpack16(make([]byte, 0, 2*len(src)), src)
	case PathArity2:
		return unpack2(make([]byte, 0, 8*len(src)), src)
	}
	panic(ErrWrongArity)
}

func encodeUnpackedBytes(unpacked []byte, arity PathArity) ([]byte, error) {
	if len(unpacked) == 0 {
		return nil, nil
	}
	switch arity {
	case PathArity256:
		return unpacked, nil
	case PathArity16:
		return encode16(unpacked)
	case PathArity2:
		return encode2(unpacked)
	}
	return nil, ErrWrongArity
}

func mustEncodeUnpackedBytes(unpacked []byte, arity PathArity) []byte {
	ret, err := encodeUnpackedBytes(unpacked, arity)
	trie_go.Assert(err == nil, "%v", err)
	return ret
}

func decodeToUnpackedBytes(encoded []byte, arity PathArity) ([]byte, error) {
	switch arity {
	case PathArity256:
		return encoded, nil
	case PathArity16:
		return decode16(encoded)
	case PathArity2:
		return decode2(encoded)
	}
	return nil, ErrWrongArity
}
