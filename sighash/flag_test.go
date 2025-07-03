package sighash

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlagHas_ReturnsExpectedResult(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		f        Flag
		check    Flag
		expected bool
	}{
		{"contains single flag", All | AnyOneCanPay, AnyOneCanPay, true},
		{"missing flag", AllForkID, AnyOneCanPay, false},
		{"fork id present", AllForkID | AnyOneCanPay, ForkID, true},
		{"wrong base flag", NoneForkID, Single, false},
		{"base flag", NoneForkID, None, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.f.Has(tt.check))
		})
	}
}

func TestFlagHasWithMask_ReturnsExpectedResult(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		f        Flag
		check    Flag
		expected bool
	}{
		{"all forkid masked", AllForkID, All, true},
		{"single forkid masked", SingleForkID | AnyOneCanPay, Single, true},
		{"none forkid masked", NoneForkID, None, true},
		{"mask mismatch", AllForkID, None, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.f.HasWithMask(tt.check))
		})
	}
}

func TestFlagString_ReturnsFlagName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		flag     Flag
		expected string
	}{
		{All, "ALL"},
		{None, "NONE"},
		{Single, "SINGLE"},
		{All | AnyOneCanPay, "ALL|ANYONECANPAY"},
		{None | AnyOneCanPay, "NONE|ANYONECANPAY"},
		{Single | AnyOneCanPay, "SINGLE|ANYONECANPAY"},
		{AllForkID, "ALL|FORKID"},
		{NoneForkID, "NONE|FORKID"},
		{SingleForkID, "SINGLE|FORKID"},
		{AllForkID | AnyOneCanPay, "ALL|FORKID|ANYONECANPAY"},
		{NoneForkID | AnyOneCanPay, "NONE|FORKID|ANYONECANPAY"},
		{SingleForkID | AnyOneCanPay, "SINGLE|FORKID|ANYONECANPAY"},
		{Old, "ALL"},
		{Flag(0xFF), "ALL"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.flag.String())
		})
	}
}
