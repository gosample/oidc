package oidc

import (
	"context"
	"errors"
	"io/ioutil"
	"log"
	"net/url"
	"strings"
	"sync"
)

//go:generate mockery -name TokenSource -case underscore

// TokenSource is anything that can return an oidc token and verifier for token verification.
type TokenSource interface {
	// OIDCToken must be safe for concurrent use by multiple goroutines.
	// The returned Token must not be modified.
	OIDCToken() (*Token, error)
	Verifier() Verifier
}

// ReuseTokenSource is a oidc TokenSource that holds a single token in memory
// and validates its expiry before each call to retrieve it with
// Token. If it's expired, it will be auto-refreshed using the
// new TokenSource.
type ReuseTokenSource struct {
	ctx context.Context // ctx for HTTP requests.

	new TokenSource // called when t is expired.
	mu  sync.Mutex  // guards t
	t   *Token

	// Optional std logger for debug log. The only case which will be logged is why OIDC token was invalid.
	debugLogger *log.Logger
}

// NewReuseTokenSource returns a TokenSource which repeatedly returns the
// same token as long as it's valid, starting with t.
// As a second argument it returns reset function that enables to reset h
// When its cached token is invalid, a new token is obtained from source.
func NewReuseTokenSource(ctx context.Context, t *Token, src TokenSource) (ret TokenSource, clearIDToken func()) {
	s := &ReuseTokenSource{
		ctx:         ctx,
		t:           t,
		new:         src,
		debugLogger: log.New(ioutil.Discard, "", 0),
	}
	return s, s.reset
}

// NewReuseTokenSourceWithDebugLogger is the same as NewReuseTokenSource but with logger.
func NewReuseTokenSourceWithDebugLogger(ctx context.Context, debugLogger *log.Logger, t *Token, src TokenSource) (ret TokenSource, clearIDToken func()) {
	s := &ReuseTokenSource{
		ctx:         ctx,
		t:           t,
		new:         src,
		debugLogger: debugLogger,
	}
	return s, s.reset
}

// OIDCToken returns the current token if it's still valid, else will
// refresh the current token (using r.Context for HTTP client
// information) and return the new one.
func (s *ReuseTokenSource) OIDCToken() (*Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.t != nil {
		err := s.t.IsValid(s.ctx, s.Verifier())
		if err == nil {
			return s.t, nil
		}
		s.debugLogger.Printf("reuseTokenSource: Token not valid. Obtaining new one. Cause: %v\n", err)
	} else {
		s.debugLogger.Println("reuseTokenSource: No token to reuse. Obtaining new one")
	}
	t, err := s.new.OIDCToken()
	if err != nil {
		return nil, err
	}
	s.t = t
	return t, nil
}

// Verifier returns verifier from underlying token source.
func (s *ReuseTokenSource) Verifier() Verifier {
	return s.new.Verifier()
}

func (s *ReuseTokenSource) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.t = nil
}

// TokenRefresher is a TokenSource that makes "grant_type"=="refresh_token"
// HTTP requests to renew a token using a RefreshToken.
type TokenRefresher struct {
	ctx context.Context // used to get HTTP requests

	refreshToken string
	client       *Client

	cfg Config
}

// NewTokenRefresher constructs token refresher.
func NewTokenRefresher(ctx context.Context, client *Client, cfg Config, refreshToken string) TokenSource {
	return &TokenRefresher{
		ctx:          ctx,
		refreshToken: refreshToken,
		client:       client,
		cfg:          cfg,
	}
}

// OIDCToken is not safe for concurrent access, as it
// updates the tokenRefresher's refreshToken field.
// It is meant to be used with ReuseTokenSource which
// synchronizes calls to this method with its own mutex.
// NOTE: Returned token is not verified.
func (tf *TokenRefresher) OIDCToken() (*Token, error) {
	if tf.refreshToken == "" {
		return nil, errors.New("oauth2: token expired and refresh token is not set")
	}

	v := url.Values{
		"grant_type":    {GrantTypeRefreshToken},
		"refresh_token": {tf.refreshToken},
	}

	if len(tf.cfg.Scopes) > 0 {
		v.Set("scope", strings.Join(tf.cfg.Scopes, " "))
	}

	tk, err := tf.client.token(tf.ctx, tf.cfg.ClientID, tf.cfg.ClientSecret, v)
	if err != nil {
		return nil, err
	}

	if tf.refreshToken != tk.RefreshToken {
		tf.refreshToken = tk.RefreshToken
	}

	return tk, err
}

// Verifier returns verifier for ID Token.
func (tf *TokenRefresher) Verifier() Verifier {
	return tf.client.Verifier(VerificationConfig{ClientID: tf.cfg.ClientID})
}

// StaticTokenSource returns a TokenSource that always returns the same token.
// Because the provided token t is never refreshed, StaticTokenSource is only
// useful for tokens that never expire.
func StaticTokenSource(t *Token) TokenSource {
	return staticTokenSource{t}
}

// staticTokenSource is a TokenSource that always returns the same Token.
type staticTokenSource struct {
	t *Token
}

// OIDCToken returns saved pointer to token.
func (s staticTokenSource) OIDCToken() (*Token, error) {
	return s.t, nil
}

// Verifier returns nil, since it is static.
func (s staticTokenSource) Verifier() Verifier {
	return nil
}
