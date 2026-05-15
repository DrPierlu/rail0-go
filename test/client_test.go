// Package rail0_test contains unit tests for the RAIL0 Go SDK.
//
// Tests in this file cover the HTTP client layer: correct URL routing, request
// body serialisation, error decoding, retry logic, and logger callbacks.
// All network I/O is intercepted via a mock http.RoundTripper — no real server
// is started (equivalent to vi.spyOn(globalThis, 'fetch') in the TypeScript tests).
package rail0_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	rail0 "github.com/rail0/go-sdk"
)

// ================================================================
//  Mock transport — replaces the fetch spy used in the TS tests
// ================================================================

type mockCall struct {
	resp *http.Response
	err  error
}

type mockTransport struct {
	t        *testing.T
	queue    []mockCall
	i        int
	recorded []*http.Request
}

func (m *mockTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	// Clone request before body is consumed so tests can inspect it later.
	clone := r.Clone(context.Background())
	if r.Body != nil {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r.Body)
		clone.Body = io.NopCloser(&buf)
	}
	m.recorded = append(m.recorded, clone)

	if m.i >= len(m.queue) {
		m.t.Fatalf("unexpected request #%d: %s %s", m.i+1, r.Method, r.URL)
	}
	c := m.queue[m.i]
	m.i++
	return c.resp, c.err
}

func (m *mockTransport) push(resp *http.Response) { m.queue = append(m.queue, mockCall{resp: resp}) }
func (m *mockTransport) fail(err error)            { m.queue = append(m.queue, mockCall{err: err}) }

func jsonResp(status int, body any) *http.Response {
	b, _ := json.Marshal(body)
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(b)),
	}
}

// newMockClient wires a fresh Client with the mock transport and zero retry delay.
func newMockClient(t *testing.T, mt *mockTransport, extra ...func(*rail0.ClientOptions)) *rail0.Client {
	t.Helper()
	opts := rail0.ClientOptions{
		BaseURL:   "http://test.invalid",
		Transport: mt,
	}
	for _, f := range extra {
		f(&opts)
	}
	return rail0.NewClient(opts)
}

// ================================================================
//  Shared fixtures
// ================================================================

const mockPaymentID = "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"

var mockPayment = rail0.Payment{
	Payer:               "0x1111111111111111111111111111111111111111",
	Payee:               "0x2222222222222222222222222222222222222222",
	Token:               "0x3333333333333333333333333333333333333333",
	MaxAmount:           "1000000",
	AuthorizationExpiry: 9999999999,
	RefundExpiry:        9999999999,
	FeeBps:              0,
	FeeReceiver:         "0x0000000000000000000000000000000000000000",
}

var mockSig = struct{ V int; R, S rail0.Bytes32 }{
	V: 27,
	R: "0x1111111111111111111111111111111111111111111111111111111111111111",
	S: "0x2222222222222222222222222222222222222222222222222222222222222222",
}

var mockTx = map[string]any{
	"transactionHash": "0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	"status":          "pending",
}

var mockPaymentResp = map[string]any{
	"paymentId": mockPaymentID,
	"state": map[string]any{
		"exists":           true,
		"capturableAmount": "1000000",
		"refundableAmount": "0",
	},
	"configHash": "0xabababababababababababababababababababababababababababababababababab",
}

// ================================================================
//  Payments.Get
// ================================================================

func TestGet_ReturnsPaymentResponse(t *testing.T) {
	mt := &mockTransport{t: t}
	mt.push(jsonResp(200, mockPaymentResp))
	client := newMockClient(t, mt)

	res, err := client.Payments.Get(context.Background(), mockPaymentID)
	if err != nil {
		t.Fatal(err)
	}

	if res.PaymentID != mockPaymentID {
		t.Errorf("PaymentID: got %s, want %s", res.PaymentID, mockPaymentID)
	}
	if !res.State.Exists {
		t.Error("State.Exists should be true")
	}
	if res.State.CapturableAmount != "1000000" {
		t.Errorf("CapturableAmount: got %s, want 1000000", res.State.CapturableAmount)
	}
}

