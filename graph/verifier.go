package graph

import (
	"net/http"

	"github.com/39alpha/dorothy/core"
	"github.com/go-chi/oauth"
)

type UserVerifier struct {
	oauth.CredentialsVerifier
	db *core.DatabaseSession
}

func (v *UserVerifier) ValidateUser(username, password, scope string, r *http.Request) error {
	return core.ValidateCredentials(v.db, username, password)
}

func (*UserVerifier) ValidateClient(clientID, clientSecret, scope string, r *http.Request) error {
	return nil
}

func (*UserVerifier) AddClaims(tokenType oauth.TokenType, credential, tokenID, scope string, r *http.Request) (map[string]string, error) {
	return nil, nil
}

func (*UserVerifier) StoreTokenID(tokenType oauth.TokenType, credential, tokenID, refreshTokenID string) error {
	return nil
}

func (*UserVerifier) AddProperties(tokenType oauth.TokenType, credential, tokenID, scope string, r *http.Request) (map[string]string, error) {
	return nil, nil
}
