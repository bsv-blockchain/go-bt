// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package chainhash

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mainNetGenesisHash is the hash of the first block in the block chain for the
// main network (genesis block).
var mainNetGenesisHash = Hash([HashSize]byte{ // Make go vet happy.
	0x6f, 0xe2, 0x8c, 0x0a, 0xb6, 0xf1, 0xb3, 0x72,
	0xc1, 0xa6, 0xa2, 0x46, 0xae, 0x63, 0xf7, 0x4f,
	0x93, 0x1e, 0x83, 0x65, 0xe1, 0x5a, 0x08, 0x9c,
	0x68, 0xd6, 0x19, 0x00, 0x00, 0x00, 0x00, 0x00,
})

// TestHash tests the Hash API.
func TestHash(t *testing.T) {
	// Hash of block 234439.
	blockHashStr := "14a0810ac680a3eb3f82edc878cea25ec41d6b790744e5daeef"
	blockHash, err := NewHashFromStr(blockHashStr)
	if err != nil {
		t.Errorf("NewHashFromStr: %v", err)
	}

	// Hash of block 234440 as byte slice.
	buf := []byte{
		0x79, 0xa6, 0x1a, 0xdb, 0xc6, 0xe5, 0xa2, 0xe1,
		0x39, 0xd2, 0x71, 0x3a, 0x54, 0x6e, 0xc7, 0xc8,
		0x75, 0x63, 0x2e, 0x75, 0xf1, 0xdf, 0x9c, 0x3f,
		0xa6, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	hash, err := NewHash(buf)
	if err != nil {
		t.Errorf("NewHash: unexpected error %v", err)
	}

	// Ensure proper size.
	if len(*hash) != HashSize {
		t.Errorf("NewHash: hash length mismatch - got: %v, want: %v",
			len(*hash), HashSize)
	}

	// Ensure contents match.
	if !bytes.Equal(hash[:], buf) {
		t.Errorf("NewHash: hash contents mismatch - got: %v, want: %v",
			hash[:], buf)
	}

	// Ensure contents of hash of block 234440 don't match 234439.
	if hash.IsEqual(blockHash) {
		t.Errorf("IsEqual: hash contents should not match - got: %v, want: %v",
			hash, blockHash)
	}

	// Set hash from byte slice and ensure contents match.
	err = hash.SetBytes(blockHash.CloneBytes())
	if err != nil {
		t.Errorf("SetBytes: %v", err)
	}
	if !hash.IsEqual(blockHash) {
		t.Errorf("IsEqual: hash contents mismatch - got: %v, want: %v",
			hash, blockHash)
	}

	// Ensure nil hashes are handled properly.
	if !(*Hash)(nil).IsEqual(nil) {
		t.Error("IsEqual: nil hashes should match")
	}
	if hash.IsEqual(nil) {
		t.Error("IsEqual: non-nil hash matches nil hash")
	}

	// Invalid size for SetBytes.
	err = hash.SetBytes([]byte{0x00})
	if err == nil {
		t.Errorf("SetBytes: failed to received expected err - got: nil")
	}

	// Invalid size for NewHash.
	invalidHash := make([]byte, HashSize+1)
	_, err = NewHash(invalidHash)
	if err == nil {
		t.Errorf("NewHash: failed to received expected err - got: nil")
	}
}

// TestHashString tests the string output for hashes.
func TestHashString(t *testing.T) {
	// Block 100000 hash.
	wantStr := "000000000003ba27aa200b1cecaad478d2b00432346c3f1f3986da1afd33e506"
	hash := Hash([HashSize]byte{ // Make go vet happy.
		0x06, 0xe5, 0x33, 0xfd, 0x1a, 0xda, 0x86, 0x39,
		0x1f, 0x3f, 0x6c, 0x34, 0x32, 0x04, 0xb0, 0xd2,
		0x78, 0xd4, 0xaa, 0xec, 0x1c, 0x0b, 0x20, 0xaa,
		0x27, 0xba, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00,
	})

	hashStr := hash.String()
	if hashStr != wantStr {
		t.Errorf("String: wrong hash string - got %v, want %v",
			hashStr, wantStr)
	}
}

// TestNewHashFromStr executes tests against the NewHashFromStr function.
func TestNewHashFromStr(t *testing.T) {
	tests := []struct {
		in   string
		want Hash
		err  error
	}{
		// Genesis hash.
		{
			"000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f",
			mainNetGenesisHash,
			nil,
		},

		// Genesis hash with stripped leading zeros.
		{
			"19d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f",
			mainNetGenesisHash,
			nil,
		},

		// Empty string.
		{
			"",
			Hash{},
			nil,
		},

		// Single digit hash.
		{
			"1",
			Hash([HashSize]byte{ // Make go vet happy.
				0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			}),
			nil,
		},

		// Block 203707 with stripped leading zeros.
		{
			"3264bc2ac36a60840790ba1d475d01367e7c723da941069e9dc",
			Hash([HashSize]byte{ // Make go vet happy.
				0xdc, 0xe9, 0x69, 0x10, 0x94, 0xda, 0x23, 0xc7,
				0xe7, 0x67, 0x13, 0xd0, 0x75, 0xd4, 0xa1, 0x0b,
				0x79, 0x40, 0x08, 0xa6, 0x36, 0xac, 0xc2, 0x4b,
				0x26, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			}),
			nil,
		},

		// Hash string that is too long.
		{
			"01234567890123456789012345678901234567890123456789012345678912345",
			Hash{},
			ErrHashStrSize,
		},

		// Hash string that is contains non-hex chars.
		{
			"abcdefg",
			Hash{},
			hex.InvalidByteError('g'),
		},
	}

	unexpectedErrStr := "NewHashFromStr #%d failed to detect expected error - got: %v want: %v"
	unexpectedResultStr := "NewHashFromStr #%d got: %v want: %v"
	t.Logf("Running %d tests", len(tests))
	for i, test := range tests {
		result, err := NewHashFromStr(test.in)
		if !errors.Is(err, test.err) {
			t.Errorf(unexpectedErrStr, i, err, test.err)
			continue
		} else if err != nil {
			// Got expected error. Move on to the next test.
			continue
		}
		if !test.want.IsEqual(result) {
			t.Errorf(unexpectedResultStr, i, result, &test.want)
			continue
		}
	}
}

func TestMarshaling(t *testing.T) {
	type test struct {
		Hash Hash `json:"hash"`
	}

	myData := &test{
		Hash: HashH([]byte("hello")),
	}

	assert.Equal(t, "24988b93623304735e42a71f5c1e161b9ee2b9c52a3be8260ea3b05fba4df22c", myData.Hash.String())

	b, err := json.Marshal(myData)
	require.NoError(t, err)
	assert.JSONEq(t, `{"hash":"24988b93623304735e42a71f5c1e161b9ee2b9c52a3be8260ea3b05fba4df22c"}`, string(b))

	var myData2 test
	err = json.Unmarshal(b, &myData2)
	require.NoError(t, err)
	assert.Equal(t, "24988b93623304735e42a71f5c1e161b9ee2b9c52a3be8260ea3b05fba4df22c", myData2.Hash.String())
}

func TestHashMarshal(t *testing.T) {
	h := DoubleHashH([]byte("test"))
	tests := []struct {
		name    string
		hash    *Hash
		want    []byte
		wantErr bool
	}{
		{
			name:    "nil hash",
			hash:    nil,
			want:    nil,
			wantErr: false,
		},
		{
			name:    "valid hash",
			hash:    &h,
			want:    h[:],
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.hash.Marshal()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHashMarshalTo(t *testing.T) {
	h := DoubleHashH([]byte("test"))
	tests := []struct {
		name    string
		hash    *Hash
		data    []byte
		want    int
		wantErr bool
	}{
		{
			name:    "nil hash",
			hash:    nil,
			data:    make([]byte, HashSize),
			want:    0,
			wantErr: false,
		},
		{
			name:    "valid hash",
			hash:    &h,
			data:    make([]byte, HashSize),
			want:    HashSize,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytes, err := tt.hash.Marshal()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, bytes, tt.want)

			if tt.hash != nil {
				assert.Equal(t, tt.hash[:], bytes)
			}
		})
	}
}

func TestHashUnmarshal(t *testing.T) {
	h := DoubleHashH([]byte("test"))
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "invalid length",
			data:    []byte("too short"),
			wantErr: true,
		},
		{
			name:    "valid hash",
			data:    h[:],
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := new(Hash)
			err := hash.Unmarshal(tt.data)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.True(t, bytes.Equal(tt.data, hash[:]))
		})
	}
}

