package rail0

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
	ChainID             int           `json:"chainId"`
	AuthorizationExpiry int64         `json:"authorizationExpiry"`
	RefundExpiry        int64         `json:"refundExpiry"`
	OnChain             *OnChainState `json:"onChain,omitempty"`
}

// ReleaseRequest is the optional body for Payments.PrepareRelease.
// Omit CallerAddress (or leave empty) to build the transaction for the payee;
// pass the payer address to let the payer submit the release themselves.
type ReleaseRequest struct {
	CallerAddress Address `json:"callerAddress,omitempty"`
}

// SubmitApproveRequest is the body for Payments.SubmitApprove.
// Amount is optional but recommended so the API can record the approved amount.
type SubmitApproveRequest struct {
	SignedTransaction string        `json:"signedTransaction"`
	Amount           Uint256String `json:"amount,omitempty"`
}

// APIErrorBody is the JSON shape of error responses from the RAIL0 API.
// Used internally by the HTTP client to parse non-2xx responses.
type APIErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
