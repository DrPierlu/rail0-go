package rail0

// Address is a checksummed or lowercase Ethereum address (42 chars, 0x-prefixed).
type Address = string

// Bytes32 is a 32-byte value hex-encoded with 0x prefix (66 chars total).
// Used for payment IDs, hashes, and signature components.
type Bytes32 = string

// Uint256String is an unsigned 256-bit integer serialised as a decimal string.
// Avoids precision loss for amounts that exceed int64.
type Uint256String = string

// PaymentConfig holds the immutable configuration committed on the first Authorize or Charge call.
// Every subsequent operation on the same paymentId must supply the exact same struct —
// a mismatch causes the contract to revert with PaymentMismatch.
type PaymentConfig struct {
	Payer               Address       `json:"payer"`
	Payee               Address       `json:"payee"`
	Token               Address       `json:"token"`
	MaxAmount           Uint256String `json:"maxAmount"`
	AuthorizationExpiry int64         `json:"authorizationExpiry"`
	RefundExpiry        int64         `json:"refundExpiry"`
	FeeBps              int           `json:"feeBps"`
	FeeReceiver         Address       `json:"feeReceiver"`
}

// EIP712Domain is the EIP-712 domain for the token contract.
type EIP712Domain struct {
	Name              string  `json:"name"`
	Version           string  `json:"version"`
	ChainID           int64   `json:"chainId"`
	VerifyingContract Address `json:"verifyingContract"`
}

// EIP712TypeEntry is a single field entry in EIP-712 type definitions.
type EIP712TypeEntry struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// EIP712Types holds the type definitions for the SigningPayload.
type EIP712Types struct {
	TransferWithAuthorization []EIP712TypeEntry `json:"TransferWithAuthorization"`
}

// EIP3009Message is the message fields for the EIP-3009 TransferWithAuthorization signature.
type EIP3009Message struct {
	From        Address       `json:"from"`
	To          Address       `json:"to"`
	Value       Uint256String `json:"value"`
	ValidAfter  Uint256String `json:"validAfter"`
	ValidBefore Uint256String `json:"validBefore"`
	Nonce       Bytes32       `json:"nonce"`
}

// SigningPayload is the EIP-712 typed-data structure returned by POST /payments.
// Pass verbatim to eth_signTypedData_v4, or compute the digest manually with any EIP-712 library.
type SigningPayload struct {
	Domain      EIP712Domain `json:"domain"`
	Types       EIP712Types  `json:"types"`
	PrimaryType string       `json:"primaryType"`
	Message     EIP3009Message `json:"message"`
}

// ================================================================
//  Request bodies
// ================================================================

// CreatePaymentRequest is the body for Payments.CreatePayment.
type CreatePaymentRequest struct {
	Payment PaymentConfig `json:"payment"`
	Amount  Uint256String `json:"amount"`
	ChainID int64         `json:"chainId"`
	Mode    string        `json:"mode"` // "authorize" or "charge"
}

// PayerSignatureRequest is the body for Payments.Sign.
type PayerSignatureRequest struct {
	V int     `json:"v"`
	R Bytes32 `json:"r"`
	S Bytes32 `json:"s"`
}

// CapturePaymentRequest is the body for Payments.PrepareCapture.
type CapturePaymentRequest struct {
	Amount Uint256String `json:"amount"`
}

// SubmitTransactionRequest is the body for submit operations (capture, void, approve, refund).
type SubmitTransactionRequest struct {
	SignedTransaction string `json:"signedTransaction"`
}

// ApproveRequest is the body for Payments.PrepareApprove.
type ApproveRequest struct {
	Amount Uint256String `json:"amount"`
}

// RefundPaymentRequest is the body for Payments.PrepareRefund.
type RefundPaymentRequest struct {
	Amount Uint256String `json:"amount"`
}

// ReleaseRequest is the optional body for Payments.PrepareRelease.
type ReleaseRequest struct {
	CallerAddress Address `json:"callerAddress,omitempty"`
}

// SubmitApproveRequest is the body for Payments.SubmitApprove.
// Amount is optional but recommended so the API can record it.
type SubmitApproveRequest struct {
	SignedTransaction string        `json:"signedTransaction"`
	Amount           Uint256String `json:"amount,omitempty"`
}

// OnChainState holds live on-chain escrow balances for a payment.
type OnChainState struct {
	Exists           bool          `json:"exists"`
	CapturableAmount Uint256String `json:"capturableAmount"`
	RefundableAmount Uint256String `json:"refundableAmount"`
}

