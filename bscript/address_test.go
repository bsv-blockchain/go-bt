package bscript_test

import (
	"encoding/hex"
	"testing"

	"github.com/bsv-blockchain/go-bt/v2/bscript"
	bec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testPublicKeyHash = "00ac6144c4db7b5790f343cf0477a65fb8a02eb7"

func TestNewAddressFromString(t *testing.T) {
	t.Parallel()

	t.Run("mainnet", func(t *testing.T) {
		addressMain := "1E7ucTTWRTahCyViPhxSMor2pj4VGQdFMr"

		addr, err := bscript.NewAddressFromString(addressMain)
		require.NoError(t, err)
		assert.NotNil(t, addr)

		assert.Equal(t, "8fe80c75c9560e8b56ed64ea3c26e18d2c52211b", addr.PublicKeyHash, addressMain)
		assert.Equal(t, addressMain, addr.AddressString)
	})

	t.Run("testnet", func(t *testing.T) {
		addressTestnet := "mtdruWYVEV1wz5yL7GvpBj4MgifCB7yhPd"

		addr, err := bscript.NewAddressFromString(addressTestnet)
		require.NoError(t, err)
		assert.NotNil(t, addr)

		assert.Equal(t, "8fe80c75c9560e8b56ed64ea3c26e18d2c52211b", addr.PublicKeyHash, addressTestnet)
		assert.Equal(t, addressTestnet, addr.AddressString)
	})

	t.Run("short address", func(t *testing.T) {
		shortAddress := "ADD8E55"
		addr, err := bscript.NewAddressFromString(shortAddress)
		require.Error(t, err)
		assert.Nil(t, addr)
		require.EqualError(t, err, "invalid address length for '"+shortAddress+"'")
	})

	t.Run("unsupported address", func(t *testing.T) {
		unsupportedAddress := "27BvY7rFguYQvEL872Y7Fo77Y3EBApC2EK"
		addr, err := bscript.NewAddressFromString(unsupportedAddress)
		require.Error(t, err)
		assert.Nil(t, addr)
		require.EqualError(t, err, "address not supported "+unsupportedAddress)
	})
}

func TestNewAddressFromPublicKeyString(t *testing.T) {
	t.Parallel()

	t.Run("mainnet", func(t *testing.T) {
		addr, err := bscript.NewAddressFromPublicKeyString(
			"026cf33373a9f3f6c676b75b543180703df225f7f8edbffedc417718a8ad4e89ce",
			true,
		)
		require.NoError(t, err)
		assert.NotNil(t, addr)

		assert.Equal(t, testPublicKeyHash, addr.PublicKeyHash)
		assert.Equal(t, "114ZWApV4EEU8frr7zygqQcB1V2BodGZuS", addr.AddressString)
	})

	t.Run("testnet", func(t *testing.T) {
		addr, err := bscript.NewAddressFromPublicKeyString(
			"026cf33373a9f3f6c676b75b543180703df225f7f8edbffedc417718a8ad4e89ce",
			false,
		)
		require.NoError(t, err)
		assert.NotNil(t, addr)

		assert.Equal(t, testPublicKeyHash, addr.PublicKeyHash)
		assert.Equal(t, "mfaWoDuTsFfiunLTqZx4fKpVsUctiDV9jk", addr.AddressString)
	})
}

func TestNewAddressFromPublicKey(t *testing.T) {
	t.Parallel()

	pubKeyBytes, err := hex.DecodeString("026cf33373a9f3f6c676b75b543180703df225f7f8edbffedc417718a8ad4e89ce")
	require.NoError(t, err)

	var pubKey *bec.PublicKey
	pubKey, err = bec.ParsePubKey(pubKeyBytes)
	require.NoError(t, err)
	assert.NotNil(t, pubKey)

	var addr *bscript.Address
	addr, err = bscript.NewAddressFromPublicKey(pubKey, true)
	require.NoError(t, err)
	assert.NotNil(t, addr)

	assert.Equal(t, testPublicKeyHash, addr.PublicKeyHash)
	assert.Equal(t, "114ZWApV4EEU8frr7zygqQcB1V2BodGZuS", addr.AddressString)
}

func TestBase58EncodeMissingChecksum(t *testing.T) {
	t.Parallel()

	input, err := hex.DecodeString("0488b21e000000000000000000362f7a9030543db8751401c387d6a71e870f1895b3a62569d455e8ee5f5f5e5f03036624c6df96984db6b4e625b6707c017eb0e0d137cd13a0c989bfa77a4473fd")
	require.NoError(t, err)

	assert.Equal(t,
		"xpub661MyMwAqRbcF5ivRisXcZTEoy7d9DfLF6fLqpu5GWMfeUyGHuWJHVp5uexDqXTWoySh8pNx3ELW7qymwPNg3UEYHjwh1tpdm3P9J2j4g32",
		bscript.Base58EncodeMissingChecksum(input),
	)
}