func TestHashSize(t *testing.T) {
	h := DoubleHashH([]byte("test"))
	tests := []struct {
		name string
		hash *Hash
		want int
	}{
		{
			name: "nil hash",
			hash: nil,
			want: 0,
		},
		{
			name: "valid hash",
			hash: &h,
			want: HashSize,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.hash.Size()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHashProtobufSerialization(t *testing.T) {
	// Create an original hash
	original := DoubleHashH([]byte("test data"))
	h := &original

	// Marshal to protobuf format
	data, err := h.Marshal()
	require.NoError(t, err)
	assert.NotNil(t, data)
	assert.Len(t, data, HashSize)

	// Create a new hash and unmarshal the data
	unmarshaled := new(Hash)
	err = unmarshaled.Unmarshal(data)
	require.NoError(t, err)

	// Verify the unmarshaled hash matches the original
	assert.Equal(t, h[:], unmarshaled[:])
	assert.Equal(t, original[:], unmarshaled[:])
}

func TestHashSQLSerialization(t *testing.T) {
	// Test cases for scanning different types of input
	testCases := []struct {
		name     string
		input    interface{}
		wantHash *Hash
		wantErr  bool
	}{
		{
			name:     "scan nil",
			input:    nil,
			wantHash: &Hash{},
			wantErr:  true,
		},
		{
			name:     "scan bytes",
			input:    mainNetGenesisHash[:],
			wantHash: &mainNetGenesisHash,
			wantErr:  false,
		},
		{
			name:     "scan string",
			input:    mainNetGenesisHash.String(),
			wantHash: &mainNetGenesisHash,
			wantErr:  false,
		},
		{
			name:     "scan invalid bytes",
			input:    []byte{0x00},
			wantHash: &Hash{},
			wantErr:  true,
		},
		{
			name:     "scan invalid string",
			input:    "invalid",
			wantHash: &Hash{},
			wantErr:  true,
		},
		{
			name:     "scan unsupported type",
			input:    123,
			wantHash: &Hash{},
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var hash Hash
			err := hash.Scan(tc.input)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantHash, &hash)
			}
		})
	}

	// Test Value() method
	t.Run("value nil hash", func(t *testing.T) {
		var hash *Hash
		val, err := hash.Value()
		require.NoError(t, err)
		assert.Nil(t, val)
	})

	t.Run("value non-nil hash", func(t *testing.T) {
		hash := mainNetGenesisHash
		val, err := hash.Value()
		require.NoError(t, err)
		assert.Equal(t, mainNetGenesisHash[:], val)

		// Verify we can scan the value back
		var newHash Hash
		err = newHash.Scan(val)
		require.NoError(t, err)
		assert.Equal(t, hash, newHash)
	})
}
