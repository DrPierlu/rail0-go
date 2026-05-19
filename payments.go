package rail0

import (
	"context"
)

// PaymentsService implements the payment lifecycle operations.
type PaymentsService struct {
	http *httpClient
}

// CreatePayment creates a payment intent and returns the EIP-712 signingPayload for the payer to sign.
func (s *PaymentsService) CreatePayment(ctx context.Context, params CreatePaymentRequest) (*CreatePaymentResponse, error) {
	var out CreatePaymentResponse
	if err := s.http.post(ctx, "/payments", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Sign deposits the payer's EIP-712 signature (v, r, s).
func (s *PaymentsService) Sign(ctx context.Context, paymentID Bytes32, params PayerSignatureRequest) (*PayerSignatureResponse, error) {
	var out PayerSignatureResponse
	if err := s.http.put(ctx, "/payments/"+paymentID+"/sign", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Authorize relays the stored EIP-3009 signature to the RAIL0 authorize() function. Called by the payee.
func (s *PaymentsService) Authorize(ctx context.Context, paymentID Bytes32) (*AuthorizePaymentResponse, error) {
	var out AuthorizePaymentResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/authorize", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Charge relays the stored EIP-3009 signature to the RAIL0 charge() function (one-shot). Called by the payee.
func (s *PaymentsService) Charge(ctx context.Context, paymentID Bytes32) (*ChargePaymentResponse, error) {
	var out ChargePaymentResponse
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

// SubmitCapture broadcasts a signed capture transaction. Called by the payee.
func (s *PaymentsService) SubmitCapture(ctx context.Context, paymentID Bytes32, params SubmitTransactionRequest) (*CapturePaymentResponse, error) {
	var out CapturePaymentResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/capture/submit", params, &out); err != nil {
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

// SubmitVoid broadcasts a signed void transaction. Called by the payee.
func (s *PaymentsService) SubmitVoid(ctx context.Context, paymentID Bytes32, params SubmitTransactionRequest) (*VoidPaymentResponse, error) {
	var out VoidPaymentResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/void/submit", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Release releases escrowed funds back to the payer after AuthorizationExpiry. Permissionless.
func (s *PaymentsService) Release(ctx context.Context, paymentID Bytes32) (*ReleasePaymentResponse, error) {
	var out ReleasePaymentResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/release", nil, &out); err != nil {
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

// SubmitApprove broadcasts a signed ERC-20 approve transaction. Called by the payee.
func (s *PaymentsService) SubmitApprove(ctx context.Context, paymentID Bytes32, params SubmitTransactionRequest) (*ApproveResponse, error) {
	var out ApproveResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/approve/submit", params, &out); err != nil {
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

// SubmitRefund broadcasts a signed refund transaction. Called by the payee.
func (s *PaymentsService) SubmitRefund(ctx context.Context, paymentID Bytes32, params SubmitTransactionRequest) (*RefundPaymentResponse, error) {
	var out RefundPaymentResponse
	if err := s.http.post(ctx, "/payments/"+paymentID+"/refund/submit", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
