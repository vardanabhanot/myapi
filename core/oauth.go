package core

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ponytail: client_credentials grant only, credentials sent as a Basic auth
// header (the RFC 6749 MUST-support method); add body-credentials or the
// password grant when a real server needs them.

type oauthEntry struct {
	token  string
	expiry time.Time
}

var (
	oauthMu    sync.Mutex
	oauthCache = map[string]oauthEntry{}
)

// oauthToken fetches (or reuses a cached) access token for the request's
// OAuth2 settings. Tokens are cached in memory per tokenURL+clientID+scope
// until shortly before expiry, so repeated sends don't round-trip to the
// token endpoint.
func oauthToken(ctx context.Context, a *Auth, skipTLS bool) (string, error) {
	tokenURL := ApplyEnv(a.OAuthTokenURL)
	clientID := ApplyEnv(a.OAuthClientID)
	secret := ApplyEnv(a.OAuthClientSecret)
	scope := ApplyEnv(a.OAuthScope)

	key := tokenURL + "\x00" + clientID + "\x00" + scope

	oauthMu.Lock()
	if e, ok := oauthCache[key]; ok && time.Now().Before(e.expiry) {
		oauthMu.Unlock()
		return e.token, nil
	}
	oauthMu.Unlock()

	form := url.Values{"grant_type": {"client_credentials"}}
	if scope != "" {
		form.Set("scope", scope)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, secret)

	client := &http.Client{Timeout: 30 * time.Second}
	if skipTLS {
		client.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("oauth2 token request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("oauth2 token endpoint returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var tr struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", fmt.Errorf("oauth2 token response: %w", err)
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("oauth2 token response has no access_token")
	}

	// Only cache tokens that outlive a 30s safety margin; expires_in is
	// optional in the RFC, absent means we re-fetch every send.
	if tr.ExpiresIn > 30 {
		oauthMu.Lock()
		oauthCache[key] = oauthEntry{tr.AccessToken, time.Now().Add(time.Duration(tr.ExpiresIn-30) * time.Second)}
		oauthMu.Unlock()
	}

	return tr.AccessToken, nil
}
