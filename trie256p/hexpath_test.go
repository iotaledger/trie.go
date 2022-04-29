package trie256p

import (
	"encoding/hex"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHexKeysEmpty(t *testing.T) {
	var key []byte
	k16 := make([]byte, 0, 2*len(key))
	k16 = unpackK16(k16, key)
	t.Logf("key = %v", hex.EncodeToString(key))
	t.Logf("k16 = %v", hex.EncodeToString(k16))

	enc, err := encodeK16(k16)
	require.NoError(t, err)
	t.Logf("enc = %v", hex.EncodeToString(enc))

	dec, err := decodeK16(enc)
	require.NoError(t, err)
	t.Logf("dec = %v", hex.EncodeToString(dec))
	require.EqualValues(t, k16, dec)
}

func TestHexKeys1(t *testing.T) {
	key := []byte{0x31, 0x32, 0x33, 0x34, 0x35}
	k16 := make([]byte, 0, 2*len(key))
	k16 = unpackK16(k16, key)
	t.Logf("key = %v", hex.EncodeToString(key))
	t.Logf("k16 = %v", hex.EncodeToString(k16))

	enc, err := encodeK16(k16)
	require.NoError(t, err)
	t.Logf("enc = %v", hex.EncodeToString(enc))

	cut1 := k16[:3]
	cut2 := k16[3:]
	cut3 := k16[:6]

	t.Logf("cut1 = %v", hex.EncodeToString(cut1))
	t.Logf("cut2 = %v", hex.EncodeToString(cut2))
	t.Logf("cut3 = %v", hex.EncodeToString(cut3))

	enc1, err := encodeK16(cut1)
	require.NoError(t, err)
	t.Logf("enc1 = %v", hex.EncodeToString(enc1))

	enc2, err := encodeK16(cut2)
	require.NoError(t, err)
	t.Logf("enc2 = %v", hex.EncodeToString(enc2))

	enc3, err := encodeK16(cut3)
	require.NoError(t, err)
	t.Logf("enc3 = %v", hex.EncodeToString(enc3))

	dec1, err := decodeK16(enc1)
	require.NoError(t, err)
	t.Logf("dec1 = %v", hex.EncodeToString(dec1))
	require.EqualValues(t, cut1, dec1)

	dec2, err := decodeK16(enc2)
	require.NoError(t, err)
	t.Logf("dec2 = %v", hex.EncodeToString(dec2))
	require.EqualValues(t, cut2, dec2)

	dec3, err := decodeK16(enc3)
	require.NoError(t, err)
	t.Logf("dec3 = %v", hex.EncodeToString(dec3))
	require.EqualValues(t, cut3, dec3)
}
