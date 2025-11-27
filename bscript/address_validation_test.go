package bscript_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bsv-blockchain/go-bt/v2/bscript"
)

func TestValidateAddress(t *testing.T) {
	t.Parallel()

	t.Run("mainnet P2PKH", func(t *testing.T) {
		ok, err := bscript.ValidateAddress("114ZWApV4EEU8frr7zygqQcB1V2BodGZuS")
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("testnet P2PKH", func(t *testing.T) {
		ok, err := bscript.ValidateAddress("mfaWoDuTsFfiunLTqZx4fKpVsUctiDV9jk")
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("BIP276", func(t *testing.T) {
		ok, err := bscript.ValidateAddress("bitcoin-script:0101522102e5b3f2970648b5592b7303367ab7d7d49e6e27dd80c7b5da18a22dac67a51a322103da6bf6a0c1a06ae7c4091542e0eaa29f2678e7957b78ba09cbe5a36241a4ad0452aeb245ccc7")
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("empty address", func(t *testing.T) {
		ok, err := bscript.ValidateAddress("")
		require.Error(t, err)
		assert.False(t, ok)
	})

	t.Run("empty script", func(t *testing.T) {
		ok, err := bscript.ValidateAddress("bitcoin-script:")
		require.Error(t, err)
		assert.False(t, ok)
	})

	t.Run("invalid address", func(t *testing.T) {
		ok, err := bscript.ValidateAddress("invalid")
		require.Error(t, err)
		assert.False(t, ok)
	})

	t.Run("invalid script", func(t *testing.T) {
		ok, err := bscript.ValidateAddress("bitcoin-script:invalid")
		require.Error(t, err)
		assert.False(t, ok)
	})
}
