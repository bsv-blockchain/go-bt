package bt_test

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bsv-blockchain/go-bt/v2"
	"github.com/bsv-blockchain/go-bt/v2/bscript"
)

// BenchmarkBytes benchmarks the Bytes method of a transaction.
func BenchmarkBytes(b *testing.B) {
	tx, _ := bt.NewTxFromString("0200000003a9bc457fdc6a54d99300fb137b23714d860c350a9d19ff0f571e694a419ff3a0010000006b48304502210086c83beb2b2663e4709a583d261d75be538aedcafa7766bd983e5c8db2f8b2fc02201a88b178624ab0ad1748b37c875f885930166237c88f5af78ee4e61d337f935f412103e8be830d98bb3b007a0343ee5c36daa48796ae8bb57946b1e87378ad6e8a090dfeffffff0092bb9a47e27bf64fc98f557c530c04d9ac25e2f2a8b600e92a0b1ae7c89c20010000006b483045022100f06b3db1c0a11af348401f9cebe10ae2659d6e766a9dcd9e3a04690ba10a160f02203f7fbd7dfcfc70863aface1a306fcc91bbadf6bc884c21a55ef0d32bd6b088c8412103e8be830d98bb3b007a0343ee5c36daa48796ae8bb57946b1e87378ad6e8a090dfeffffff9d0d4554fa692420a0830ca614b6c60f1bf8eaaa21afca4aa8c99fb052d9f398000000006b483045022100d920f2290548e92a6235f8b2513b7f693a64a0d3fa699f81a034f4b4608ff82f0220767d7d98025aff3c7bd5f2a66aab6a824f5990392e6489aae1e1ae3472d8dffb412103e8be830d98bb3b007a0343ee5c36daa48796ae8bb57946b1e87378ad6e8a090dfeffffff02807c814a000000001976a9143a6bf34ebfcf30e8541bbb33a7882845e5a29cb488ac76b0e60e000000001976a914bd492b67f90cb85918494767ebb23102c4f06b7088ac67000000")

	b.Run("toBytesHelper", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			txBytes := tx.Bytes()
			_ = txBytes
		}
	})
}

// BenchmarkClone benchmarks the Clone method of a transaction.
func BenchmarkClone(b *testing.B) {
	tx, _ := bt.NewTxFromString("0200000003a9bc457fdc6a54d99300fb137b23714d860c350a9d19ff0f571e694a419ff3a0010000006b48304502210086c83beb2b2663e4709a583d261d75be538aedcafa7766bd983e5c8db2f8b2fc02201a88b178624ab0ad1748b37c875f885930166237c88f5af78ee4e61d337f935f412103e8be830d98bb3b007a0343ee5c36daa48796ae8bb57946b1e87378ad6e8a090dfeffffff0092bb9a47e27bf64fc98f557c530c04d9ac25e2f2a8b600e92a0b1ae7c89c20010000006b483045022100f06b3db1c0a11af348401f9cebe10ae2659d6e766a9dcd9e3a04690ba10a160f02203f7fbd7dfcfc70863aface1a306fcc91bbadf6bc884c21a55ef0d32bd6b088c8412103e8be830d98bb3b007a0343ee5c36daa48796ae8bb57946b1e87378ad6e8a090dfeffffff9d0d4554fa692420a0830ca614b6c60f1bf8eaaa21afca4aa8c99fb052d9f398000000006b483045022100d920f2290548e92a6235f8b2513b7f693a64a0d3fa699f81a034f4b4608ff82f0220767d7d98025aff3c7bd5f2a66aab6a824f5990392e6489aae1e1ae3472d8dffb412103e8be830d98bb3b007a0343ee5c36daa48796ae8bb57946b1e87378ad6e8a090dfeffffff02807c814a000000001976a9143a6bf34ebfcf30e8541bbb33a7882845e5a29cb488ac76b0e60e000000001976a914bd492b67f90cb85918494767ebb23102c4f06b7088ac67000000")

	b.Run("clone", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			clone := tx.Clone()
			_ = clone
		}
	})
}

