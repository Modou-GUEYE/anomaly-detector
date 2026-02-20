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

func LoadPrivateKeyFromEnv() (*rsa.PrivateKey, error) {
	pemStr := os.Getenv("KEY_JWT_PRIVATE_KEY")
	if pemStr != "" {
		block, _ := pem.Decode([]byte(pemStr))
		if block == nil {
			return nil, errors.New("invalid PEM data in KEY_JWT_PRIVATE_KEY")
		}
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			pk, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err2 != nil {
				return nil, err
			}
			if rsaKey, ok := pk.(*rsa.PrivateKey); ok {
				return rsaKey, nil
			}
			return nil, errors.New("private key is not RSA")
		}
		return key, nil
	}

	if keyFile := os.Getenv("KEY_JWT_PRIVATE_KEY_FILE"); keyFile != "" {
		data, err := os.ReadFile(keyFile)
		if err != nil {
			return nil, err
		}
		block, _ := pem.Decode(data)
		if block == nil {
			return nil, errors.New("invalid PEM data in KEY_JWT_PRIVATE_KEY_FILE")
		}
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			pk, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err2 != nil {
				return nil, err
			}
			if rsaKey, ok := pk.(*rsa.PrivateKey); ok {
				return rsaKey, nil
			}
			return nil, errors.New("private key in file is not RSA")
		}
		return key, nil
	}

	if cachedPriv := getCachedPrivate(); cachedPriv != nil {
		return cachedPriv, nil
	}

	return nil, errors.New("no private RSA key configured: set KEY_JWT_PRIVATE_KEY or KEY_JWT_PRIVATE_KEY_FILE")
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

	// charger la clé privée et signer.
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
