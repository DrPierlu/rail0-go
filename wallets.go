package rail0

import (
	"context"
	"fmt"
	"net/url"
)

// WalletsService exposes wallet token queries.
type WalletsService struct {
	http *httpClient
}

// ListWalletTokensParams holds optional query parameters for wallet tokens.
type ListWalletTokensParams struct {
	Symbol  string // filter by token symbol (case-insensitive)
	Active  *bool  // filter by token active flag; nil = no filter
	Page    int    // page number (1-based); 0 = use server default
	PerPage int    // items per page; 0 = use server default
}

// Tokens lists the tokens associated with a wallet.
// Each token includes a nested Blockchain object.
func (s *WalletsService) Tokens(ctx context.Context, walletID string, params ...ListWalletTokensParams) ([]Token, error) {
	var out []Token
	path := fmt.Sprintf("/wallets/%s/tokens", walletID)
	if len(params) > 0 {
		p := params[0]
		q := url.Values{}
		if p.Symbol != "" {
			q.Set("symbol", p.Symbol)
		}
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
