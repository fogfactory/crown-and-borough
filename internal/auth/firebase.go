package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	firebase "firebase.google.com/go/v4"
	firebaseauth "firebase.google.com/go/v4/auth"
)

var (
	ErrMissingProjectID = errors.New("auth: Firebase project ID is required")
	ErrInvalidIdentity  = errors.New("auth: Firebase token has no usable identity")
)

type FirebaseVerifier struct {
	client       *firebaseauth.Client
	projectID    string
	checkRevoked bool
}

type FirebaseVerifierOptions struct {
	CheckRevoked bool
}

func NewFirebaseVerifier(ctx context.Context, projectID string) (*FirebaseVerifier, error) {
	return NewFirebaseVerifierWithOptions(ctx, projectID, FirebaseVerifierOptions{CheckRevoked: true})
}

func NewFirebaseVerifierWithOptions(ctx context.Context, projectID string, options FirebaseVerifierOptions) (*FirebaseVerifier, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, ErrMissingProjectID
	}
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("auth: initialize Firebase app: %w", err)
	}
	client, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("auth: initialize Firebase Auth client: %w", err)
	}
	return &FirebaseVerifier{client: client, projectID: projectID, checkRevoked: options.CheckRevoked}, nil
}

func NewFirebaseVerifierFromEnv(ctx context.Context) (*FirebaseVerifier, error) {
	projectID := strings.TrimSpace(os.Getenv("FIREBASE_PROJECT_ID"))
	if projectID == "" {
		projectID = strings.TrimSpace(os.Getenv("GOOGLE_CLOUD_PROJECT"))
	}
	return NewFirebaseVerifier(ctx, projectID)
}

func (v *FirebaseVerifier) VerifyIDToken(ctx context.Context, idToken string) (Identity, error) {
	if v == nil || v.client == nil || strings.TrimSpace(idToken) == "" {
		return Identity{}, ErrInvalidIdentity
	}
	if ctx == nil {
		ctx = context.Background()
	}
	verifyContext, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var (
		token *firebaseauth.Token
		err   error
	)
	if v.checkRevoked {
		token, err = v.client.VerifyIDTokenAndCheckRevoked(verifyContext, idToken)
	} else {
		token, err = v.client.VerifyIDToken(verifyContext, idToken)
	}
	if err != nil {
		return Identity{}, fmt.Errorf("auth: verify Firebase ID token: %w", err)
	}
	if token == nil || strings.TrimSpace(token.UID) == "" {
		return Identity{}, ErrInvalidIdentity
	}
	if !validFirebaseProject(token.Claims, v.projectID) {
		return Identity{}, ErrInvalidIdentity
	}
	email, _ := token.Claims["email"].(string)
	return Identity{UID: strings.TrimSpace(token.UID), Email: strings.TrimSpace(email)}, nil
}

func validFirebaseProject(claims map[string]interface{}, projectID string) bool {
	audience, ok := claims["aud"].(string)
	if !ok || audience != projectID {
		return false
	}
	issuer, ok := claims["iss"].(string)
	return ok && issuer == "https://securetoken.google.com/"+projectID
}
