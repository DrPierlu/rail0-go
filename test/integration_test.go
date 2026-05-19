// Package rail0_test contains integration tests for the RAIL0 Go SDK.
//
// Tests in this file verify that every endpoint is correctly serialised/deserialised
// over real HTTP using net/http/httptest — no mocked transport. The mock server
// returns fixed, correctly-typed responses for each route.
package rail0_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	rail0 "github.com/rail0/go-sdk"
)

// ================================================================
//  Mock server
// ================================================================

var (
	bytes32RE = regexp.MustCompile(`^0x[0-9a-fA-F]{64}$`)
)

// newTestServer creates an httptest.Server that serves all RAIL0 API routes
// with fixture responses and returns a connected Client.
func newTestServer(t *testing.T) *rail0.Client {
	t.Helper()

	respond := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(v)
	}

	createPaymentFixture := map[string]any{
		"paymentId":  "0x1111111111111111111111111111111111111111111111111111111111111111",
		"configHash": "0xcccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		"payment": map[string]any{
			"payer": "0xBuyerAddress0000000000000000000000000000",
			"payee": "0xMerchantAddress00000000000000000000000000",
			"token": "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
			"maxAmount": "100000000", "authorizationExpiry": 9999999999, "refundExpiry": 9999999999,
			"feeBps": 0, "feeReceiver": "0x0000000000000000000000000000000000000000",
		},
		"amount":  "50000000",
		"chainId": 8453,
		"rail0Contract": "0x4444444444444444444444444444444444444444",
		"signingPayload": map[string]any{
			"domain": map[string]any{"name": "USD Coin", "version": "2", "chainId": 8453, "verifyingContract": "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"},
			"types": map[string]any{"TransferWithAuthorization": []any{}},
			"primaryType": "TransferWithAuthorization",
			"message": map[string]any{
				"from": "0xBuyerAddress0000000000000000000000000000",
				"to": "0x4444444444444444444444444444444444444444",
				"value": "50000000", "validAfter": "0", "validBefore": "9999999999",
				"nonce": "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			},
		},
	}

	sigFixture := map[string]any{
		"paymentId": "0x1111111111111111111111111111111111111111111111111111111111111111",
		"status":    "signature_stored",
	}

	authorizeFixture := map[string]any{
		"paymentId":           "0x1111111111111111111111111111111111111111111111111111111111111111",
		"transactionHash":     "0xdddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		"capturableAmount":    "50000000",
		"authorizationExpiry": 9999999999,
	}

	chargeFixture := map[string]any{
		"paymentId":        "0x1111111111111111111111111111111111111111111111111111111111111111",
		"transactionHash":  "0xdddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		"chargedAmount":    "50000000",
		"feeAmount":        "0",
		"refundableAmount": "50000000",
	}

	prepareFixture := map[string]any{
		"unsignedTransaction":  "0x02f8beef",
		"to":                   "0x4444444444444444444444444444444444444444",
		"data":                 "0x",
		"chainId":              8453,
		"nonce":                1,
		"maxFeePerGas":         "1000000000",
		"maxPriorityFeePerGas": "1000000000",
		"gasLimit":             "100000",
	}

	captureSubmitFixture := map[string]any{
		"paymentId":        "0x1111111111111111111111111111111111111111111111111111111111111111",
		"transactionHash":  "0xdddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		"capturedAmount":   "50000000",
		"capturableAmount": "0",
		"refundableAmount": "50000000",
	}

	voidSubmitFixture := map[string]any{
		"paymentId":       "0x1111111111111111111111111111111111111111111111111111111111111111",
		"transactionHash": "0xdddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		"releasedAmount":  "50000000",
	}

	releaseFixture := map[string]any{
		"paymentId":       "0x1111111111111111111111111111111111111111111111111111111111111111",
		"transactionHash": "0xdddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		"releasedAmount":  "50000000",
	}

	approveSubmitFixture := map[string]any{
		"transactionHash": "0xdddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		"token":           "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
		"spender":         "0x4444444444444444444444444444444444444444",
		"amount":          "1000000",
	}

	refundSubmitFixture := map[string]any{
		"paymentId":        "0x1111111111111111111111111111111111111111111111111111111111111111",
		"transactionHash":  "0xdddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		"refundedAmount":   "10000000",
		"refundableAmount": "40000000",
	}

	mux := http.NewServeMux()

	// POST /payments
	mux.HandleFunc("POST /payments", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		respond(w, createPaymentFixture)
	})

	// PUT /payments/{id}/sign
	mux.HandleFunc("PUT /payments/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/payments/")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) == 2 && parts[1] == "sign" {
			respond(w, sigFixture)
		} else {
			http.NotFound(w, r)
		}
	})

	// POST /payments/{id}/...
	mux.HandleFunc("POST /payments/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/payments/")
		parts := strings.SplitN(path, "/", 3)
		if len(parts) < 2 {
			http.NotFound(w, r)
			return
		}
		action := strings.Join(parts[1:], "/")
		switch action {
		case "authorize":
			respond(w, authorizeFixture)
		case "charge":
			respond(w, chargeFixture)
		case "capture":
			respond(w, prepareFixture)
		case "capture/submit":
			respond(w, captureSubmitFixture)
		case "void":
			respond(w, prepareFixture)
		case "void/submit":
			respond(w, voidSubmitFixture)
		case "release":
			respond(w, releaseFixture)
		case "approve":
			respond(w, prepareFixture)
		case "approve/submit":
			respond(w, approveSubmitFixture)
		case "refund":
			respond(w, prepareFixture)
		case "refund/submit":
			respond(w, refundSubmitFixture)
		default:
			http.NotFound(w, r)
		}
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return rail0.NewClient(rail0.ClientOptions{BaseURL: srv.URL})
}

