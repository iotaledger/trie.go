package trie256p

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHexKeys1(t *testing.T) {
	key := []byte{1, 2, 3, 4, 5}
	hexkey := UnpackToHexKey(key)
	t.Logf("key = %v", key)
	t.Logf("key = %v", hexkey)

	keyBack, err := PackHexKeyToKey(hexkey)
	require.NoError(t, err)
	require.True(t, keyBack[0] == 0)
	require.True(t, bytes.Equal(key, keyBack[1:]))
}

func TestHexKeys2(t *testing.T) {
	key := []byte{0x11, 0x12, 0x13, 0x14, 0x15}
	hexkey := UnpackToHexKey(key)
	t.Logf("key = %v", key)
	t.Logf("key = %v", hexkey)

	hexkeyOdd := hexkey[:3]
	t.Logf("keyOdd = %v", hexkeyOdd)

	keyBackOdd, err := PackHexKeyToKey(hexkeyOdd)
	t.Logf("keyBackOdd = %v", keyBackOdd)
	require.NoError(t, err)
	require.True(t, keyBackOdd[0] == 1)
	require.True(t, bytes.Equal([]byte{0x11, 0x10}, keyBackOdd[1:]))
}
