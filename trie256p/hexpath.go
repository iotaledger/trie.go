package trie256p

import (
	"encoding/hex"
	"errors"
	"fmt"
)

type key256 []byte
type key16 []byte
type encodedPath []byte

var ErrWrongNibble = errors.New("key16 byte must be less than 0x0F")

// TODO WIP hex keys will be used to transparently optimize blake2b proof size

func (k key256) ToKey16() key16 {
	ret := make([]byte, len(k)*2)
	for i, c := range k {
		ret[2*i] = c >> 4
		ret[2*i+1] = c & 0x0F
	}
	return ret
}

func (k key256) String() string {
	return fmt.Sprintf("key256(%s)", hex.EncodeToString(k))
}

func (k key256) encodePath() encodedPath {
	return encodedPath(k)
}

func (hk key16) encodePath() (encodedPath, error) {
	isOdd := len(hk)%2 != 0
	var ret []byte
	if isOdd {
		ret = make([]byte, (len(hk)+1)/2+1)
	} else {
		ret = make([]byte, len(hk)/2+1)
	}
	if isOdd {
		ret[0] = 1
	}
	for i, c := range hk {
		if c > 0x0F {
			return nil, ErrWrongNibble
		}
		if i%2 == 0 {
			ret[i/2+1] = c << 4
		} else {
			ret[i/2+1] |= c
		}
	}
	return ret, nil
}

func (hk key16) isValid() error {
	for _, c := range hk {
		if c&0xF0 != 0 {
			return ErrWrongNibble
		}
	}
	return nil
}

func (k key16) String() string {
	return fmt.Sprintf("key16(%s)", hex.EncodeToString(k))
}
