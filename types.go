package rail0

// Address is a checksummed or lowercase Ethereum address (42 chars, 0x-prefixed).
type Address = string

// Bytes32 is a 32-byte value hex-encoded with 0x prefix (66 chars total).
// Used for payment IDs, hashes, and signature components.
type Bytes32 = string

// Uint256String is an unsigned 256-bit integer serialised as a decimal string.
// Avoids precision loss for amounts that exceed int64.
type Uint256String = string

// Payment holds the immutable configuration committed on the first Authorize or Charge call.
// Every subsequent operation on the same paymentId must supply the exact same struct —
// a mismatch causes the contract to revert with PaymentMismatch.
type Payment struct {
	// Payer is the buyer address that signs the EIP-3009 authorization.
	Payer Address `json:"payer"`
	// Payee is the merchant address that must call Capture, Void, or Refund.
	Payee Address `json:"payee"`
	// Token is the ERC-20 address. Must be in the allowlist and support EIP-3009.
	Token Address `json:"token"`
	// MaxAmount is the upper bound on what can be authorized (fits in uint120 on-chain).
	MaxAmount Uint256String `json:"maxAmount"`
	// AuthorizationExpiry is the Unix timestamp after which Capture is rejected and Release opens.
	AuthorizationExpiry int64 `json:"authorizationExpiry"`
	// RefundExpiry is the Unix timestamp after which Refund is rejected.
	RefundExpiry int64 `json:"refundExpiry"`
	// FeeBps is the protocol fee in basis points (0 = no fee).
	FeeBps int `json:"feeBps"`
	// FeeReceiver receives the protocol fee. Must be zero when FeeBps is 0.
	FeeReceiver Address `json:"feeReceiver"`
}

// PaymentState is the on-chain mutable state for a payment, packed in one storage slot.
// CapturableAmount holds escrowed funds (authorize → capture/void/release path).
// RefundableAmount holds already-disbursed funds (capture → refund path).
type PaymentState struct {
	// Exists is true once a payment has been created via Authorize or Charge.
	Exists bool `json:"exists"`
	// CapturableAmount is the escrowed balance available for Capture or Release.
	CapturableAmount Uint256String `json:"capturableAmount"`
	// RefundableAmount is the balance already sent to the payee but still eligible for Refund.
	RefundableAmount Uint256String `json:"refundableAmount"`
}

// AuthorizeParams is the request body for Authorize and Charge.
// V, R, S are the EIP-3009 transferWithAuthorization signature produced by the payer's key.
// Use SignAuthorize or SignCharge to build the signature off-chain.
type AuthorizeParams struct {
	Payment Payment       `json:"payment"`
	Amount  Uint256String `json:"amount"`
	// V is the recovery identifier from the EIP-3009 signature (27 or 28).
	V int     `json:"v"`
	R Bytes32 `json:"r"`
	S Bytes32 `json:"s"`
}

// ChargeParams is an alias for AuthorizeParams (one-shot authorize + capture).
type ChargeParams = AuthorizeParams

// CaptureParams is the request body for Capture.
type CaptureParams struct {
	Payment Payment       `json:"payment"`
	Amount  Uint256String `json:"amount"`
}

// VoidParams is the request body for Void.
type VoidParams struct {
	Payment Payment `json:"payment"`
}

// ReleaseParams is the request body for Release (permissionless after AuthorizationExpiry).
type ReleaseParams = VoidParams

// RefundParams is the request body for Refund.
type RefundParams = CaptureParams

// PaymentResponse is the full on-chain state returned by Get.
type PaymentResponse struct {
	PaymentID  Bytes32      `json:"paymentId"`
	State      PaymentState `json:"state"`
	// ConfigHash is the EIP-712 digest of the Payment configuration committed on creation.
	ConfigHash Bytes32      `json:"configHash"`
}

// TransactionResponse is returned by every write operation.
// The transaction may still be pending confirmation on-chain.
type TransactionResponse struct {
	TransactionHash Bytes32 `json:"transactionHash"`
	// Status is one of "pending", "confirmed", or "failed".
	Status string `json:"status"`
}

// TokenStatusResponse is returned by IsAccepted.
type TokenStatusResponse struct {
	Address  Address `json:"address"`
	Accepted bool    `json:"accepted"`
}

// NonceResponse is returned by AuthorizeNonce and ChargeNonce.
// Pass the Nonce value into SignAuthorize or SignCharge when building the signature.
type NonceResponse struct {
	Nonce Bytes32 `json:"nonce"`
}

// HashResponse holds the EIP-712 digest returned by Hash.
type HashResponse struct {
	Hash Bytes32 `json:"hash"`
}

// DomainSeparatorResponse holds the EIP-712 domain separator of the RAIL0 contract.
type DomainSeparatorResponse struct {
	DomainSeparator Bytes32 `json:"domainSeparator"`
}

// VersionResponse holds the contract version number.
type VersionResponse struct {
	Version int `json:"version"`
}

// APIErrorBody is the shape of error responses from the RAIL0 API.
type APIErrorBody struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