func TestGet_404_ReturnsAPIError(t *testing.T) {
	mt := &mockTransport{t: t}
	mt.push(jsonResp(404, map[string]any{"error": "PaymentNotFound", "message": "No payment found."}))
	client := newMockClient(t, mt)

	_, err := client.Payments.Get(context.Background(), mockPaymentID)

	var apiErr *rail0.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.Status != 404 {
		t.Errorf("Status: got %d, want 404", apiErr.Status)
	}
	if apiErr.Code != "PaymentNotFound" {
		t.Errorf("Code: got %s, want PaymentNotFound", apiErr.Code)
	}
}

// ================================================================
//  Payments.Authorize — URL + body routing
// ================================================================

func TestAuthorize_PostsToCorrectURL(t *testing.T) {
	mt := &mockTransport{t: t}
	mt.push(jsonResp(202, mockTx))
	client := newMockClient(t, mt)

	_, err := client.Payments.Authorize(context.Background(), mockPaymentID, rail0.AuthorizeParams{
		Payment: mockPayment,
		Amount:  "1000000",
		V:       mockSig.V,
		R:       mockSig.R,
		S:       mockSig.S,
	})
	if err != nil {
		t.Fatal(err)
	}

	req := mt.recorded[0]
	if req.Method != http.MethodPost {
		t.Errorf("method: got %s, want POST", req.Method)
	}
	wantPath := "/payments/" + mockPaymentID + "/authorize"
	if req.URL.Path != wantPath {
		t.Errorf("path: got %s, want %s", req.URL.Path, wantPath)
	}
}

func TestAuthorize_SendsJSONBody(t *testing.T) {
	mt := &mockTransport{t: t}
	mt.push(jsonResp(202, mockTx))
	client := newMockClient(t, mt)

	params := rail0.AuthorizeParams{
		Payment: mockPayment,
		Amount:  "1000000",
		V:       mockSig.V,
		R:       mockSig.R,
		S:       mockSig.S,
	}
	if _, err := client.Payments.Authorize(context.Background(), mockPaymentID, params); err != nil {
		t.Fatal(err)
	}

	body, _ := io.ReadAll(mt.recorded[0].Body)
	var decoded rail0.AuthorizeParams
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
	if decoded.V != 27 {
		t.Errorf("v: got %d, want 27", decoded.V)
	}
	if decoded.Amount != "1000000" {
		t.Errorf("amount: got %s, want 1000000", decoded.Amount)
	}
}

// ================================================================
//  Retry logic
// ================================================================

func TestRetry_SucceedsOnThirdAttempt(t *testing.T) {
	mt := &mockTransport{t: t}
	mt.fail(errors.New("network failure"))
	mt.fail(errors.New("network failure"))
	mt.push(jsonResp(200, mockPaymentResp))

	client := newMockClient(t, mt, func(o *rail0.ClientOptions) {
		o.MaxRetries = 2
		o.RetryDelay = time.Nanosecond
	})

	res, err := client.Payments.Get(context.Background(), mockPaymentID)
	if err != nil {
		t.Fatalf("expected success after 2 retries, got: %v", err)
	}
	if res.PaymentID != mockPaymentID {
		t.Errorf("PaymentID: got %s", res.PaymentID)
	}
	if mt.i != 3 {
		t.Errorf("expected 3 total attempts, got %d", mt.i)
	}
}

func TestRetry_ThrowsAfterExhausted(t *testing.T) {
	networkErr := errors.New("network failure")
	mt := &mockTransport{t: t}
	mt.fail(networkErr)
	mt.fail(networkErr)
	mt.fail(networkErr)

	client := newMockClient(t, mt, func(o *rail0.ClientOptions) {
		o.MaxRetries = 2
		o.RetryDelay = time.Nanosecond
	})

	_, err := client.Payments.Get(context.Background(), mockPaymentID)
	if !errors.Is(err, networkErr) {
		t.Errorf("expected networkErr, got %v", err)
	}
	if mt.i != 3 {
		t.Errorf("expected 3 total attempts, got %d", mt.i)
	}
}

