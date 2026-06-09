package rail0

import (
	"context"
	"fmt"
	"net/url"
)

// AccountsService exposes account wallet management operations.
type AccountsService struct {
	http *httpClient
}

// ListWalletsParams holds optional query parameters for Wallets.
type ListWalletsParams struct {
	Active  *bool // filter by active flag; nil = no filter
	Page    int   // page number (1-based); 0 = use server default
	PerPage int   // items per page; 0 = use server default
}

// Wallets lists the wallets for the given account.
func (s *AccountsService) Wallets(ctx context.Context, accountID string, params ...ListWalletsParams) ([]Wallet, error) {
	var out []Wallet
	path := fmt.Sprintf("/accounts/%s/wallets", accountID)
	if len(params) > 0 {
		p := params[0]
		q := url.Values{}
		if p.Active != nil {
			if *p.Active {
				q.Set("active", "true")
			} else {
				q.Set("active", "false")
			}
		}
		if p.Page > 0 {
			q.Set("page", fmt.Sprintf("%d", p.Page))
		}
		if p.PerPage > 0 {
			q.Set("per_page", fmt.Sprintf("%d", p.PerPage))
		}
		if len(q) > 0 {
			path = path + "?" + q.Encode()
		}
	}
	if err := s.http.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetWallet fetches a single wallet by ID scoped to the account.
func (s *AccountsService) GetWallet(ctx context.Context, accountID, walletID string) (*Wallet, error) {
	var out Wallet
	if err := s.http.get(ctx, fmt.Sprintf("/accounts/%s/wallets/%s", accountID, walletID), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateWalletRequest is the body for adding a wallet to an account.
type CreateWalletRequest struct {
	Address string `json:"address"`
	Label   string `json:"label,omitempty"`
}

// CreateWallet adds a new EVM wallet address to the account.
func (s *AccountsService) CreateWallet(ctx context.Context, accountID string, req CreateWalletRequest) (*Wallet, error) {
	var out Wallet
	if err := s.http.post(ctx, fmt.Sprintf("/accounts/%s/wallets", accountID), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateWalletRequest holds the fields that can be patched on a wallet.
type UpdateWalletRequest struct {
	Label  string `json:"label,omitempty"`
	Active *bool  `json:"active,omitempty"`
}

// UpdateWallet patches label or active status on a wallet.
func (s *AccountsService) UpdateWallet(ctx context.Context, accountID, walletID string, req UpdateWalletRequest) (*Wallet, error) {
	var out Wallet
	if err := s.http.patch(ctx, fmt.Sprintf("/accounts/%s/wallets/%s", accountID, walletID), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteWallet deactivates (soft-deletes) a wallet.
func (s *AccountsService) DeleteWallet(ctx context.Context, accountID, walletID string) error {
	return s.http.delete(ctx, fmt.Sprintf("/accounts/%s/wallets/%s", accountID, walletID), nil)
}
