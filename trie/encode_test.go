package trie

import (
	"encoding/hex"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test16KeysEmpty(t *testing.T) {
	var key []byte
	k16 := make([]byte, 0, 2*len(key))
	k16 = unpack16(k16, key)
	t.Logf("unpackedKey = %v", hex.EncodeToString(key))
	t.Logf("k16 = %v", hex.EncodeToString(k16))

	enc, err := encode16(k16)
	require.NoError(t, err)
	t.Logf("enc = %v", hex.EncodeToString(enc))

	dec, err := decode16(enc)
	require.NoError(t, err)
	t.Logf("dec = %v", hex.EncodeToString(dec))
	require.EqualValues(t, k16, dec)
}

func Test16Keys1(t *testing.T) {
	key := []byte{0x31, 0x32, 0x33, 0x34, 0x35}
	k16 := make([]byte, 0, 2*len(key))
	k16 = unpack16(k16, key)
	t.Logf("unpackedKey = %v", hex.EncodeToString(key))
	t.Logf("k16 = %v", hex.EncodeToString(k16))

	enc, err := encode16(k16)
	require.NoError(t, err)
	t.Logf("enc = %v", hex.EncodeToString(enc))

	cut1 := k16[:3]
	cut2 := k16[3:]
	cut3 := k16[:6]

	t.Logf("cut1 = %v", hex.EncodeToString(cut1))
	t.Logf("cut2 = %v", hex.EncodeToString(cut2))
	t.Logf("cut3 = %v", hex.EncodeToString(cut3))

	enc1, err := encode16(cut1)
	require.NoError(t, err)
	t.Logf("enc1 = %v", hex.EncodeToString(enc1))

	enc2, err := encode16(cut2)
	require.NoError(t, err)
	t.Logf("enc2 = %v", hex.EncodeToString(enc2))

	enc3, err := encode16(cut3)
	require.NoError(t, err)
	t.Logf("enc3 = %v", hex.EncodeToString(enc3))

	dec1, err := decode16(enc1)
	require.NoError(t, err)
	t.Logf("dec1 = %v", hex.EncodeToString(dec1))
	require.EqualValues(t, cut1, dec1)

	dec2, err := decode16(enc2)
	require.NoError(t, err)
	t.Logf("dec2 = %v", hex.EncodeToString(dec2))
	require.EqualValues(t, cut2, dec2)

	dec3, err := decode16(enc3)
	require.NoError(t, err)
	t.Logf("dec3 = %v", hex.EncodeToString(dec3))
	require.EqualValues(t, cut3, dec3)
}

func Test2KeysEmpty(t *testing.T) {
	var key []byte
	k2 := make([]byte, 0, 8*len(key))
	k2 = unpack2(k2, key)
	t.Logf("unpackedKey = %v", hex.EncodeToString(key))
	t.Logf("k2 = %v", hex.EncodeToString(k2))

	enc, err := encode2(k2)
	require.NoError(t, err)
	t.Logf("enc = %v", hex.EncodeToString(enc))

	dec, err := decode2(enc)
	require.NoError(t, err)
	t.Logf("dec = %v", hex.EncodeToString(dec))
	require.EqualValues(t, k2, dec)
}

func Test2Keys1(t *testing.T) {
	key := []byte{0x31, 0x32, 0x33, 0x34, 0x35}
	t.Logf("unpackedKey = %v", hex.EncodeToString(key))
	k2 := make([]byte, 0, 8*len(key))
	k2 = unpack2(k2, key)
	t.Logf("k2 = %+v", k2)

	enc, err := encode2(k2)
	require.NoError(t, err)
	t.Logf("enc = %v", hex.EncodeToString(enc))

	cut1 := k2[:3]
	cut2 := k2[3:]
	cut3 := k2[:6]

	t.Logf("cut1 = %+v", cut1)
	t.Logf("cut2 = %+v", cut2)
	t.Logf("cut3 = %+v", cut3)

	enc1, err := encode2(cut1)
	require.NoError(t, err)
	t.Logf("enc1 = %s", hex.EncodeToString(enc1))

	enc2, err := encode2(cut2)
	require.NoError(t, err)
	t.Logf("enc2 = %s", hex.EncodeToString(enc2))

	enc3, err := encode2(cut3)
	require.NoError(t, err)
	t.Logf("enc3 = %s", hex.EncodeToString(enc3))

	dec1, err := decode2(enc1)
	require.NoError(t, err)
	t.Logf("dec1 = %s", hex.EncodeToString(dec1))
	require.EqualValues(t, cut1, dec1)

	dec2, err := decode2(enc2)
	require.NoError(t, err)
	t.Logf("dec2 = %s", hex.EncodeToString(dec2))
	require.EqualValues(t, cut2, dec2)

	dec3, err := decode2(enc3)
	require.NoError(t, err)
	t.Logf("dec3 = %s", hex.EncodeToString(dec3))
	require.EqualValues(t, cut3, dec3)
}
