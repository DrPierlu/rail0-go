package rail0

import (
	"context"
	"fmt"
)

// MerchantsService implements merchant configuration operations.
type MerchantsService struct {
	http *httpClient
}

// PaymentMethods returns the active payment methods (chain + token + wallet) for the given merchant.
func (s *MerchantsService) PaymentMethods(ctx context.Context, merchantID int) ([]PaymentMethod, error) {
	var out []PaymentMethod
	path := fmt.Sprintf("/merchants/%d/payment-methods", merchantID)
	if err := s.http.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return out, nil
}
