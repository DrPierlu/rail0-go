package rail0

import (
	"context"
	"fmt"
	"net/url"
)

// AccountsService exposes account configuration operations.
type AccountsService struct {
	http *httpClient
}

// PaymentMethodsFilters holds optional filters for PaymentMethods.
type PaymentMethodsFilters struct {
	StablecoinID     string // filter by token id
	StablecoinSymbol string // filter by token symbol (case-insensitive)
	BlockchainID     string // filter by blockchain id
	BlockchainSlug   string // filter by blockchain slug (case-insensitive)
}

// PaymentMethods returns the active payment methods (chain + token + wallet) for the given account.
// All non-empty filter fields are applied as AND conditions; discordant filters return an empty slice.
func (s *AccountsService) PaymentMethods(ctx context.Context, accountID string, filters ...PaymentMethodsFilters) ([]PaymentMethod, error) {
	var out []PaymentMethod
	path := fmt.Sprintf("/accounts/%s/payment-methods", accountID)

	if len(filters) > 0 {
		f := filters[0]
		q := url.Values{}
		if f.StablecoinID != "" {
			q.Set("stablecoin_id", f.StablecoinID)
		}
		if f.StablecoinSymbol != "" {
			q.Set("stablecoin_symbol", f.StablecoinSymbol)
		}
		if f.BlockchainID != "" {
			q.Set("blockchain_id", f.BlockchainID)
		}
		if f.BlockchainSlug != "" {
			q.Set("blockchain_slug", f.BlockchainSlug)
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
