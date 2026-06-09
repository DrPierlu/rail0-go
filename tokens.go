package rail0

import (
	"context"
	"strconv"
)

// TokensService exposes the supported token catalog.
type TokensService struct {
	http *httpClient
}

// List returns all active catalog tokens. Pass chainID > 0 to filter by chain.
// Returns the flat CatalogToken shape (chain_id, chain_slug, symbol, address, decimals).
func (s *TokensService) List(ctx context.Context, chainID int) ([]CatalogToken, error) {
	var out []CatalogToken
	path := "/tokens"
	if chainID > 0 {
		path += "?chain_id=" + strconv.Itoa(chainID)
	}
	if err := s.http.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return out, nil
}
