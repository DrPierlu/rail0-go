// Package rail0 provides a Go client for the RAIL0 stablecoin payment API.
//
// Quick start:
//
//	client := rail0.NewClient(rail0.ClientOptions{BaseURL: "https://api.rail0.xyz"})
//	resp, err := client.Payments.CreatePayment(ctx, rail0.CreatePaymentRequest{...})
package rail0

// Client is the entry point for the RAIL0 SDK.
type Client struct {
	// Payments exposes the full payment lifecycle: CreatePayment, Sign, Authorize, Charge,
	// PrepareCapture, SubmitCapture, PrepareVoid, SubmitVoid, Release,
	// PrepareApprove, SubmitApprove, PrepareRefund, SubmitRefund.
	Payments *PaymentsService
}

// NewClient creates a new Client with the provided options.
func NewClient(opts ClientOptions) *Client {
	h := newHTTPClient(opts)
	return &Client{
		Payments: &PaymentsService{http: h},
	}
}
