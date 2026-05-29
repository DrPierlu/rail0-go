package rail0

import (
	"context"
)

// PaymentsService implements the payment lifecycle operations.
type PaymentsService struct {
	http *httpClient
}

// List returns all payments for the authenticated wallet (requires JWT).
func (s *PaymentsService) List(ctx context.Context) ([]PaymentResponse, error) {
	var out []PaymentResponse
	if err := s.http.get(ctx, "/payments", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Get fetches the current payment state (DB status + live on-chain amounts).
func (s *PaymentsService) Get(ctx context.Context, paymentID Bytes32) (*PaymentResponse, error) {
	var out PaymentResponse
	if err := s.http.get(ctx, "/payments/"+paymentID, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreatePayment creates a payment intent and returns the EIP-712 signingPayload for the payer to sign.
func (s *PaymentsService) CreatePayment(ctx context.Context, params CreatePaymentRequest) (*CreatePaymentResponse, error) {
	var out CreatePaymentResponse
	if err := s.http.post(ctx, "/payments", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Sign deposits the payer's EIP-712 signature. params.Signature must be the
// 65-byte secp256k1 signature as a 0x-prefixed 130-char hex string (r+s+v).
func (s *PaymentsService) Sign(ctx context.Context, paymentID Bytes32, params PayerSignatureRequest) (*PayerSignatureResponse, error) {
	var out PayerSignatureResponse
	if err := s.http.put(ctx, "/payments/"+paymentID+"/sign", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Authorize relays the stored EIP-3009 signature to the RAIL0 authorize() function. Called by the payee.
func (s *PaymentsService) Authorize(ctx context.Context, paymentID Bytes32) (*PrepareTransactionResponse, error) {
	var out PrepareTransactionResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/authorize", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Charge builds the unsigned charge() transaction. The payer signature must have been
// submitted first via Sign. Charge is a one-shot alternative to Authorize+Capture:
// funds are transferred directly to the payee without an escrow window.
// Sign the returned UnsignedTransaction and submit it with Submit.
func (s *PaymentsService) Charge(ctx context.Context, paymentID Bytes32) (*PrepareTransactionResponse, error) {
	var out PrepareTransactionResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/charge", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// PrepareCapture builds the unsigned capture() transaction. Called by the payee.
func (s *PaymentsService) PrepareCapture(ctx context.Context, paymentID Bytes32, params CapturePaymentRequest) (*PrepareTransactionResponse, error) {
	var out PrepareTransactionResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/capture", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// PrepareVoid builds the unsigned void() transaction. Called by the payee.
func (s *PaymentsService) PrepareVoid(ctx context.Context, paymentID Bytes32) (*PrepareTransactionResponse, error) {
	var out PrepareTransactionResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/void", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// PrepareRelease builds the unsigned release() transaction. Pass CallerAddress to build the tx
// for the buyer; omit (or pass empty) to build it for the payee.
// Release can only be called after AuthorizationExpiry has passed on-chain.
func (s *PaymentsService) PrepareRelease(ctx context.Context, paymentID Bytes32, params ReleaseRequest) (*PrepareTransactionResponse, error) {
	var out PrepareTransactionResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/release", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// PrepareApprove builds the unsigned ERC-20 approve() transaction needed before a refund. Called by the payee.
func (s *PaymentsService) PrepareApprove(ctx context.Context, paymentID Bytes32, params ApproveRequest) (*PrepareTransactionResponse, error) {
	var out PrepareTransactionResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/approve", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// PrepareRefund builds the unsigned refund() transaction. Called by the payee.
func (s *PaymentsService) PrepareRefund(ctx context.Context, paymentID Bytes32, params RefundPaymentRequest) (*PrepareTransactionResponse, error) {
	var out PrepareTransactionResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/refund", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Submit enqueues a signed transaction for asynchronous broadcast.
// The operation is inferred from pending_operation set by the preceding prepare step
// (Authorize, Charge, PrepareCapture, PrepareVoid, PrepareRelease, PrepareApprove, PrepareRefund).
//
// Returns HTTP 202 immediately with status "submitting". Poll Payments.Get until
// status leaves "submitting" to learn the final on-chain outcome.
func (s *PaymentsService) Submit(ctx context.Context, paymentID Bytes32, params SubmitTransactionRequest) (*SubmitTransactionAcceptedResponse, error) {
	var out SubmitTransactionAcceptedResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/transactions/submit", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
