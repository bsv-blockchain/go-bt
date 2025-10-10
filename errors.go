package bt

import "github.com/pkg/errors"

// General errors.
var (
	ErrInvalidTxID       = errors.New("invalid TxID")
	ErrTxNil             = errors.New("tx is nil")
	ErrTxTooShort        = errors.New("too short to be a tx - even an empty tx has 10 bytes")
	ErrNLockTimeLength   = errors.New("nLockTime length must be 4 bytes long")
	ErrEmptyValues       = errors.New("empty value or values passed, all arguments are required and cannot be empty")
	ErrUnsupportedScript = errors.New("non-P2PKH input used in the tx - unsupported")
	ErrInvalidScriptType = errors.New("invalid script type")
	ErrNoUnlocker        = errors.New("unlocker not supplied")
)

// Sentinel errors reported by inputs.
var (
	ErrInputNoExist  = errors.New("specified input does not exist")
	ErrInputTooShort = errors.New("input length too short")

	// You should not be able to spend an input with 0 Satoshi value.
	// Most likely, the input Satoshi value is not provided.

	// ErrInputSatsZero is returned when the input Satoshi value is not provided.
	ErrInputSatsZero = errors.New("input satoshi value is not provided")
)

// Sentinel errors reported by outputs.
var (
	ErrOutputNoExist  = errors.New("specified output does not exist")
	ErrOutputTooShort = errors.New("output length too short")
)

// Sentinel errors reported by change.
var (

	// ErrInsufficientInputs is returned when the total inputted satoshis are less than the outputted satoshis.
	ErrInsufficientInputs = errors.New("satoshis inputted to the tx are less than the outputted satoshis")
)

// Sentinel errors reported by signature hash.
var (
	ErrEmptyPreviousTxID     = errors.New("'PreviousTxID' not supplied")
	ErrEmptyPreviousTxScript = errors.New("'PreviousTxScript' not supplied")
)

// Sentinel errors reported by the fees.
var (
	ErrFeeQuotesNotInit = errors.New("feeQuotes have not been setup, call NewFeeQuotes")
	ErrMinerNoQuotes    = errors.New("miner has no quotes stored")
	ErrFeeTypeNotFound  = errors.New("feetype not found")
	ErrFeeQuoteNotInit  = errors.New("feeQuote has not been initialized, call NewFeeQuote()")
	ErrUnknownFeeType   = errors.New("unknown fee type")
)

// Sentinel errors reported by the Fund.
var (
	// ErrNoUTXO signals the UTXOGetterFunc has reached the end of its input.
	ErrNoUTXO = errors.New("no remaining utxos")

	// ErrInsufficientFunds insufficient funds provided for funding
	ErrInsufficientFunds = errors.New("insufficient funds provided")
)

// Sentinel errors reported by ordinal inscriptions.
var (
	// ErrOutputsNotEmpty is returned when the transaction outputs are not empty
	ErrOutputsNotEmpty = errors.New("transaction outputs must be empty to avoid messing with Ordinal ordering scheme")
)

// Sentinel errors reported by PSBTs.
var (
	ErrDummyInput            = errors.New("failed to add dummy input 0")
	ErrInsufficientUTXOs     = errors.New("need at least 2 utxos")
	ErrInsufficientUTXOValue = errors.New("need at least 1 utxos which is > ordinal price")
	ErrUTXOInputMismatch     = errors.New("utxo and input mismatch")
	ErrInvalidSellOffer      = errors.New("invalid sell offer (partially signed tx)")
	ErrEmptyScripts          = errors.New("at least one of needed scripts is empty")
	ErrInsufficientFees      = errors.New("fee paid not enough with new locking script")
	ErrUnlockerNotFound      = errors.New("UTXO unlocker not found")
)
