package rail0

import "context"

// UtilsService exposes contract introspection endpoints.
type UtilsService struct {
	http *httpClient
}

// DomainSeparator returns the EIP-712 domain separator for the RAIL0 contract on the current chain.
func (s *UtilsService) DomainSeparator(ctx context.Context) (*DomainSeparatorResponse, error) {
	var out DomainSeparatorResponse
	if err := s.http.get(ctx, "/domain-separator", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Version returns the contract version number.
func (s *UtilsService) Version(ctx context.Context) (*VersionResponse, error) {
	var out VersionResponse
	if err := s.http.get(ctx, "/version", &out); err != nil {
		return nil, err
	}
	return &out, nil
}