// BenchmarkSize benchmarks the zero-allocation Size method vs len(Bytes()).
func BenchmarkSize(b *testing.B) {
	tx, _ := bt.NewTxFromString("0200000003a9bc457fdc6a54d99300fb137b23714d860c350a9d19ff0f571e694a419ff3a0010000006b48304502210086c83beb2b2663e4709a583d261d75be538aedcafa7766bd983e5c8db2f8b2fc02201a88b178624ab0ad1748b37c875f885930166237c88f5af78ee4e61d337f935f412103e8be830d98bb3b007a0343ee5c36daa48796ae8bb57946b1e87378ad6e8a090dfeffffff0092bb9a47e27bf64fc98f557c530c04d9ac25e2f2a8b600e92a0b1ae7c89c20010000006b483045022100f06b3db1c0a11af348401f9cebe10ae2659d6e766a9dcd9e3a04690ba10a160f02203f7fbd7dfcfc70863aface1a306fcc91bbadf6bc884c21a55ef0d32bd6b088c8412103e8be830d98bb3b007a0343ee5c36daa48796ae8bb57946b1e87378ad6e8a090dfeffffff9d0d4554fa692420a0830ca614b6c60f1bf8eaaa21afca4aa8c99fb052d9f398000000006b483045022100d920f2290548e92a6235f8b2513b7f693a64a0d3fa699f81a034f4b4608ff82f0220767d7d98025aff3c7bd5f2a66aab6a824f5990392e6489aae1e1ae3472d8dffb412103e8be830d98bb3b007a0343ee5c36daa48796ae8bb57946b1e87378ad6e8a090dfeffffff02807c814a000000001976a9143a6bf34ebfcf30e8541bbb33a7882845e5a29cb488ac76b0e60e000000001976a914bd492b67f90cb85918494767ebb23102c4f06b7088ac67000000")

	b.Run("Size_arithmetic", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			s := tx.Size()
			_ = s
		}
	})

	b.Run("Size_via_Bytes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			s := len(tx.Bytes())
			_ = s
		}
	})
}

// TestSize_MatchesBytes verifies the arithmetic Size() matches len(Bytes()) for various tx shapes.
func TestSize_MatchesBytes(t *testing.T) {
	tests := []struct {
		name    string
		buildTx func(t *testing.T) *bt.Tx
	}{
		{
			name: "real tx with 3 inputs 2 outputs",
			buildTx: func(t *testing.T) *bt.Tx {
				tx, err := bt.NewTxFromString("0200000003a9bc457fdc6a54d99300fb137b23714d860c350a9d19ff0f571e694a419ff3a0010000006b48304502210086c83beb2b2663e4709a583d261d75be538aedcafa7766bd983e5c8db2f8b2fc02201a88b178624ab0ad1748b37c875f885930166237c88f5af78ee4e61d337f935f412103e8be830d98bb3b007a0343ee5c36daa48796ae8bb57946b1e87378ad6e8a090dfeffffff0092bb9a47e27bf64fc98f557c530c04d9ac25e2f2a8b600e92a0b1ae7c89c20010000006b483045022100f06b3db1c0a11af348401f9cebe10ae2659d6e766a9dcd9e3a04690ba10a160f02203f7fbd7dfcfc70863aface1a306fcc91bbadf6bc884c21a55ef0d32bd6b088c8412103e8be830d98bb3b007a0343ee5c36daa48796ae8bb57946b1e87378ad6e8a090dfeffffff9d0d4554fa692420a0830ca614b6c60f1bf8eaaa21afca4aa8c99fb052d9f398000000006b483045022100d920f2290548e92a6235f8b2513b7f693a64a0d3fa699f81a034f4b4608ff82f0220767d7d98025aff3c7bd5f2a66aab6a824f5990392e6489aae1e1ae3472d8dffb412103e8be830d98bb3b007a0343ee5c36daa48796ae8bb57946b1e87378ad6e8a090dfeffffff02807c814a000000001976a9143a6bf34ebfcf30e8541bbb33a7882845e5a29cb488ac76b0e60e000000001976a914bd492b67f90cb85918494767ebb23102c4f06b7088ac67000000")
				require.NoError(t, err)
				return tx
			},
		},
		{
			name: "coinbase tx",
			buildTx: func(t *testing.T) *bt.Tx {
				tx, err := bt.NewTxFromString("01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff0704ffff001d0104ffffffff0100f2052a0100000043410496b538e853519c726a2c91e61ec11600ae1390813a627c66fb8be7947be63c52da7589379515d4e0a604f8141781e62294721166bf621e73a82cbf2342c858eeac00000000")
				require.NoError(t, err)
				return tx
			},
		},
		{
			name: "single input single output",
			buildTx: func(t *testing.T) *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.From("b7b0650a7c3a1bd4f7571b4c1e38f05171b565b8e28b2e337031ee31e9fa8eb6", 0, "76a914167c3e911a14a92760b81334d01045da61e9681888ac", 100000))
				tx.AddOutput(&bt.Output{
					Satoshis:      99000,
					LockingScript: bscript.NewFromBytes([]byte{0x76, 0xa9, 0x14}),
				})
				return tx
			},
		},
		{
			name: "large script output",
			buildTx: func(t *testing.T) *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.From("b7b0650a7c3a1bd4f7571b4c1e38f05171b565b8e28b2e337031ee31e9fa8eb6", 0, "76a914167c3e911a14a92760b81334d01045da61e9681888ac", 100000))
				bigScript := make([]byte, 100000)
				_, _ = rand.Read(bigScript)
				tx.AddOutput(&bt.Output{
					Satoshis:      1,
					LockingScript: bscript.NewFromBytes(bigScript),
				})
				return tx
			},
		},
		{
			name: "empty unlocking scripts",
			buildTx: func(t *testing.T) *bt.Tx {
				tx := bt.NewTx()
				tx.AddOutput(&bt.Output{
					Satoshis:      1000,
					LockingScript: bscript.NewFromBytes([]byte{0x51}),
				})
				return tx
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := tt.buildTx(t)
			expected := len(tx.Bytes())
			got := tx.Size()
			require.Equal(t, expected, got, "Size() = %d, len(Bytes()) = %d", got, expected)
		})
	}
}

