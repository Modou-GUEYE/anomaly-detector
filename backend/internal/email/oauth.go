package email

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ===================== OAUTH 2.0 (XOAUTH2) =====================

const (
	defaultOAuthScope    = "https://outlook.office365.com/.default"
	defaultOAuthTokenURL = "https://login.microsoftonline.com/%s/oauth2/v2.0/token"
	// tokenRefreshMargin renouvelle le jeton avant son expiration réelle.
	tokenRefreshMargin = 5 * time.Minute
)

// OAuthConfig décrit une application Entra ID (Azure AD) autorisée à envoyer via SMTP
// (permission SMTP.SendAsApp, flux client_credentials).
type OAuthConfig struct {
	TenantID     string
	ClientID     string
	ClientSecret string
	Scope        string // https://outlook.office365.com/.default par défaut
	TokenURL     string // point de terminaison Microsoft par défaut (surchargé pour les tests)
}

// Enabled indique si l'authentification OAuth est configurée.
func (c OAuthConfig) Enabled() bool {
	return c.TenantID != "" && c.ClientID != "" && c.ClientSecret != ""
}

type cachedToken struct {
	value   string
	expires time.Time
}

var (
	tokenMu    sync.Mutex
	tokenCache = map[string]cachedToken{}
)

// accessToken retourne un jeton valide, depuis le cache ou en le demandant au fournisseur.
func (c OAuthConfig) accessToken(timeout time.Duration) (string, error) {
	key := c.TenantID + "|" + c.ClientID + "|" + c.scope()

	tokenMu.Lock()
	defer tokenMu.Unlock()
	if t, ok := tokenCache[key]; ok && time.Now().Before(t.expires) {
		return t.value, nil
	}

	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {c.ClientID},
		"client_secret": {c.ClientSecret},
		"scope":         {c.scope()},
	}
	httpClient := &http.Client{Timeout: timeout}
	resp, err := httpClient.Post(c.tokenURL(), "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	var payload struct {
		AccessToken      string `json:"access_token"`
		ExpiresIn        int    `json:"expires_in"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("réponse du fournisseur OAuth illisible (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK || payload.AccessToken == "" {
		return "", fmt.Errorf("HTTP %d: %s %s", resp.StatusCode, payload.Error, payload.ErrorDescription)
	}

	lifetime := time.Duration(payload.ExpiresIn) * time.Second
	if lifetime > tokenRefreshMargin {
		lifetime -= tokenRefreshMargin
	}
	tokenCache[key] = cachedToken{value: payload.AccessToken, expires: time.Now().Add(lifetime)}
	return payload.AccessToken, nil
}

func (c OAuthConfig) scope() string {
	if c.Scope != "" {
		return c.Scope
	}
	return defaultOAuthScope
}

func (c OAuthConfig) tokenURL() string {
	if c.TokenURL != "" {
		return c.TokenURL
	}
	return fmt.Sprintf(defaultOAuthTokenURL, url.PathEscape(c.TenantID))
}
