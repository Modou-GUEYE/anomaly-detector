package auth

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// LoadPrivateKeyFromEnv retourne la clé RSA privée. Elle est lue et parsée une seule fois
// puis conservée en mémoire : la relire à chaque requête coûtait un accès disque + un parsing
// x509 sur le chemin critique de chaque appel authentifié. Une rotation de clé exige un redémarrage.
func LoadPrivateKeyFromEnv() (*rsa.PrivateKey, error) {
	if cachedPriv := getCachedPrivate(); cachedPriv != nil {
		return cachedPriv, nil
	}

	if key, ok, err := loadPrivateKeyFromEnvValue("KEY_JWT_PRIVATE_KEY", "invalid PEM data in KEY_JWT_PRIVATE_KEY", "private key is not RSA"); ok {
		if err == nil {
			setCachedPrivate(key)
		}
		return key, err
	}

	if key, ok, err := loadPrivateKeyFromEnvFile("KEY_JWT_PRIVATE_KEY_FILE", "invalid PEM data in KEY_JWT_PRIVATE_KEY_FILE", "private key in file is not RSA"); ok {
		if err == nil {
			setCachedPrivate(key)
		}
		return key, err
	}

	return nil, errors.New("no private RSA key configured: set KEY_JWT_PRIVATE_KEY or KEY_JWT_PRIVATE_KEY_FILE")
}

func loadPrivateKeyFromEnvValue(envVar string, invalidPEMMessage string, notRSAMessage string) (*rsa.PrivateKey, bool, error) {
	pemStr := os.Getenv(envVar)
	if pemStr == "" {
		return nil, false, nil
	}

	key, err := parsePrivateKeyPEM([]byte(pemStr), invalidPEMMessage, notRSAMessage)
	return key, true, err
}

func loadPrivateKeyFromEnvFile(envVar string, invalidPEMMessage string, notRSAMessage string) (*rsa.PrivateKey, bool, error) {
	keyFile := os.Getenv(envVar)
	if keyFile == "" {
		return nil, false, nil
	}

	data, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, true, err
	}

	key, parseErr := parsePrivateKeyPEM(data, invalidPEMMessage, notRSAMessage)
	return key, true, parseErr
}

func parsePrivateKeyPEM(pemData []byte, invalidPEMMessage string, notRSAMessage string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New(invalidPEMMessage)
	}
	return parseRSAPrivateKeyBytes(block.Bytes, notRSAMessage)
}

func parseRSAPrivateKeyBytes(derBytes []byte, notRSAMessage string) (*rsa.PrivateKey, error) {
	key, err := x509.ParsePKCS1PrivateKey(derBytes)
	if err == nil {
		return key, nil
	}

	pk, err2 := x509.ParsePKCS8PrivateKey(derBytes)
	if err2 != nil {
		return nil, err
	}

	rsaKey, ok := pk.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New(notRSAMessage)
	}
	return rsaKey, nil
}

var (
	cachedPrivate *rsa.PrivateKey
	cachedMu      sync.RWMutex
	cachedJwk     map[string]interface{}
	cachedKid     string
)

func getCachedPrivate() *rsa.PrivateKey {
	cachedMu.RLock()
	k := cachedPrivate
	cachedMu.RUnlock()
	return k
}

func setCachedPrivate(k *rsa.PrivateKey) {
	cachedMu.Lock()
	cachedPrivate = k
	cachedMu.Unlock()
}

func ensureCachedJwk() error {

	if cachedJwk != nil && cachedKid != "" {
		return nil
	}
	priv, err := LoadPrivateKeyFromEnv()
	if err != nil {
		return err
	}
	pub := &priv.PublicKey

	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return err
	}
	h := sha256.Sum256(der)
	kid := base64.RawURLEncoding.EncodeToString(h[:])

	n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	eBig := big.NewInt(int64(pub.E))
	e := base64.RawURLEncoding.EncodeToString(eBig.Bytes())

	jwk := map[string]interface{}{
		"kty": "RSA",
		"alg": "RS256",
		"use": "sig",
		"kid": kid,
		"n":   n,
		"e":   e,
	}

	cachedMu.Lock()
	cachedJwk = jwk
	cachedKid = kid
	cachedMu.Unlock()
	return nil
}

func GetJwks() (map[string]interface{}, error) {
	if err := ensureCachedJwk(); err != nil {
		return nil, err
	}
	return map[string]interface{}{"keys": []interface{}{cachedJwk}}, nil
}

func ParseAndValidateToken(tokenStr string) (jwt.MapClaims, error) {
	priv, err := LoadPrivateKeyFromEnv()
	if err != nil {
		return nil, err
	}
	pub := &priv.PublicKey
	var claims jwt.MapClaims
	token, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, errors.New("méthode de signature inattendue, attendu RS256")
		}
		return pub, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func NewSignedTokenFromEnv(subject string, roles []string, ttl time.Duration, mustChangePassword bool) (string, error) {
	claims := jwt.MapClaims{
		"sub":                  subject,
		"roles":                roles,
		"iat":                  time.Now().Unix(),
		"must_change_password": mustChangePassword,
	}
	if ttl > 0 {
		claims["exp"] = time.Now().Add(ttl).Unix()
	}

	return signTokenFromClaims(claims)
}

func NewPasswordResetTokenFromEnv(subject string, email string, ttl time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"sub":     subject,
		"email":   email,
		"purpose": "password_reset",
		"iat":     time.Now().Unix(),
	}
	if ttl > 0 {
		claims["exp"] = time.Now().Add(ttl).Unix()
	}

	return signTokenFromClaims(claims)
}

func signTokenFromClaims(claims jwt.MapClaims) (string, error) {
	priv, err := LoadPrivateKeyFromEnv()
	if err != nil {
		return "", err
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	if err := ensureCachedJwk(); err == nil {
		token.Header["kid"] = cachedKid
	}
	return token.SignedString(priv)
}
