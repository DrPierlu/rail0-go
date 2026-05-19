package rail0

import (
	"context"
	"fmt"
)

// PaymentsService implements the payment lifecycle operations.
type PaymentsService struct {
	http *httpClient
}

// Get returns the current on-chain state and config hash for a payment.
func (s *PaymentsService) Get(ctx context.Context, paymentID Bytes32) (*PaymentResponse, error) {
	var out PaymentResponse
	if err := s.http.get(ctx, "/payments/"+paymentID, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Authorize pulls Amount from the payer into escrow using an EIP-3009 transferWithAuthorization signature.
// Use SignAuthorize to build params.V, params.R, params.S off-chain.
func (s *PaymentsService) Authorize(ctx context.Context, paymentID Bytes32, params AuthorizeParams) (*TransactionResponse, error) {
	var out TransactionResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/authorize", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Charge authorizes and immediately captures in a single transaction — funds go directly to the payee.
// Use SignCharge to build the EIP-3009 signature in params.
func (s *PaymentsService) Charge(ctx context.Context, paymentID Bytes32, params ChargeParams) (*TransactionResponse, error) {
	var out TransactionResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/charge", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Capture moves escrowed funds to the payee. The API submits the transaction on behalf of the payee.
func (s *PaymentsService) Capture(ctx context.Context, paymentID Bytes32, params CaptureParams) (*TransactionResponse, error) {
	var out TransactionResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/capture", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Void cancels an authorization, returning all escrowed funds to the payer. Caller must be the payee.
func (s *PaymentsService) Void(ctx context.Context, paymentID Bytes32, params VoidParams) (*TransactionResponse, error) {
	var out TransactionResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/void", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Release returns escrowed funds to the payer after AuthorizationExpiry has passed without a capture.
// Permissionless — anyone may call it.
func (s *PaymentsService) Release(ctx context.Context, paymentID Bytes32, params ReleaseParams) (*TransactionResponse, error) {
	var out TransactionResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/release", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Refund returns a previously captured amount from the payee to the payer. Must be called before RefundExpiry.
func (s *PaymentsService) Refund(ctx context.Context, paymentID Bytes32, params RefundParams) (*TransactionResponse, error) {
	var out TransactionResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/refund", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AuthorizeNonce returns the EIP-3009 nonce the payer must include when signing an Authorize call.
// configHash is the EIP-712 digest of the Payment configuration (from Payments.Hash).
// Pass the returned Nonce into SignAuthorize.
func (s *PaymentsService) AuthorizeNonce(ctx context.Context, paymentID Bytes32, configHash Bytes32) (*NonceResponse, error) {
	var out NonceResponse
	path := fmt.Sprintf("/payments/%s/authorize-nonce?configHash=%s", paymentID, configHash)
	if err := s.http.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ChargeNonce returns the EIP-3009 nonce the payer must include when signing a Charge call.
// configHash is the EIP-712 digest of the Payment configuration (from Payments.Hash).
// Pass the returned Nonce into SignCharge.
func (s *PaymentsService) ChargeNonce(ctx context.Context, paymentID Bytes32, configHash Bytes32) (*NonceResponse, error) {
	var out NonceResponse
	path := fmt.Sprintf("/payments/%s/charge-nonce?configHash=%s", paymentID, configHash)
	if err := s.http.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Hash computes the canonical EIP-712 digest of a Payment configuration.
// Useful to pre-compute the config hash before calling Authorize or Charge.
func (s *PaymentsService) Hash(ctx context.Context, payment Payment) (*HashResponse, error) {
	var out HashResponse
	if err := s.http.post(ctx, "/payments/hash", payment, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
