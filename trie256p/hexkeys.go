package trie256p

import "errors"

// TODO WIP hex keys will be used to transparently optimize blake2b proof size at the expense of the databse size

// UnpackToHexKey results in even length
func UnpackToHexKey(key []byte) []byte {
	ret := make([]byte, len(key)*2)
	for i, c := range key {
		ret[2*i] = c >> 4
		ret[2*i+1] = c & 0x0F
	}
	return ret
}

// PackHexKeyToKey packs hex key, if possible.
// First byte of the packed representation contains 1 if hexkey has odd bytes, otherwise 0
// The rest of bytes are the packed key. In case it is odd, the low 4 bits of the last byte are not used
func PackHexKeyToKey(hexkey []byte) ([]byte, error) {
	isOdd := len(hexkey)%2 != 0
	var ret []byte
	if isOdd {
		ret = make([]byte, (len(hexkey)+1)/2+1)
	} else {
		ret = make([]byte, len(hexkey)/2+1)
	}
	if isOdd {
		ret[0] = 1
	}
	for i, c := range hexkey {
		if c > 0x0F {
			return nil, errors.New("hexkey byte must be less than 0x0F")
		}
		if i%2 == 0 {
			ret[i/2+1] = c << 4
		} else {
			ret[i/2+1] |= c
		}
	}
	return ret, nil
}
