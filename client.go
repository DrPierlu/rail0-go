// Package rail0 provides a Go client for the RAIL0 stablecoin payment API.
//
// Quick start:
//
//	client := rail0.NewClient(rail0.ClientOptions{BaseURL: "https://api.rail0.xyz"})
//	resp, err := client.Payments.CreatePayment(ctx, rail0.CreatePaymentRequest{...})
package rail0

// Client is the entry point for the RAIL0 SDK.
type Client struct {
	// Accounts exposes wallet CRUD under /accounts/:id/wallets.
	Accounts *AccountsService
	// Wallets exposes GET /wallets/:id/tokens.
	Wallets *WalletsService
	// Auth exposes SIWE authentication: GetNonce, Verify.
	Auth *AuthService
	// Chains exposes the supported blockchain catalog.
	Chains *ChainsService
	// Tokens exposes the flat token catalog (GET /tokens).
	Tokens *TokensService
	// Payments exposes the full payment lifecycle.
	Payments *PaymentsService
}

// NewClient creates a new Client with the provided options.
func NewClient(opts ClientOptions) *Client {
	h := newHTTPClient(opts)
	return &Client{
		Accounts: &AccountsService{http: h},
		Wallets:  &WalletsService{http: h},
		Auth:     &AuthService{http: h},
		Chains:   &ChainsService{http: h},
		Tokens:   &TokensService{http: h},
		Payments: &PaymentsService{http: h},
	}
}
