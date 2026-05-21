package bt_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bsv-blockchain/go-bt/v2"
)

func TestArena_NewAndAlloc(t *testing.T) {
	a := bt.NewArena(64)
	require.NotNil(t, a)
	require.GreaterOrEqual(t, a.Cap(), 64)
	require.Equal(t, 0, a.Used())

	b1 := a.Alloc(8)
	require.Len(t, b1, 8)
	require.Equal(t, 8, cap(b1), "Alloc must return a 3-arg-sliced []byte with cap == len")
	require.Equal(t, 8, a.Used())

	b2 := a.Alloc(16)
	require.Len(t, b2, 16)
	require.Equal(t, 16, cap(b2))
	require.Equal(t, 24, a.Used())

	// Writes to b1 must not affect b2 (no overlap)
	for i := range b1 {
		b1[i] = 0xAA
	}
	for _, x := range b2 {
		require.Equal(t, byte(0), x)
	}
}

func TestArena_Alloc_BumpAndGrow(t *testing.T) {
	a := bt.NewArena(16)
	initialCap := a.Cap()
	require.GreaterOrEqual(t, initialCap, 16)

	// Fill to initial cap
	_ = a.Alloc(initialCap)

	// Next alloc must grow
	b := a.Alloc(8)
	require.Len(t, b, 8)
	require.GreaterOrEqual(t, a.Cap(), initialCap+8)
}

func TestArena_ZeroValue(t *testing.T) {
	var a bt.Arena
	b := a.Alloc(32)
	require.Len(t, b, 32)
	require.Equal(t, 32, a.Used())
}

func TestArena_Reset_RetainsSlab(t *testing.T) {
	a := bt.NewArena(64)
	_ = a.Alloc(32)
	capBefore := a.Cap()

	a.Reset()
	require.Equal(t, 0, a.Used())
	require.Equal(t, capBefore, a.Cap(), "Reset must retain the slab")

	b := a.Alloc(32)
	require.Len(t, b, 32)
}

func TestArena_ResetAndShrink(t *testing.T) {
	a := bt.NewArena(1024)
	_ = a.Alloc(800)

	a.ResetAndShrink(256) // 1024 > 256 -> drop slab
	require.Equal(t, 0, a.Used())
	require.Equal(t, 0, a.Cap())

	// Next alloc must regrow
	_ = a.Alloc(64)
	require.GreaterOrEqual(t, a.Cap(), 64)
}

func TestArena_ResetAndShrink_KeepsSmallSlab(t *testing.T) {
	a := bt.NewArena(64)
	_ = a.Alloc(32)
	capBefore := a.Cap()

	a.ResetAndShrink(256) // 64 < 256 -> keep
	require.Equal(t, capBefore, a.Cap())
}

func TestArena_AllocPanicsOnExcessive(t *testing.T) {
	a := bt.NewArena(0)
	require.Panics(t, func() {
		_ = a.Alloc(bt.MaxArenaAlloc + 1)
	})
	require.Panics(t, func() {
		_ = a.Alloc(-1)
	})
}

// TestArena_Alloc_320MB confirms a legitimate large-output allocation works.
// Skipped under -short because it allocates 320 MiB.
func TestArena_Alloc_320MB(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	const size = 320 << 20
	a := bt.NewArena(0)
	b := a.Alloc(size)
	require.Len(t, b, size)
	require.Equal(t, size, a.Used())
	require.GreaterOrEqual(t, a.Cap(), size)
}