// PaymentResponse is returned by Payments.Get.
type PaymentResponse struct {
	PaymentID           Bytes32       `json:"paymentId"`
	Status              string        `json:"status"`
	Mode                string        `json:"mode"`
	Amount              Uint256String `json:"amount"`
	Payer               Address       `json:"payer"`
	Payee               Address       `json:"payee"`
	Token               Address       `json:"token"`
	ChainID             int64         `json:"chainId"`
	AuthorizationExpiry int64         `json:"authorizationExpiry"`
	RefundExpiry        int64         `json:"refundExpiry"`
	OnChain             *OnChainState `json:"onChain,omitempty"`
}

// ================================================================
//  Response shapes
// ================================================================

// CreatePaymentResponse is returned by Payments.CreatePayment.
type CreatePaymentResponse struct {
	PaymentID      Bytes32        `json:"paymentId"`
	ConfigHash     Bytes32        `json:"configHash"`
	Payment        PaymentConfig  `json:"payment"`
	Amount         Uint256String  `json:"amount"`
	ChainID        int64          `json:"chainId"`
	Rail0Contract  Address        `json:"rail0Contract"`
	SigningPayload SigningPayload  `json:"signingPayload"`
}

// PayerSignatureResponse is returned by Payments.Sign.
type PayerSignatureResponse struct {
	PaymentID      Bytes32 `json:"paymentId"`
	Status         string  `json:"status"`
	RecoveredPayer Address `json:"recoveredPayer,omitempty"`
}

// AuthorizePaymentResponse is returned by Payments.Authorize.
type AuthorizePaymentResponse struct {
	PaymentID           Bytes32       `json:"paymentId"`
	TransactionHash     Bytes32       `json:"transactionHash"`
	CapturableAmount    Uint256String `json:"capturableAmount"`
	AuthorizationExpiry int64         `json:"authorizationExpiry,omitempty"`
}

// ChargePaymentResponse is returned by Payments.Charge.
type ChargePaymentResponse struct {
	PaymentID        Bytes32       `json:"paymentId"`
	TransactionHash  Bytes32       `json:"transactionHash"`
	ChargedAmount    Uint256String `json:"chargedAmount"`
	FeeAmount        Uint256String `json:"feeAmount"`
	RefundableAmount Uint256String `json:"refundableAmount"`
}

// PrepareTransactionResponse is returned by prepare operations.
// It contains an unsigned EIP-1559 transaction ready for the payee to sign.
type PrepareTransactionResponse struct {
	UnsignedTransaction    string        `json:"unsignedTransaction"`
	To                     Address       `json:"to"`
	Data                   string        `json:"data"`
	ChainID                int64         `json:"chainId"`
	Nonce                  int64         `json:"nonce"`
	MaxFeePerGas           Uint256String `json:"maxFeePerGas"`
	MaxPriorityFeePerGas   Uint256String `json:"maxPriorityFeePerGas"`
	GasLimit               Uint256String `json:"gasLimit"`
}

// CapturePaymentResponse is returned by Payments.SubmitCapture.
type CapturePaymentResponse struct {
	PaymentID           Bytes32       `json:"paymentId"`
	TransactionHash     Bytes32       `json:"transactionHash"`
	CapturedAmount      Uint256String `json:"capturedAmount"`
	FeeAmount           Uint256String `json:"feeAmount,omitempty"`
	CapturableAmount    Uint256String `json:"capturableAmount"`
	RefundableAmount    Uint256String `json:"refundableAmount"`
	AuthorizationExpiry int64         `json:"authorizationExpiry,omitempty"`
}

// VoidPaymentResponse is returned by Payments.SubmitVoid.
type VoidPaymentResponse struct {
	PaymentID       Bytes32       `json:"paymentId"`
	TransactionHash Bytes32       `json:"transactionHash"`
	ReleasedAmount  Uint256String `json:"releasedAmount"`
}

// ReleasePaymentResponse is returned by Payments.Release.
type ReleasePaymentResponse struct {
	PaymentID       Bytes32       `json:"paymentId"`
	TransactionHash Bytes32       `json:"transactionHash"`
	ReleasedAmount  Uint256String `json:"releasedAmount"`
}

// ApproveResponse is returned by Payments.SubmitApprove.
type ApproveResponse struct {
	TransactionHash Bytes32       `json:"transactionHash"`
	Token           Address       `json:"token"`
	Spender         Address       `json:"spender"`
	Amount          Uint256String `json:"amount"`
}

// RefundPaymentResponse is returned by Payments.SubmitRefund.
type RefundPaymentResponse struct {
	PaymentID        Bytes32       `json:"paymentId"`
	TransactionHash  Bytes32       `json:"transactionHash"`
	RefundedAmount   Uint256String `json:"refundedAmount"`
	RefundableAmount Uint256String `json:"refundableAmount"`
}

// APIErrorBody is the shape of error responses from the RAIL0 API.
type APIErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
