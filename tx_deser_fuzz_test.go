package bt

import (
	"encoding/hex"
	"testing"
)

// FuzzNewTxFromBytes feeds arbitrary bytes to the transaction deserializer, the
// code path that runs on untrusted transactions received from peers. Parsing
// must never panic or attempt an unbounded allocation from a count/length field
// (input count, output count, or a script length varint claiming far more bytes
// than are present).
func FuzzNewTxFromBytes(f *testing.F) {
	if valid, err := hex.DecodeString("0100000003d5da6f960610cc65153521fd16dbe96b499143ac8d03222c13a9b97ce2dd8e3c000000006b48304502210081214df575da1e9378f1d5a29dfd6811e93466a7222fb010b7c50dd2d44d7f2e0220399bb396336d2e294049e7db009926b1b30018ac834ee0cbca20b9d99f488038412102798913bc057b344de675dac34faafe3dc2f312c758cd9068209f810877306d66ffffffffd5da6f960610cc65153521fd16dbe96b499143ac8d03222c13a9b97ce2dd8e3c0200000069463043021f7059426d6aeb7d74275e52819a309b2bf903bd18b2b4d942d0e8e037681df702203f851f8a45aabfefdca5822f457609600f5d12a173adc09c6e7e2d4fdff7620a412102798913bc057b344de675dac34faafe3dc2f312c758cd9068209f810877306d66ffffffffd5da6f960610cc65153521fd16dbe96b499143ac8d03222c13a9b97ce2dd8e3c720000006b483045022100e7b3837f2818fe00a05293e0f90e9005d59b0c5c8890f22bd31c36190a9b55e9022027de4b77b78139ea21b9fd30876a447bbf29662bd19d7914028c607bccd772e4412102798913bc057b344de675dac34faafe3dc2f312c758cd9068209f810877306d66ffffffff01e8030000000000001976a914eb0bd5edba389198e73f8efabddfc61666969ff788ac00000000"); err == nil {
		f.Add(valid)
	}
	f.Add([]byte{})
	// version(4) + input-count varint = 0xFFFFFFFFFFFFFFFF, no inputs behind it.
	f.Add([]byte{0x01, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})
	// extended-format marker (0000 + 0xEF) then a huge input count.
	f.Add([]byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0xef, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})

	f.Fuzz(func(_ *testing.T, data []byte) {
		_, _ = NewTxFromBytes(data)
	})
}
