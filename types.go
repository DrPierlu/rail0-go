package rail0

// OnChainState holds live on-chain escrow balances for a payment.
type OnChainState struct {
	Exists           bool          `json:"exists"`
	CapturableAmount Uint256String `json:"capturableAmount"`
	RefundableAmount Uint256String `json:"refundableAmount"`
}

// Transaction is a single on-chain operation attempt associated with a payment.
type Transaction struct {
	Operation       string `json:"operation"`
	Status          string `json:"status"`
	TransactionHash string `json:"transactionHash"`
	Amount          string `json:"amount,omitempty"`
	BlockNumber     *int   `json:"blockNumber,omitempty"`
}

// PaymentResponse is returned by Payments.Get.
type PaymentResponse struct {
	Rail0Id             Bytes32        `json:"rail0_id"`
	Status              string         `json:"status"`
	Mode                string         `json:"mode"`
	Amount              Uint256String  `json:"amount"`
	Payer               Address        `json:"payer"`
	Payee               Address        `json:"payee"`
	Token               Address        `json:"token"`
	ChainID             int            `json:"chainId"`
	AuthorizationExpiry int64          `json:"authorizationExpiry"`
	RefundExpiry        int64          `json:"refundExpiry"`
	FeeBps              int            `json:"feeBps"`
	FeeReceiver         Address        `json:"feeReceiver"`
	OnChain             *OnChainState  `json:"onChain,omitempty"`
	// Populated only when status = "pending_signature" so the payer can sign locally.
	SigningPayload  *SigningPayload `json:"signingPayload,omitempty"`
	Rail0Contract  Address        `json:"rail0Contract,omitempty"`
	Transactions   []Transaction  `json:"transactions,omitempty"`
}

// ReleaseRequest is the optional body for Payments.PrepareRelease.
// Omit CallerAddress (or leave empty) to build the transaction for the payee;
// pass the payer address to let the payer submit the release themselves.
type ReleaseRequest struct {
	CallerAddress Address `json:"caller_address,omitempty"`
}

// SubmitTransactionAcceptedResponse is returned by Payments.Submit (HTTP 202).
// The submission is asynchronous — poll Payments.Get until status leaves "submitting"
// to learn whether the transaction was confirmed on-chain.
// Token and Spender are populated only when the pending operation is "approve".
type SubmitTransactionAcceptedResponse struct {
	Rail0Id string `json:"rail0_id"`
	Status  string `json:"status"` // always "submitting"
	Token   string `json:"token,omitempty"`
	Spender string `json:"spender,omitempty"`
}

// APIErrorBody is the JSON shape of error responses from the RAIL0 API.
// Used internally by the HTTP client to parse non-2xx responses.
type APIErrorBody struct {
	Code    string `json:"error"`
	Message string `json:"message"`
}
