package review

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

// GitHubAppAuth handles GitHub App authentication
type GitHubAppAuth struct {
	appID      int64
	privateKey *rsa.PrivateKey
	client     *github.Client
}

// NewGitHubAppAuth creates a new GitHub App authenticator
func NewGitHubAppAuth(appID int64, privateKeyPath string) (*GitHubAppAuth, error) {
	// Read private key
	keyData, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	// Parse private key
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Create client for JWT requests
	client := github.NewClient(nil)

	return &GitHubAppAuth{
		appID:      appID,
		privateKey: privateKey,
		client:     client,
	}, nil
}

// GenerateJWT creates a JWT for GitHub App authentication
func (auth *GitHubAppAuth) GenerateJWT() (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(10 * time.Minute)),
		Issuer:    fmt.Sprintf("%d", auth.appID),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(auth.privateKey)
}

// GetInstallationToken gets an access token for a specific installation
func (auth *GitHubAppAuth) GetInstallationToken(ctx context.Context, installationID int64) (string, error) {
	// Generate JWT
	jwt, err := auth.GenerateJWT()
	if err != nil {
		return "", fmt.Errorf("failed to generate JWT: %w", err)
	}

	// Create authenticated client with JWT
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: jwt})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Get installation access token
	token, _, err := client.Apps.CreateInstallationToken(ctx, installationID, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create installation token: %w", err)
	}

	return token.GetToken(), nil
}
