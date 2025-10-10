// Package chainhash provides a type for representing hashes used in the
// Bitcoin protocol and provides functions for working with them.
//
// Copyright (c) 2013-2016 The btcsuite developers
// Copyright (c) 2015 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.
package chainhash

import (
	"bytes"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// HashSize of an array used to store hashes.  See Hash.
const HashSize = 32

// MaxHashStringSize is the maximum length of a Hash string.
const MaxHashStringSize = HashSize * 2

// ErrHashStrSize describes an error that indicates the caller specified a hash
// string that has too many characters.
var ErrHashStrSize = fmt.Errorf("max hash string length is %v bytes", MaxHashStringSize)

// Static errors for err113 linter compliance
var (
	ErrInvalidHashLength      = errors.New("invalid hash length")
	ErrInvalidUnmarshalLength = errors.New("invalid length for chainhash.Hash")
	ErrUnsupportedType        = errors.New("unsupported type for hash")
)

// Hash is used in several of the bitcoin messages and common structures.  It
// typically represents the double sha256 of data.
type Hash [HashSize]byte

// String returns the Hash as the hexadecimal string of the byte-reversed
// hash.
func (h Hash) String() string {
	for i := 0; i < HashSize/2; i++ {
		h[i], h[HashSize-1-i] = h[HashSize-1-i], h[i]
	}
	return hex.EncodeToString(h[:])
}

// CloneBytes returns a copy of the bytes which represent the hash as a byte
// slice.
//
// NOTE: It is generally less expensive to just slice the hash directly thereby reusing
// the same bytes rather than calling this method.
func (h *Hash) CloneBytes() []byte {
	newHash := make([]byte, HashSize)
	copy(newHash, h[:])

	return newHash
}

// SetBytes sets the bytes which represent the hash.  An error is returned if
// the number of bytes passed in is not HashSize.
func (h *Hash) SetBytes(newHash []byte) error {
	nhLen := len(newHash)
	if nhLen != HashSize {
		return fmt.Errorf("%w: got %v, want %v", ErrInvalidHashLength, nhLen, HashSize)
	}
	copy(h[:], newHash)

	return nil
}

// IsEqual returns true if target is the same as hash.
func (h *Hash) IsEqual(target *Hash) bool {
	if h == nil && target == nil {
		return true
	}
	if h == nil || target == nil {
		return false
	}
	return *h == *target
}

// MarshalJSON returns the JSON encoding of the hash as a hexadecimal
func (h *Hash) MarshalJSON() ([]byte, error) {
	return json.Marshal(h.String())
}

// UnmarshalJSON parses the JSON-encoded hash string and sets the hash to the
func (h *Hash) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	hs, err := NewHashFromStr(s)
	if err != nil {
		return err
	}
	*h = *hs
	return nil
}

// NewHash returns a new Hash from a byte slice.  An error is returned if
// the number of bytes passed in is not HashSize.
func NewHash(newHash []byte) (*Hash, error) {
	var sh Hash
	err := sh.SetBytes(newHash)
	if err != nil {
		return nil, err
	}
	return &sh, err
}

// func NewHashNoError(newHash []byte) *Hash {
// 	sh, _ := NewHash(newHash)
// 	return sh
// }

// NewHashFromStr creates a Hash from a hash string.  The string should be
// the hexadecimal string of a byte-reversed hash, but any missing characters
// result in zero padding at the end of the Hash.
func NewHashFromStr(hash string) (*Hash, error) {
	ret := new(Hash)
	err := Decode(ret, hash)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// func NewHashFromStrNoError(hash string) *Hash {
// 	sh, _ := NewHashFromStr(hash)
// 	return sh
// }

// Decode decodes the byte-reversed hexadecimal string encoding of a Hash to a
// destination.
func Decode(dst *Hash, src string) error {
	// Return error if hash string is too long.
	if len(src) > MaxHashStringSize {
		return ErrHashStrSize
	}

	// Hex decoder expects the hash to be a multiple of two.  When not, pad
	// with a leading zero.
	var srcBytes []byte
	if len(src)%2 == 0 {
		srcBytes = []byte(src)
	} else {
		srcBytes = make([]byte, 1+len(src))
		srcBytes[0] = '0'
		copy(srcBytes[1:], src)
	}

	// Hex decode the source bytes to a temporary destination.
	var reversedHash Hash
	_, err := hex.Decode(reversedHash[HashSize-hex.DecodedLen(len(srcBytes)):], srcBytes)
	if err != nil {
		return err
	}

	// Reverse copy from the temporary hash to destination.  Because the
	// temporary was zeroed, the written result will be correctly padded.
	for i, b := range reversedHash[:HashSize/2] {
		dst[i], dst[HashSize-1-i] = reversedHash[HashSize-1-i], b
	}

	return nil
}

// Marshal converts a chainhash.Hash to a protobuf []byte.
func (h *Hash) Marshal() ([]byte, error) {
	if h == nil {
		return nil, nil
	}

	return h[:], nil
}

// Unmarshal converts a protobuf []byte to chainhash.Hash.
func (h *Hash) Unmarshal(data []byte) error {
	if len(data) != 32 {
		return ErrInvalidUnmarshalLength
	}
	copy(h[:], data)
	return nil
}

var _ proto.Message = (*Hash)(nil)

// ProtoReflect implements proto.Message
func (h *Hash) ProtoReflect() protoreflect.Message {
	return nil // `nil` is acceptable for non-nested fields like bytes
}

// Size implements proto.Sizer
func (h *Hash) Size() int {
	if h == nil {
		return 0
	}

	return HashSize
}

// Equal compares two Hashes for equality.
func (h Hash) Equal(other Hash) bool {
	return bytes.Equal(h[0:], other[0:])
}

// Scan implements the sql.Scanner
func (h *Hash) Scan(value interface{}) error {
	switch v := value.(type) {
	case []byte:
		hs, err := NewHash(v)
		if err != nil {
			return fmt.Errorf("failed to convert bytes to hash: %w", err)
		}

		*h = *hs

		return nil
	case string:
		hs, err := NewHashFromStr(v)
		if err != nil {
			return fmt.Errorf("failed to convert string to hash: %w", err)
		}

		*h = *hs

		return nil
	default:
		return fmt.Errorf("%w: %T", ErrUnsupportedType, value)
	}
}

// Value implements the driver.Valuer
func (h *Hash) Value() (driver.Value, error) {
	if h == nil {
		return nil, nil //nolint:nilnil // allow nil values
	}

	return h[:], nil
}