// TestWriteTo_MatchesBytes verifies WriteTo produces identical output to Bytes() for various tx shapes.
func TestWriteTo_MatchesBytes(t *testing.T) {
	tests := []struct {
		name    string
		buildTx func(t *testing.T) *bt.Tx
	}{
		{
			name: "real tx with 3 inputs 2 outputs",
			buildTx: func(t *testing.T) *bt.Tx {
				tx, err := bt.NewTxFromString("0200000003a9bc457fdc6a54d99300fb137b23714d860c350a9d19ff0f571e694a419ff3a0010000006b48304502210086c83beb2b2663e4709a583d261d75be538aedcafa7766bd983e5c8db2f8b2fc02201a88b178624ab0ad1748b37c875f885930166237c88f5af78ee4e61d337f935f412103e8be830d98bb3b007a0343ee5c36daa48796ae8bb57946b1e87378ad6e8a090dfeffffff0092bb9a47e27bf64fc98f557c530c04d9ac25e2f2a8b600e92a0b1ae7c89c20010000006b483045022100f06b3db1c0a11af348401f9cebe10ae2659d6e766a9dcd9e3a04690ba10a160f02203f7fbd7dfcfc70863aface1a306fcc91bbadf6bc884c21a55ef0d32bd6b088c8412103e8be830d98bb3b007a0343ee5c36daa48796ae8bb57946b1e87378ad6e8a090dfeffffff9d0d4554fa692420a0830ca614b6c60f1bf8eaaa21afca4aa8c99fb052d9f398000000006b483045022100d920f2290548e92a6235f8b2513b7f693a64a0d3fa699f81a034f4b4608ff82f0220767d7d98025aff3c7bd5f2a66aab6a824f5990392e6489aae1e1ae3472d8dffb412103e8be830d98bb3b007a0343ee5c36daa48796ae8bb57946b1e87378ad6e8a090dfeffffff02807c814a000000001976a9143a6bf34ebfcf30e8541bbb33a7882845e5a29cb488ac76b0e60e000000001976a914bd492b67f90cb85918494767ebb23102c4f06b7088ac67000000")
				require.NoError(t, err)
				return tx
			},
		},
		{
			name: "coinbase tx",
			buildTx: func(t *testing.T) *bt.Tx {
				tx, err := bt.NewTxFromString("01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff0704ffff001d0104ffffffff0100f2052a0100000043410496b538e853519c726a2c91e61ec11600ae1390813a627c66fb8be7947be63c52da7589379515d4e0a604f8141781e62294721166bf621e73a82cbf2342c858eeac00000000")
				require.NoError(t, err)
				return tx
			},
		},
		{
			name: "single input single output",
			buildTx: func(t *testing.T) *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.From("b7b0650a7c3a1bd4f7571b4c1e38f05171b565b8e28b2e337031ee31e9fa8eb6", 0, "76a914167c3e911a14a92760b81334d01045da61e9681888ac", 100000))
				tx.AddOutput(&bt.Output{
					Satoshis:      99000,
					LockingScript: bscript.NewFromBytes([]byte{0x76, 0xa9, 0x14}),
				})
				return tx
			},
		},
		{
			name: "large script output",
			buildTx: func(t *testing.T) *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.From("b7b0650a7c3a1bd4f7571b4c1e38f05171b565b8e28b2e337031ee31e9fa8eb6", 0, "76a914167c3e911a14a92760b81334d01045da61e9681888ac", 100000))
				bigScript := make([]byte, 100000)
				_, _ = rand.Read(bigScript)
				tx.AddOutput(&bt.Output{
					Satoshis:      1,
					LockingScript: bscript.NewFromBytes(bigScript),
				})
				return tx
			},
		},
		{
			name: "empty unlocking scripts",
			buildTx: func(t *testing.T) *bt.Tx {
				tx := bt.NewTx()
				tx.AddOutput(&bt.Output{
					Satoshis:      1000,
					LockingScript: bscript.NewFromBytes([]byte{0x51}),
				})
				return tx
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := tt.buildTx(t)

			// Test standard WriteTo
			expected := tx.Bytes()
			var buf bytes.Buffer
			n, err := tx.WriteTo(&buf)
			require.NoError(t, err)
			require.Equal(t, int64(len(expected)), n)
			require.Equal(t, expected, buf.Bytes())

			// Test SerializeTo matches SerializeBytes
			expectedSerialized := tx.SerializeBytes()
			buf.Reset()
			n, err = tx.SerializeTo(&buf)
			require.NoError(t, err)
			require.Equal(t, int64(len(expectedSerialized)), n)
			require.Equal(t, expectedSerialized, buf.Bytes())
		})
	}
}

