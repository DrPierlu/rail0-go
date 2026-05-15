// Package rail0 provides a Go client for the RAIL0 stablecoin payment API.
//
// Quick start:
//
//	client := rail0.NewClient(rail0.ClientOptions{BaseURL: "https://api.rail0.xyz"})
//	state, err := client.Payments.Get(ctx, paymentID)
package rail0

// Client is the entry point for the RAIL0 SDK.
type Client struct {
	// Payments exposes the full payment lifecycle: Authorize, Charge, Capture, Void, Release, Refund.
	Payments *PaymentsService
	// Tokens exposes token allowlist queries.
	Tokens *TokensService
	// Utils exposes contract introspection: DomainSeparator, Version.
	Utils *UtilsService
}

// NewClient creates a new Client with the provided options.
func NewClient(opts ClientOptions) *Client {
	h := newHTTPClient(opts)
	return &Client{
		Payments: &PaymentsService{http: h},
		Tokens:   &TokensService{http: h},
		Utils:    &UtilsService{http: h},
	}
}