func TestRetry_DoesNotRetryHTTPErrors(t *testing.T) {
	mt := &mockTransport{t: t}
	mt.push(jsonResp(404, map[string]any{"error": "PaymentNotFound", "message": "Not found."}))

	client := newMockClient(t, mt, func(o *rail0.ClientOptions) {
		o.MaxRetries = 2
		o.RetryDelay = time.Nanosecond
	})

	_, err := client.Payments.Get(context.Background(), mockPaymentID)
	var apiErr *rail0.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if mt.i != 1 {
		t.Errorf("HTTP errors must not be retried; got %d attempts", mt.i)
	}
}

// ================================================================
//  Logger callbacks
// ================================================================

func TestLogger_SuccessEntry(t *testing.T) {
	mt := &mockTransport{t: t}
	mt.push(jsonResp(200, mockPaymentResp))

	var entries []rail0.LogEntry
	client := newMockClient(t, mt, func(o *rail0.ClientOptions) {
		o.Logger = func(e rail0.LogEntry) { entries = append(entries, e) }
	})

	if _, err := client.Payments.Get(context.Background(), mockPaymentID); err != nil {
		t.Fatal(err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Method != "GET" {
		t.Errorf("Method: got %s", e.Method)
	}
	if !strings.Contains(e.URL, "/payments/"+mockPaymentID) {
		t.Errorf("URL missing payment path: %s", e.URL)
	}
	if e.Status != 200 {
		t.Errorf("Status: got %d", e.Status)
	}
	if e.DurationMs < 0 {
		t.Error("DurationMs must be non-negative")
	}
	if e.Err != nil {
		t.Errorf("Err should be nil, got %v", e.Err)
	}
	if e.RequestBody != nil {
		t.Error("RequestBody should be nil for GET")
	}
}

func TestLogger_IncludesRequestBodyOnPOST(t *testing.T) {
	mt := &mockTransport{t: t}
	mt.push(jsonResp(202, mockTx))

	var entries []rail0.LogEntry
	client := newMockClient(t, mt, func(o *rail0.ClientOptions) {
		o.Logger = func(e rail0.LogEntry) { entries = append(entries, e) }
	})

	params := rail0.AuthorizeParams{Payment: mockPayment, Amount: "1000000", V: 27, R: mockSig.R, S: mockSig.S}
	if _, err := client.Payments.Authorize(context.Background(), mockPaymentID, params); err != nil {
		t.Fatal(err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Method != "POST" {
		t.Errorf("Method: got %s", e.Method)
	}
	if e.Status != 202 {
		t.Errorf("Status: got %d", e.Status)
	}
	if e.RequestBody == nil {
		t.Error("RequestBody should be non-nil for POST")
	}
	if e.Err != nil {
		t.Errorf("Err should be nil, got %v", e.Err)
	}
}

func TestLogger_HTTPError(t *testing.T) {
	mt := &mockTransport{t: t}
	errBody := map[string]any{"error": "PaymentNotFound", "message": "No payment found."}
	mt.push(jsonResp(404, errBody))

	var entries []rail0.LogEntry
	client := newMockClient(t, mt, func(o *rail0.ClientOptions) {
		o.Logger = func(e rail0.LogEntry) { entries = append(entries, e) }
	})

	_, _ = client.Payments.Get(context.Background(), mockPaymentID)

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Status != 404 {
		t.Errorf("Status: got %d", e.Status)
	}
	var apiErr *rail0.APIError
	if !errors.As(e.Err, &apiErr) {
		t.Errorf("Err should be *APIError, got %T", e.Err)
	}
}

func TestLogger_NetworkError(t *testing.T) {
	networkErr := errors.New("network failure")
	mt := &mockTransport{t: t}
	mt.fail(networkErr)

	var entries []rail0.LogEntry
	client := newMockClient(t, mt, func(o *rail0.ClientOptions) {
		o.Logger = func(e rail0.LogEntry) { entries = append(entries, e) }
	})

	_, _ = client.Payments.Get(context.Background(), mockPaymentID)

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Status != 0 {
		t.Errorf("Status should be 0 on network error, got %d", e.Status)
	}
	if !errors.Is(e.Err, networkErr) {
		t.Errorf("Err: expected networkErr, got %v", e.Err)
	}
	if e.ResponseBody != nil {
		t.Error("ResponseBody should be nil on network error")
	}
}

func TestLogger_AttemptAndWillRetryFields(t *testing.T) {
	networkErr := errors.New("network failure")
	mt := &mockTransport{t: t}
	mt.fail(networkErr)
	mt.push(jsonResp(200, mockPaymentResp))

	var entries []rail0.LogEntry
	client := newMockClient(t, mt, func(o *rail0.ClientOptions) {
		o.MaxRetries = 1
		o.RetryDelay = time.Nanosecond
		o.Logger = func(e rail0.LogEntry) { entries = append(entries, e) }
	})

	if _, err := client.Payments.Get(context.Background(), mockPaymentID); err != nil {
		t.Fatal(err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 log entries, got %d", len(entries))
	}
	fail := entries[0]
	if fail.Attempt != 1 {
		t.Errorf("fail.Attempt: got %d, want 1", fail.Attempt)
	}
	if !fail.WillRetry {
		t.Error("fail.WillRetry should be true")
	}
	if !errors.Is(fail.Err, networkErr) {
		t.Errorf("fail.Err: expected networkErr")
	}

	success := entries[1]
	if success.Attempt != 2 {
		t.Errorf("success.Attempt: got %d, want 2", success.Attempt)
	}
	if success.WillRetry {
		t.Error("success.WillRetry should be false")
	}
	if success.Err != nil {
		t.Errorf("success.Err should be nil, got %v", success.Err)
	}
}

// ================================================================
//  APIError
// ================================================================

func TestAPIError_ExposesStatusAndCode(t *testing.T) {
	err := &rail0.APIError{Status: 422, Code: "InvalidAmount", Message: "Amount is zero."}

	if err.Status != 422 {
		t.Errorf("Status: got %d", err.Status)
	}
	if err.Code != "InvalidAmount" {
		t.Errorf("Code: got %s", err.Code)
	}
	if err.Message != "Amount is zero." {
		t.Errorf("Message: got %s", err.Message)
	}
	if !strings.Contains(err.Error(), "422") {
		t.Errorf("Error() should contain status code: %s", err.Error())
	}
}

// ================================================================
//  DebugLogger smoke test
// ================================================================

func TestDebugLogger_DoesNotPanic(t *testing.T) {
	// Just ensure DebugLogger runs without panicking on a normal entry.
	rail0.DebugLogger(rail0.LogEntry{
		Method:     "GET",
		URL:        "http://test.invalid/payments/0xabc",
		Status:     200,
		DurationMs: 42,
	})
}

// Confirm that a tiny RetryDelay makes retries complete without significant latency.
func TestRetry_SmallDelayIsRespected(t *testing.T) {
	networkErr := errors.New("net")
	mt := &mockTransport{t: t}
	mt.fail(networkErr)
	mt.fail(networkErr)
	mt.push(jsonResp(200, mockPaymentResp))

	client := newMockClient(t, mt, func(o *rail0.ClientOptions) {
		o.MaxRetries = 2
		o.RetryDelay = time.Nanosecond
	})

	start := time.Now()
	if _, err := client.Payments.Get(context.Background(), mockPaymentID); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Errorf("zero RetryDelay took too long: %s", elapsed)
	}
}
