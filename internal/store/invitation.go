package store

import (
	"context"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"
)

const InvitationCodeLength = 6

// The alphabet avoids characters that are commonly confused when a code is
// copied from a screen or dictated over a voice call.
const invitationAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

var ErrInvitationCodeGenerator = errors.New("store: invitation code generator failed")

type InvitationCodeGenerator func() (string, error)

type MemoryInvitationStore struct {
	mu          sync.RWMutex
	invitations map[string]Invitation
	generator   InvitationCodeGenerator
	now         func() time.Time
}

var _ InvitationStore = (*MemoryInvitationStore)(nil)

func NewMemoryInvitationStore() *MemoryInvitationStore {
	return NewMemoryInvitationStoreWithGenerator(nil)
}

func NewMemoryInvitationStoreWithGenerator(generator InvitationCodeGenerator) *MemoryInvitationStore {
	if generator == nil {
		generator = randomInvitationCode
	}
	return &MemoryInvitationStore{
		invitations: make(map[string]Invitation),
		generator:   generator,
		now:         time.Now,
	}
}

func (s *MemoryInvitationStore) CreateInvitation(ctx context.Context, gameID GameID, createdBy string) (InvitationSecret, error) {
	if err := contextError(ctx); err != nil {
		return InvitationSecret{}, err
	}
	gameID = GameID(strings.TrimSpace(string(gameID)))
	createdBy = strings.TrimSpace(createdBy)
	if gameID == "" || createdBy == "" {
		return InvitationSecret{}, ErrInvalidInvitation
	}
	for range 32 {
		if err := contextError(ctx); err != nil {
			return InvitationSecret{}, err
		}
		code, err := s.generator()
		if err != nil {
			return InvitationSecret{}, errors.Join(ErrInvitationCodeGenerator, err)
		}
		code = strings.ToUpper(strings.TrimSpace(code))
		if !validInvitationCode(code) {
			return InvitationSecret{}, ErrInvitationCodeGenerator
		}
		hash := InvitationCodeHash(code)
		s.mu.Lock()
		if _, exists := s.invitations[hash]; exists {
			s.mu.Unlock()
			continue
		}
		s.invitations[hash] = Invitation{
			GameID:    gameID,
			CreatedBy: createdBy,
			CodeHash:  hash,
			CreatedAt: s.now().UTC(),
			Active:    true,
		}
		s.mu.Unlock()
		return InvitationSecret{GameID: gameID, CreatedBy: createdBy, Code: code}, nil
	}
	return InvitationSecret{}, ErrInvitationCodeGenerator
}

func (s *MemoryInvitationStore) LookupInvitation(ctx context.Context, code string) (Invitation, error) {
	if err := contextError(ctx); err != nil {
		return Invitation{}, err
	}
	code = strings.ToUpper(strings.TrimSpace(code))
	if !validInvitationCode(code) {
		return Invitation{}, ErrInvalidInvitation
	}
	hash := InvitationCodeHash(code)
	s.mu.RLock()
	invitation, ok := s.invitations[hash]
	s.mu.RUnlock()
	if !ok {
		return Invitation{}, ErrInvalidInvitation
	}
	if !invitation.Active {
		return Invitation{}, ErrInvitationInactive
	}
	return invitation, nil
}

func (s *MemoryInvitationStore) DisableInvitation(ctx context.Context, codeHash string) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	codeHash = strings.ToLower(strings.TrimSpace(codeHash))
	s.mu.Lock()
	defer s.mu.Unlock()
	invitation, ok := s.invitations[codeHash]
	if !ok {
		return ErrInvalidInvitation
	}
	invitation.Active = false
	s.invitations[codeHash] = invitation
	return nil
}

func InvitationCodeHash(code string) string {
	hash := sha256.Sum256([]byte(strings.ToUpper(strings.TrimSpace(code))))
	return hex.EncodeToString(hash[:])
}

func validInvitationCode(code string) bool {
	if len(code) != InvitationCodeLength {
		return false
	}
	for _, character := range code {
		if !strings.ContainsRune(invitationAlphabet, character) {
			return false
		}
	}
	return true
}

func randomInvitationCode() (string, error) {
	code := make([]byte, InvitationCodeLength)
	limit := 256 - (256 % len(invitationAlphabet))
	for index := 0; index < InvitationCodeLength; {
		var value [1]byte
		if _, err := cryptorand.Read(value[:]); err != nil {
			return "", err
		}
		if int(value[0]) >= limit {
			continue
		}
		code[index] = invitationAlphabet[int(value[0])%len(invitationAlphabet)]
		index++
	}
	return string(code), nil
}
