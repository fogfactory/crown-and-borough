// Package firestorestore contains the Firestore persistence adapter for the
// hosted game store. The engine and the public store contracts remain unaware
// of this package and of the Firestore SDK.
package firestorestore

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	cloudfirestore "cloud.google.com/go/firestore"

	"github.com/fogfactory/crown-and-borough/internal/api"
	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/store"
)

const (
	DefaultLeaseTimeout     = 30 * time.Second
	DefaultOperationTimeout = 15 * time.Second
	DefaultMaximumReports   = 64

	defaultFirestoreDatabase = "(default)"
)

var (
	ErrMissingProjectID = errors.New("firestorestore: Firebase project ID is required")
	ErrNilClient        = errors.New("firestorestore: Firestore client is required")
	ErrSchemaVersion    = errors.New("firestorestore: unsupported schema version")
	ErrInconsistentGame = errors.New("firestorestore: inconsistent game documents")
)

// Options controls an adapter created with an existing Firestore client.
// Client is intentionally accepted from the caller so tests can use the
// Firestore emulator and the server can own client shutdown explicitly.
type Options struct {
	Client                  *cloudfirestore.Client
	PrivacyTracker          store.PrivacyTracker
	InvitationCodeGenerator func() (string, error)
	StrictMembership        bool
	LeaseTimeout            time.Duration
	OperationTimeout        time.Duration
	MaximumReports          int
	Now                     func() time.Time
	CloseClientOnClose      bool
}

// FirestoreStore persists the complete game lifecycle in Firestore. No game
// state is cached between calls; this is deliberate because Cloud Run may
// restart or route consecutive requests to different instances.
type FirestoreStore struct {
	client                  *cloudfirestore.Client
	balance                 assetgen.Balance
	assets                  assetgen.Assets
	privacyTracker          store.PrivacyTracker
	strictMembership        bool
	leaseTimeout            time.Duration
	operationTimeout        time.Duration
	maximumReports          int
	now                     func() time.Time
	closeClient             bool
	invitationCodeGenerator func() (string, error)
	metrics                 firestoreMetrics
}

var _ store.InvitationGameStore = (*FirestoreStore)(nil)
var _ store.RevisionedGameStore = (*FirestoreStore)(nil)

// NewFromEnv creates a Firestore-backed store using Application Default
// Credentials. FIRESTORE_EMULATOR_HOST is honored by the Cloud Firestore SDK
// when present; no service-account key is read from disk.
func NewFromEnv(ctx context.Context, balance assetgen.Balance, assets assetgen.Assets, options Options) (*FirestoreStore, error) {
	projectID := strings.TrimSpace(os.Getenv("FIREBASE_PROJECT_ID"))
	if projectID == "" {
		projectID = strings.TrimSpace(os.Getenv("GOOGLE_CLOUD_PROJECT"))
	}
	if projectID == "" {
		return nil, ErrMissingProjectID
	}
	if ctx == nil {
		ctx = context.Background()
	}
	databaseID := strings.TrimSpace(os.Getenv("FIRESTORE_DATABASE_ID"))
	if databaseID == "" {
		databaseID = defaultFirestoreDatabase
	}
	var (
		client *cloudfirestore.Client
		err    error
	)
	if databaseID == defaultFirestoreDatabase {
		client, err = cloudfirestore.NewClient(ctx, projectID)
	} else {
		client, err = cloudfirestore.NewClientWithDatabase(ctx, projectID, databaseID)
	}
	if err != nil {
		return nil, fmt.Errorf("firestorestore: initialize Firestore client: %w", err)
	}
	options.Client = client
	options.CloseClientOnClose = true
	return NewWithClient(balance, assets, options), nil
}

// NewWithClient creates a store around an already initialized Firestore client.
func NewWithClient(balance assetgen.Balance, assets assetgen.Assets, options Options) *FirestoreStore {
	if options.LeaseTimeout <= 0 {
		options.LeaseTimeout = DefaultLeaseTimeout
	}
	if options.OperationTimeout <= 0 {
		options.OperationTimeout = DefaultOperationTimeout
	}
	if options.MaximumReports <= 0 {
		options.MaximumReports = DefaultMaximumReports
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.PrivacyTracker == nil {
		options.PrivacyTracker = api.TrackTurnPrivacy
	}
	return &FirestoreStore{
		client:                  options.Client,
		balance:                 balance,
		assets:                  assets,
		privacyTracker:          options.PrivacyTracker,
		strictMembership:        options.StrictMembership,
		leaseTimeout:            options.LeaseTimeout,
		operationTimeout:        options.OperationTimeout,
		maximumReports:          options.MaximumReports,
		now:                     options.Now,
		closeClient:             options.CloseClientOnClose,
		invitationCodeGenerator: options.InvitationCodeGenerator,
	}
}

// Close releases the SDK client when this store owns it. A store built with
// NewWithClient leaves an injected client open unless explicitly configured to
// close it.
func (s *FirestoreStore) Close() error {
	if s == nil || s.client == nil || !s.closeClient {
		return nil
	}
	return s.client.Close()
}

func (s *FirestoreStore) operationContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithTimeout(ctx, s.operationTimeout)
}

func (s *FirestoreStore) requireClient() error {
	if s == nil || s.client == nil {
		return ErrNilClient
	}
	return nil
}

func newID(prefix string) (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	return prefix + hex.EncodeToString(bytes[:]), nil
}

func newOperationID() (string, error) {
	return newID("op-")
}
