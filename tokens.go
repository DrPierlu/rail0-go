package rail0

import "context"

// TokensService exposes token allowlist queries.
type TokensService struct {
	http *httpClient
}

// IsAccepted returns whether the given ERC-20 token is in this deployment's allowlist.
func (s *TokensService) IsAccepted(ctx context.Context, address Address) (*TokenStatusResponse, error) {
	var out TokenStatusResponse
	if err := s.http.get(ctx, "/tokens/"+address, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
