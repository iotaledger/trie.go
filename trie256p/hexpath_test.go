package trie256p

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHexKeys1(t *testing.T) {
	key := []byte{1, 2, 3, 4, 5}
	k256 := key256(key)
	k16 := k256.ToKey16()
	t.Logf("key = %v", key)
	t.Logf("key = %v", k256)
	t.Logf("key = %v", k16)

	ek256 := k256.encodePath()
	require.EqualValues(t, k256, ek256)
	require.EqualValues(t, key, ek256)

	ek16, err := k16.encodePath()
	require.NoError(t, err)
	require.True(t, ek16[0] == 0)
	require.True(t, bytes.Equal(key, ek16[1:]))
}
