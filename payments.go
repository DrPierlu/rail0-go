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

// ── Authorize ──────────────────────────────────────────────────────────────────

// AuthorizePayload builds the unsigned authorize() transaction. Creates a Transaction
// row in pending status. Sign the returned UnsignedTransaction and call Authorize.
func (s *PaymentsService) AuthorizePrepare(ctx context.Context, paymentID Bytes32) (*PrepareTransactionResponse, error) {
	var out PrepareTransactionResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/authorize/prepare", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Authorize submits the signed authorize transaction for asynchronous broadcast.
// Returns 202 immediately; poll Get until the payment reaches "authorized".
func (s *PaymentsService) Authorize(ctx context.Context, paymentID Bytes32, params SubmitTransactionRequest) (*SubmitTransactionAcceptedResponse, error) {
	var out SubmitTransactionAcceptedResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/authorize", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ── Charge ─────────────────────────────────────────────────────────────────────

// ChargePayload builds the unsigned charge() transaction (one-shot authorize+capture).
// Creates a Transaction row in pending status.
func (s *PaymentsService) ChargePrepare(ctx context.Context, paymentID Bytes32) (*PrepareTransactionResponse, error) {
	var out PrepareTransactionResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/charge/prepare", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Charge submits the signed charge transaction. Poll Get until "charged".
func (s *PaymentsService) Charge(ctx context.Context, paymentID Bytes32, params SubmitTransactionRequest) (*SubmitTransactionAcceptedResponse, error) {
	var out SubmitTransactionAcceptedResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/charge", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ── Capture ────────────────────────────────────────────────────────────────────

// CapturePayload builds the unsigned capture() transaction.
// Partial captures are supported: amount may be less than capturableAmount.
func (s *PaymentsService) CapturePrepare(ctx context.Context, paymentID Bytes32, params CapturePaymentRequest) (*PrepareTransactionResponse, error) {
	var out PrepareTransactionResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/capture/prepare", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Capture submits the signed capture transaction. Poll Get until "captured".
func (s *PaymentsService) Capture(ctx context.Context, paymentID Bytes32, params SubmitTransactionRequest) (*SubmitTransactionAcceptedResponse, error) {
	var out SubmitTransactionAcceptedResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/capture", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ── Void ───────────────────────────────────────────────────────────────────────

// VoidPayload builds the unsigned void() transaction.
func (s *PaymentsService) VoidPrepare(ctx context.Context, paymentID Bytes32) (*PrepareTransactionResponse, error) {
	var out PrepareTransactionResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/void/prepare", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Void submits the signed void transaction. Poll Get until "voided".
func (s *PaymentsService) Void(ctx context.Context, paymentID Bytes32, params SubmitTransactionRequest) (*SubmitTransactionAcceptedResponse, error) {
	var out SubmitTransactionAcceptedResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/void", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ── Release ────────────────────────────────────────────────────────────────────

// ReleasePayload builds the unsigned release() transaction.
// Pass CallerAddress to build for the payer; omit for the payee.
func (s *PaymentsService) ReleasePrepare(ctx context.Context, paymentID Bytes32, params ReleaseRequest) (*PrepareTransactionResponse, error) {
	var out PrepareTransactionResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/release/prepare", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Release submits the signed release transaction. Poll Get until "released".
func (s *PaymentsService) Release(ctx context.Context, paymentID Bytes32, params SubmitTransactionRequest) (*SubmitTransactionAcceptedResponse, error) {
	var out SubmitTransactionAcceptedResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/release", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ── Refund (EIP-3009) ─────────────────────────────────────────────────────────

// RefundPayload is a two-phase endpoint.
//
// Phase 1 — params.V/R/S empty: returns the EIP-3009 signing payload the payee
// must sign off-chain. No unsigned transaction is returned yet.
//
// Phase 2 — params.V/R/S set: builds the unsigned refund() transaction with the
// EIP-3009 signature embedded in calldata and returns it in UnsignedTransaction.
func (s *PaymentsService) RefundPrepare(ctx context.Context, paymentID Bytes32, params RefundPayloadRequest) (*RefundPayloadResponse, error) {
	var out RefundPayloadResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/refund/prepare", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Refund submits the signed refund transaction. Poll Get until "refunded".
func (s *PaymentsService) Refund(ctx context.Context, paymentID Bytes32, params SubmitTransactionRequest) (*SubmitTransactionAcceptedResponse, error) {
	var out SubmitTransactionAcceptedResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/refund", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