// BenchmarkWriteTo benchmarks WriteTo vs Bytes() serialization.
func BenchmarkWriteTo(b *testing.B) {
	tx, _ := bt.NewTxFromString("0200000003a9bc457fdc6a54d99300fb137b23714d860c350a9d19ff0f571e694a419ff3a0010000006b48304502210086c83beb2b2663e4709a583d261d75be538aedcafa7766bd983e5c8db2f8b2fc02201a88b178624ab0ad1748b37c875f885930166237c88f5af78ee4e61d337f935f412103e8be830d98bb3b007a0343ee5c36daa48796ae8bb57946b1e87378ad6e8a090dfeffffff0092bb9a47e27bf64fc98f557c530c04d9ac25e2f2a8b600e92a0b1ae7c89c20010000006b483045022100f06b3db1c0a11af348401f9cebe10ae2659d6e766a9dcd9e3a04690ba10a160f02203f7fbd7dfcfc70863aface1a306fcc91bbadf6bc884c21a55ef0d32bd6b088c8412103e8be830d98bb3b007a0343ee5c36daa48796ae8bb57946b1e87378ad6e8a090dfeffffff9d0d4554fa692420a0830ca614b6c60f1bf8eaaa21afca4aa8c99fb052d9f398000000006b483045022100d920f2290548e92a6235f8b2513b7f693a64a0d3fa699f81a034f4b4608ff82f0220767d7d98025aff3c7bd5f2a66aab6a824f5990392e6489aae1e1ae3472d8dffb412103e8be830d98bb3b007a0343ee5c36daa48796ae8bb57946b1e87378ad6e8a090dfeffffff02807c814a000000001976a9143a6bf34ebfcf30e8541bbb33a7882845e5a29cb488ac76b0e60e000000001976a914bd492b67f90cb85918494767ebb23102c4f06b7088ac67000000")

	b.Run("WriteTo", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = tx.WriteTo(io.Discard)
		}
	})

	b.Run("Bytes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = tx.Bytes()
		}
	})
}

// BenchmarkShallowClone benchmarks the ShallowClone method of a transaction.
func BenchmarkShallowClone(b *testing.B) {
	tx, _ := bt.NewTxFromString("0200000003a9bc457fdc6a54d99300fb137b23714d860c350a9d19ff0f571e694a419ff3a0010000006b48304502210086c83beb2b2663e4709a583d261d75be538aedcafa7766bd983e5c8db2f8b2fc02201a88b178624ab0ad1748b37c875f885930166237c88f5af78ee4e61d337f935f412103e8be830d98bb3b007a0343ee5c36daa48796ae8bb57946b1e87378ad6e8a090dfeffffff0092bb9a47e27bf64fc98f557c530c04d9ac25e2f2a8b600e92a0b1ae7c89c20010000006b483045022100f06b3db1c0a11af348401f9cebe10ae2659d6e766a9dcd9e3a04690ba10a160f02203f7fbd7dfcfc70863aface1a306fcc91bbadf6bc884c21a55ef0d32bd6b088c8412103e8be830d98bb3b007a0343ee5c36daa48796ae8bb57946b1e87378ad6e8a090dfeffffff9d0d4554fa692420a0830ca614b6c60f1bf8eaaa21afca4aa8c99fb052d9f398000000006b483045022100d920f2290548e92a6235f8b2513b7f693a64a0d3fa699f81a034f4b4608ff82f0220767d7d98025aff3c7bd5f2a66aab6a824f5990392e6489aae1e1ae3472d8dffb412103e8be830d98bb3b007a0343ee5c36daa48796ae8bb57946b1e87378ad6e8a090dfeffffff02807c814a000000001976a9143a6bf34ebfcf30e8541bbb33a7882845e5a29cb488ac76b0e60e000000001976a914bd492b67f90cb85918494767ebb23102c4f06b7088ac67000000")

	b.Run("clone", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			clone := tx.ShallowClone()
			_ = clone
		}
	})
}
