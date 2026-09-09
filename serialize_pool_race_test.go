package bt_test

import (
	"bytes"
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSerializeToIsSafeUnderConcurrency exercises the shared scratch pool from
// many goroutines at once, which is how Teranode uses it: four subtree writers
// run in parallel over one block. A buffer handed to two goroutines, or
// returned to the pool while still referenced, would show up here as either a
// race or as bytes from one transaction appearing inside another.
func TestSerializeToIsSafeUnderConcurrency(t *testing.T) {
	tx := extendedShapeTx(t, 4, 4)

	var want bytes.Buffer

	_, err := tx.SerializeTo(&want)
	require.NoError(t, err)

	expected := append([]byte(nil), want.Bytes()...)

	var wg sync.WaitGroup

	for g := 0; g < 16; g++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for i := 0; i < 200; i++ {
				var got bytes.Buffer

				n, err := tx.SerializeTo(&got)
				if err != nil {
					t.Error(err)
					return
				}

				if n != int64(len(expected)) || !bytes.Equal(got.Bytes(), expected) {
					t.Error("serialised bytes differ under concurrency")
					return
				}

				_, _ = tx.SerializeTo(io.Discard)
			}
		}()
	}

	wg.Wait()
}
