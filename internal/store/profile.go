package store

import (
	"context"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	MinimumDisplayNameLength = 1
	MaximumDisplayNameLength = 32
)

// MemoryProfileStore provides the O6 profile contract without coupling the
// game store to a persistence SDK. Firestore can implement ProfileStore later.
type MemoryProfileStore struct {
	mu       sync.RWMutex
	profiles map[string]PlayerProfile
	now      func() time.Time
}

var _ ProfileStore = (*MemoryProfileStore)(nil)

func NewMemoryProfileStore() *MemoryProfileStore {
	return &MemoryProfileStore{
		profiles: make(map[string]PlayerProfile),
		now:      time.Now,
	}
}

func (s *MemoryProfileStore) GetProfile(ctx context.Context, uid string) (PlayerProfile, error) {
	if err := contextError(ctx); err != nil {
		return PlayerProfile{}, err
	}
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return PlayerProfile{}, ErrProfileNotFound
	}
	s.mu.RLock()
	profile, ok := s.profiles[uid]
	s.mu.RUnlock()
	if !ok {
		return PlayerProfile{}, ErrProfileNotFound
	}
	return profile, nil
}

func (s *MemoryProfileStore) EnsureProfile(ctx context.Context, actor Actor) (PlayerProfile, error) {
	if err := contextError(ctx); err != nil {
		return PlayerProfile{}, err
	}
	uid := strings.TrimSpace(actor.ID)
	if uid == "" {
		return PlayerProfile{}, ErrProfileNotFound
	}
	email := strings.TrimSpace(actor.Email)
	s.mu.Lock()
	defer s.mu.Unlock()
	if profile, ok := s.profiles[uid]; ok {
		if email != "" && profile.Email != email {
			profile.Email = email
			profile.UpdatedAt = s.now().UTC()
			s.profiles[uid] = profile
		}
		return profile, nil
	}
	now := s.now().UTC()
	profile := PlayerProfile{
		UID:       uid,
		Email:     email,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.profiles[uid] = profile
	return profile, nil
}

func (s *MemoryProfileStore) UpdateProfile(ctx context.Context, uid, displayName string) (PlayerProfile, error) {
	if err := contextError(ctx); err != nil {
		return PlayerProfile{}, err
	}
	uid = strings.TrimSpace(uid)
	normalized, err := NormalizeDisplayName(displayName)
	if err != nil {
		return PlayerProfile{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	profile, ok := s.profiles[uid]
	if !ok {
		return PlayerProfile{}, ErrProfileNotFound
	}
	if profile.DisplayName == normalized {
		return profile, nil
	}
	profile.DisplayName = normalized
	profile.UpdatedAt = s.now().UTC()
	s.profiles[uid] = profile
	return profile, nil
}

func NormalizeDisplayName(value string) (string, error) {
	if !utf8.ValidString(value) {
		return "", ErrInvalidDisplayName
	}
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	runes := []rune(value)
	if len(runes) < MinimumDisplayNameLength || len(runes) > MaximumDisplayNameLength {
		return "", ErrInvalidDisplayName
	}
	for _, character := range runes {
		if unicode.IsControl(character) {
			return "", ErrInvalidDisplayName
		}
	}
	return value, nil
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