// ================================================================
//  Test fixtures
// ================================================================

const integrationPaymentID = "0x1111111111111111111111111111111111111111111111111111111111111111"

var integrationPayment = rail0.PaymentConfig{
	Payer:               "0xBuyerAddress0000000000000000000000000000",
	Payee:               "0xMerchantAddress00000000000000000000000000",
	Token:               "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
	MaxAmount:           "100000000",
	AuthorizationExpiry: 9999999999,
	RefundExpiry:        9999999999,
	FeeBps:              0,
	FeeReceiver:         "0x0000000000000000000000000000000000000000",
}

var integrationSig = struct{ V int; R, S rail0.Bytes32 }{
	V: 27,
	R: "0x1111111111111111111111111111111111111111111111111111111111111111",
	S: "0x2222222222222222222222222222222222222222222222222222222222222222",
}

// ================================================================
//  Payments
// ================================================================

func TestIntegration_CreatePayment(t *testing.T) {
	client := newTestServer(t)
	res, err := client.Payments.CreatePayment(context.Background(), rail0.CreatePaymentRequest{
		Payment: integrationPayment,
		Amount:  "50000000",
		ChainID: 8453,
		Mode:    "authorize",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes32RE.MatchString(res.PaymentID) {
		t.Errorf("paymentId format invalid: %s", res.PaymentID)
	}
	if !bytes32RE.MatchString(res.ConfigHash) {
		t.Errorf("configHash format invalid: %s", res.ConfigHash)
	}
}

func TestIntegration_Sign(t *testing.T) {
	client := newTestServer(t)
	res, err := client.Payments.Sign(context.Background(), integrationPaymentID, rail0.PayerSignatureRequest{
		V: integrationSig.V,
		R: integrationSig.R,
		S: integrationSig.S,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status == "" {
		t.Error("status should not be empty")
	}
}

func TestIntegration_Authorize(t *testing.T) {
	client := newTestServer(t)
	res, err := client.Payments.Authorize(context.Background(), integrationPaymentID)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes32RE.MatchString(res.TransactionHash) {
		t.Errorf("transactionHash format invalid: %s", res.TransactionHash)
	}
	if res.CapturableAmount == "" {
		t.Error("capturableAmount should not be empty")
	}
}

func TestIntegration_Charge(t *testing.T) {
	client := newTestServer(t)
	res, err := client.Payments.Charge(context.Background(), integrationPaymentID)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes32RE.MatchString(res.TransactionHash) {
		t.Errorf("transactionHash format invalid: %s", res.TransactionHash)
	}
	if res.ChargedAmount == "" {
		t.Error("chargedAmount should not be empty")
	}
}

func TestIntegration_PrepareCapture(t *testing.T) {
	client := newTestServer(t)
	res, err := client.Payments.PrepareCapture(context.Background(), integrationPaymentID, rail0.CapturePaymentRequest{
		Amount: "50000000",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.UnsignedTransaction == "" {
		t.Error("unsignedTransaction should not be empty")
	}
}

func TestIntegration_SubmitCapture(t *testing.T) {
	client := newTestServer(t)
	res, err := client.Payments.SubmitCapture(context.Background(), integrationPaymentID, rail0.SubmitTransactionRequest{
		SignedTransaction: "0x02f8...",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes32RE.MatchString(res.TransactionHash) {
		t.Errorf("transactionHash format invalid: %s", res.TransactionHash)
	}
}

func TestIntegration_PrepareVoid(t *testing.T) {
	client := newTestServer(t)
	res, err := client.Payments.PrepareVoid(context.Background(), integrationPaymentID)
	if err != nil {
		t.Fatal(err)
	}
	if res.UnsignedTransaction == "" {
		t.Error("unsignedTransaction should not be empty")
	}
}

func TestIntegration_SubmitVoid(t *testing.T) {
	client := newTestServer(t)
	res, err := client.Payments.SubmitVoid(context.Background(), integrationPaymentID, rail0.SubmitTransactionRequest{
		SignedTransaction: "0x02f8...",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes32RE.MatchString(res.TransactionHash) {
		t.Errorf("transactionHash format invalid: %s", res.TransactionHash)
	}
}

func TestIntegration_Release(t *testing.T) {
	client := newTestServer(t)
	res, err := client.Payments.Release(context.Background(), integrationPaymentID)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes32RE.MatchString(res.TransactionHash) {
		t.Errorf("transactionHash format invalid: %s", res.TransactionHash)
	}
}

func TestIntegration_PrepareApprove(t *testing.T) {
	client := newTestServer(t)
	res, err := client.Payments.PrepareApprove(context.Background(), integrationPaymentID, rail0.ApproveRequest{
		Amount: "115792089237316195423570985008687907853269984665640564039457584007913129639935",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.UnsignedTransaction == "" {
		t.Error("unsignedTransaction should not be empty")
	}
}

func TestIntegration_SubmitApprove(t *testing.T) {
	client := newTestServer(t)
	res, err := client.Payments.SubmitApprove(context.Background(), integrationPaymentID, rail0.SubmitTransactionRequest{
		SignedTransaction: "0x02f8...",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes32RE.MatchString(res.TransactionHash) {
		t.Errorf("transactionHash format invalid: %s", res.TransactionHash)
	}
}

func TestIntegration_PrepareRefund(t *testing.T) {
	client := newTestServer(t)
	res, err := client.Payments.PrepareRefund(context.Background(), integrationPaymentID, rail0.RefundPaymentRequest{
		Amount: "10000000",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.UnsignedTransaction == "" {
		t.Error("unsignedTransaction should not be empty")
	}
}

func TestIntegration_SubmitRefund(t *testing.T) {
	client := newTestServer(t)
	res, err := client.Payments.SubmitRefund(context.Background(), integrationPaymentID, rail0.SubmitTransactionRequest{
		SignedTransaction: "0x02f8...",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes32RE.MatchString(res.TransactionHash) {
		t.Errorf("transactionHash format invalid: %s", res.TransactionHash)
	}
	if res.RefundedAmount == "" {
		t.Error("refundedAmount should not be empty")
	}
}
