package rail0

import "context"

// AuthService handles SIWE authentication.
type AuthService struct {
	http *httpClient
}

// NonceResponse is returned by GetNonce.
type NonceResponse struct {
	Nonce     string `json:"nonce"`
	ExpiresAt string `json:"expires_at"`
}

// AuthRequest is the body for Verify.
type AuthRequest struct {
	Message   string `json:"message"`
	Signature string `json:"signature"`
}

// AuthResponse is returned by Verify on success.
type AuthResponse struct {
	Token     string `json:"token"`
	Address   string `json:"address"`
	AccountID string `json:"account_id"`
	ExpiresAt string `json:"expires_at"`
}

// GetNonce requests a single-use SIWE nonce from the API.
func (s *AuthService) GetNonce(ctx context.Context) (*NonceResponse, error) {
	var out NonceResponse
	if err := s.http.get(ctx, "/auth/nonce", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Verify submits a signed SIWE message and returns a JWT on success.
func (s *AuthService) Verify(ctx context.Context, req AuthRequest) (*AuthResponse, error) {
	var out AuthResponse
	if err := s.http.post(ctx, "/auth", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
