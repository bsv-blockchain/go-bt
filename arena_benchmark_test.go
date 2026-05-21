package bt_test

import (
	"bytes"
	"testing"

	"github.com/bsv-blockchain/go-bt/v2"
	"github.com/bsv-blockchain/go-bt/v2/chainhash"
)

func BenchmarkTx_ReadFrom_vs_WithArena(b *testing.B) {
	raw := mustParseTx().Bytes()

	b.Run("ReadFrom", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			tx := &bt.Tx{}
			_, _ = tx.ReadFrom(bytes.NewReader(raw))
		}
	})

	b.Run("ReadFromWithArena", func(b *testing.B) {
		b.ReportAllocs()
		arena := bt.NewArena(4096)
		for i := 0; i < b.N; i++ {
			tx := &bt.Tx{}
			_, _ = tx.ReadFromWithArena(bytes.NewReader(raw), arena)
			arena.Reset()
		}
	})
}

func BenchmarkSubtreeDecode_5kTxs(b *testing.B) {
	tx := mustParseTx()
	raw := tx.Bytes()
	const N = 5000

	stream := make([]byte, 0, len(raw)*N)
	for i := 0; i < N; i++ {
		stream = append(stream, raw...)
	}

	b.Run("PerTxReadFrom", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			r := bytes.NewReader(stream)
			for j := 0; j < N; j++ {
				t := &bt.Tx{}
				_, _ = t.ReadFrom(r)
			}
		}
	})

	b.Run("PerSubtreeArena", func(b *testing.B) {
		b.ReportAllocs()
		arena := bt.NewArena(1 << 20)
		for i := 0; i < b.N; i++ {
			r := bytes.NewReader(stream)
			arena.Reset()
			for j := 0; j < N; j++ {
				t := &bt.Tx{}
				_, _ = t.ReadFromWithArena(r, arena)
			}
		}
	})
}

func BenchmarkHashTxIDInto_vs_TxIDChainHash(b *testing.B) {
	tx := mustParseTx()

	b.Run("TxIDChainHash", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			tx.SetTxHash(nil) // bust cache
			_ = tx.TxIDChainHash()
		}
	})

	b.Run("HashTxIDInto", func(b *testing.B) {
		b.ReportAllocs()
		scratch := make([]byte, 0, tx.Size())
		for i := 0; i < b.N; i++ {
			tx.SetTxHash(nil)
			var h chainhash.Hash
			h, scratch = tx.HashTxIDInto(scratch)
			_ = h
		}
	})
}
