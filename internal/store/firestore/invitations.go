package firestorestore

import (
	"context"
	cryptorand "crypto/rand"
	"errors"
	"strings"

	cloudfirestore "cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/fogfactory/crown-and-borough/internal/store"
)

const invitationAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

func (s *FirestoreStore) CreateInvitation(ctx context.Context, actor store.Actor, id store.GameID) (store.InvitationSecret, error) {
	if err := s.requireClient(); err != nil {
		return store.InvitationSecret{}, err
	}
	id = store.GameID(strings.TrimSpace(string(id)))
	actorID := strings.TrimSpace(actor.ID)
	if id == "" || actorID == "" {
		return store.InvitationSecret{}, store.ErrInvalidInvitation
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	snapshot, err := gameRef(s.client, id).Get(operationContext)
	s.recordReads(1)
	if err != nil {
		return store.InvitationSecret{}, mapReadError(err, store.ErrUnknownGame)
	}
	document, err := decodeGameDocument(snapshot)
	if err != nil {
		return store.InvitationSecret{}, err
	}
	if document.OwnerUID != actorID {
		return store.InvitationSecret{}, store.ErrNotCreator
	}
	if assignedDocumentPlayerCount(document.Players) >= len(document.Players) {
		return store.InvitationSecret{}, store.ErrGameFull
	}
	return s.createInvitation(operationContext, id, actorID)
}

func (s *FirestoreStore) createInvitation(ctx context.Context, id store.GameID, createdBy string) (store.InvitationSecret, error) {
	for range 32 {
		select {
		case <-ctx.Done():
			return store.InvitationSecret{}, ctx.Err()
		default:
		}
		code, err := s.generateInvitationCode()
		if err != nil {
			return store.InvitationSecret{}, err
		}
		hash := store.InvitationCodeHash(code)
		document := invitationDocument{
			SchemaVersion: schemaVersion,
			GameID:        id,
			CreatedBy:     createdBy,
			CodeHash:      hash,
			CreatedAt:     s.now().UTC(),
			Active:        true,
		}
		_, err = invitationRef(s.client, hash).Create(ctx, document)
		if status.Code(err) == codes.AlreadyExists {
			continue
		}
		if err != nil {
			return store.InvitationSecret{}, wrapOperation("create invitation", err)
		}
		s.recordWrites(1)
		return store.InvitationSecret{GameID: id, CreatedBy: createdBy, Code: code}, nil
	}
	return store.InvitationSecret{}, errors.New("firestorestore: invitation code collision limit reached")
}

func (s *FirestoreStore) LookupInvitation(ctx context.Context, code string) (store.Invitation, error) {
	if err := s.requireClient(); err != nil {
		return store.Invitation{}, err
	}
	code = strings.ToUpper(strings.TrimSpace(code))
	if !validInvitationCode(code) {
		return store.Invitation{}, store.ErrInvalidInvitation
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	snapshot, err := invitationRef(s.client, store.InvitationCodeHash(code)).Get(operationContext)
	s.recordReads(1)
	if err != nil {
		return store.Invitation{}, mapReadError(err, store.ErrInvalidInvitation)
	}
	document, err := decodeInvitationDocument(snapshot)
	if err != nil {
		return store.Invitation{}, err
	}
	if !document.Active {
		return store.Invitation{}, store.ErrInvitationInactive
	}
	return store.Invitation{
		GameID:    document.GameID,
		CreatedBy: document.CreatedBy,
		CodeHash:  document.CodeHash,
		CreatedAt: document.CreatedAt,
		Active:    document.Active,
	}, nil
}

func (s *FirestoreStore) DisableInvitation(ctx context.Context, codeHash string) error {
	if err := s.requireClient(); err != nil {
		return err
	}
	codeHash = strings.ToLower(strings.TrimSpace(codeHash))
	if codeHash == "" {
		return store.ErrInvalidInvitation
	}
	operationContext, cancel := s.operationContext(ctx)
	defer cancel()
	doc := invitationRef(s.client, codeHash)
	s.recordTransaction()
	err := s.client.RunTransaction(operationContext, func(tx context.Context, transaction *cloudfirestore.Transaction) error {
		snapshot, err := transaction.Get(doc)
		s.recordReads(1)
		if status.Code(err) == codes.NotFound {
			return store.ErrInvalidInvitation
		}
		if err != nil {
			return err
		}
		document, err := decodeInvitationDocument(snapshot)
		if err != nil {
			return err
		}
		document.Active = false
		s.recordWrites(1)
		return transaction.Set(doc, document)
	})
	return wrapOperation("disable invitation", err)
}

func (s *FirestoreStore) generateInvitationCode() (string, error) {
	if s.invitationCodeGenerator != nil {
		return s.invitationCodeGenerator()
	}
	code := make([]byte, store.InvitationCodeLength)
	limit := 256 - (256 % len(invitationAlphabet))
	for index := range code {
		for {
			var value [1]byte
			if _, err := cryptorand.Read(value[:]); err != nil {
				return "", err
			}
			if int(value[0]) >= limit {
				continue
			}
			code[index] = invitationAlphabet[int(value[0])%len(invitationAlphabet)]
			break
		}
	}
	return string(code), nil
}

func validInvitationCode(code string) bool {
	if len(code) != store.InvitationCodeLength {
		return false
	}
	for _, character := range code {
		if !strings.ContainsRune(invitationAlphabet, character) {
			return false
		}
	}
	return true
}
