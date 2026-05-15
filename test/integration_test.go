// Package rail0_test contains integration tests for the RAIL0 Go SDK.
//
// Tests in this file verify that every endpoint is correctly serialised/deserialised
// over real HTTP using net/http/httptest — no mocked transport. The mock server
// returns fixed, correctly-typed responses for each route, equivalent to the
// TypeScript mock/ server used by integration.test.ts.
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

// fixtures returns static responses that mirror mock/fixtures.ts.
var fixtures = struct {
	payment  rail0.PaymentResponse
	tx       rail0.TransactionResponse
	nonce    rail0.NonceResponse
	hash     rail0.HashResponse
	token    rail0.TokenStatusResponse
	domain   rail0.DomainSeparatorResponse
	version  rail0.VersionResponse
}{
	payment: rail0.PaymentResponse{
		PaymentID: "0x1111111111111111111111111111111111111111111111111111111111111111",
		State: rail0.PaymentState{
			Exists:           false,
			CapturableAmount: "0",
			RefundableAmount: "0",
		},
		ConfigHash: "0xcccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
	},
	tx: rail0.TransactionResponse{
		TransactionHash: "0xdddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		Status:          "pending",
	},
	nonce: rail0.NonceResponse{
		Nonce: "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
	},
	hash: rail0.HashResponse{
		Hash: "0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	},
	token: rail0.TokenStatusResponse{
		Address:  "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
		Accepted: true,
	},
	domain: rail0.DomainSeparatorResponse{
		DomainSeparator: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	},
	version: rail0.VersionResponse{Version: 6},
}

// newTestServer creates an httptest.Server that serves all RAIL0 API routes
// with fixture responses and returns a connected Client.
func newTestServer(t *testing.T) *rail0.Client {
	t.Helper()

	respond := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(v)
	}
	tx := func(w http.ResponseWriter) {
		w.WriteHeader(http.StatusAccepted)
		respond(w, fixtures.tx)
	}

	mux := http.NewServeMux()

	// GET /payments/{id}
	mux.HandleFunc("GET /payments/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/payments/"), "/")
		switch {
		case len(parts) == 1: // /payments/{id}
			respond(w, fixtures.payment)
		case parts[1] == "authorize-nonce":
			respond(w, fixtures.nonce)
		case parts[1] == "charge-nonce":
			respond(w, fixtures.nonce)
		default:
			http.NotFound(w, r)
		}
	})

	// POST /payments/hash (must be before the generic /payments/ handler)
	mux.HandleFunc("POST /payments/hash", func(w http.ResponseWriter, r *http.Request) {
		respond(w, fixtures.hash)
	})

	// POST /payments/{id}/{action}
	mux.HandleFunc("POST /payments/", func(w http.ResponseWriter, r *http.Request) {
		tx(w)
	})

	// GET /tokens/{address}
	mux.HandleFunc("GET /tokens/", func(w http.ResponseWriter, r *http.Request) {
		respond(w, fixtures.token)
	})

	// GET /domain-separator
	mux.HandleFunc("GET /domain-separator", func(w http.ResponseWriter, r *http.Request) {
		respond(w, fixtures.domain)
	})

	// GET /version
	mux.HandleFunc("GET /version", func(w http.ResponseWriter, r *http.Request) {
		respond(w, fixtures.version)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return rail0.NewClient(rail0.ClientOptions{BaseURL: srv.URL})
}

// ================================================================
//  Test fixtures
// ================================================================

const integrationPaymentID = "0x1111111111111111111111111111111111111111111111111111111111111111"

var integrationPayment = rail0.Payment{
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

func TestIntegration_PaymentsGet(t *testing.T) {
	client := newTestServer(t)
	res, err := client.Payments.Get(context.Background(), integrationPaymentID)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes32RE.MatchString(res.ConfigHash) {
		t.Errorf("configHash format invalid: %s", res.ConfigHash)
	}
}

func TestIntegration_PaymentsAuthorize(t *testing.T) {
	client := newTestServer(t)
	res, err := client.Payments.Authorize(context.Background(), integrationPaymentID, rail0.AuthorizeParams{
		Payment: integrationPayment,
		Amount:  "50000000",
		V:       integrationSig.V,
		R:       integrationSig.R,
		S:       integrationSig.S,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes32RE.MatchString(res.TransactionHash) {
		t.Errorf("transactionHash format invalid: %s", res.TransactionHash)
	}
	if res.Status != "pending" && res.Status != "confirmed" && res.Status != "failed" {
		t.Errorf("unexpected status: %s", res.Status)
	}
}

func TestIntegration_PaymentsCharge(t *testing.T) {
	client := newTestServer(t)
	res, err := client.Payments.Charge(context.Background(), integrationPaymentID, rail0.ChargeParams{
		Payment: integrationPayment,
		Amount:  "50000000",
		V:       integrationSig.V,
		R:       integrationSig.R,
		S:       integrationSig.S,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes32RE.MatchString(res.TransactionHash) {
		t.Errorf("transactionHash format invalid: %s", res.TransactionHash)
	}
}

func TestIntegration_PaymentsCapture(t *testing.T) {
	client := newTestServer(t)
	res, err := client.Payments.Capture(context.Background(), integrationPaymentID, rail0.CaptureParams{
		Payment: integrationPayment,
		Amount:  "50000000",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes32RE.MatchString(res.TransactionHash) {
		t.Errorf("transactionHash format invalid: %s", res.TransactionHash)
	}
}

func TestIntegration_PaymentsVoid(t *testing.T) {
	client := newTestServer(t)
	res, err := client.Payments.Void(context.Background(), integrationPaymentID, rail0.VoidParams{
		Payment: integrationPayment,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes32RE.MatchString(res.TransactionHash) {
		t.Errorf("transactionHash format invalid: %s", res.TransactionHash)
	}
}

func TestIntegration_PaymentsRelease(t *testing.T) {
	client := newTestServer(t)
	res, err := client.Payments.Release(context.Background(), integrationPaymentID, rail0.ReleaseParams{
		Payment: integrationPayment,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes32RE.MatchString(res.TransactionHash) {
		t.Errorf("transactionHash format invalid: %s", res.TransactionHash)
	}
}

func TestIntegration_PaymentsRefund(t *testing.T) {
	client := newTestServer(t)
	res, err := client.Payments.Refund(context.Background(), integrationPaymentID, rail0.RefundParams{
		Payment: integrationPayment,
		Amount:  "10000000",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes32RE.MatchString(res.TransactionHash) {
		t.Errorf("transactionHash format invalid: %s", res.TransactionHash)
	}
}

func TestIntegration_PaymentsAuthorizeNonce(t *testing.T) {
	client := newTestServer(t)
	res, err := client.Payments.AuthorizeNonce(context.Background(), integrationPaymentID, integrationPayment.Payer)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes32RE.MatchString(res.Nonce) {
		t.Errorf("nonce format invalid: %s", res.Nonce)
	}
}

func TestIntegration_PaymentsChargeNonce(t *testing.T) {
	client := newTestServer(t)
	res, err := client.Payments.ChargeNonce(context.Background(), integrationPaymentID, integrationPayment.Payer)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes32RE.MatchString(res.Nonce) {
		t.Errorf("nonce format invalid: %s", res.Nonce)
	}
}

func TestIntegration_PaymentsHash(t *testing.T) {
	client := newTestServer(t)
	res, err := client.Payments.Hash(context.Background(), integrationPayment)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes32RE.MatchString(res.Hash) {
		t.Errorf("hash format invalid: %s", res.Hash)
	}
}

// ================================================================
//  Tokens
// ================================================================

func TestIntegration_TokensIsAccepted(t *testing.T) {
	client := newTestServer(t)
	token := "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
	res, err := client.Tokens.IsAccepted(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	// The mock server always returns true for any token address.
	_ = res.Accepted
	_ = res.Address
}

// ================================================================
//  Utils
// ================================================================

func TestIntegration_UtilsDomainSeparator(t *testing.T) {
	client := newTestServer(t)
	res, err := client.Utils.DomainSeparator(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes32RE.MatchString(res.DomainSeparator) {
		t.Errorf("domainSeparator format invalid: %s", res.DomainSeparator)
	}
}

func TestIntegration_UtilsVersion(t *testing.T) {
	client := newTestServer(t)
	res, err := client.Utils.Version(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.Version <= 0 {
		t.Errorf("version must be positive, got %d", res.Version)
	}
}
