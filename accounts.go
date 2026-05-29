package rail0

import (
	"context"
	"fmt"
)

// AccountsService exposes account configuration operations.
type AccountsService struct {
	http *httpClient
}

// PaymentMethods returns the active payment methods (chain + token + wallet) for the given account.
func (s *AccountsService) PaymentMethods(ctx context.Context, accountID string) ([]PaymentMethod, error) {
	var out []PaymentMethod
	path := fmt.Sprintf("/accounts/%s/payment-methods", accountID)
	if err := s.http.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return out, nil
}
