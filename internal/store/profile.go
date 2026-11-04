package store

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

// releaseVersionPattern accepts the SemVer versions stamped by release-please,
// with an optional pre-release or build suffix. Keep it aligned with
// VERSION_PATTERN in web/src/lib/release-notes.ts.
var releaseVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+([-+][0-9A-Za-z.+-]*)?$`)

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

func (s *MemoryProfileStore) UpdateReleaseNotes(ctx context.Context, uid string, update ReleaseNotesUpdate) (PlayerProfile, error) {
	if err := contextError(ctx); err != nil {
		return PlayerProfile{}, err
	}
	uid = strings.TrimSpace(uid)
	update, err := NormalizeReleaseNotesUpdate(update)
	if err != nil {
		return PlayerProfile{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	profile, ok := s.profiles[uid]
	if !ok {
		return PlayerProfile{}, ErrProfileNotFound
	}
	if !ApplyReleaseNotesUpdate(&profile, update) {
		return profile, nil
	}
	profile.UpdatedAt = s.now().UTC()
	s.profiles[uid] = profile
	return profile, nil
}

// NormalizeReleaseNotesUpdate validates an update and returns it with a
// trimmed version, stored without its "v" prefix.
func NormalizeReleaseNotesUpdate(update ReleaseNotesUpdate) (ReleaseNotesUpdate, error) {
	if update.LastSeenVersion == nil && update.HideReleaseNotes == nil {
		return ReleaseNotesUpdate{}, ErrInvalidReleaseNotes
	}
	if update.LastSeenVersion == nil {
		return update, nil
	}
	version := strings.TrimPrefix(strings.TrimSpace(*update.LastSeenVersion), "v")
	if !releaseVersionPattern.MatchString(version) {
		return ReleaseNotesUpdate{}, ErrInvalidReleaseNotes
	}
	update.LastSeenVersion = &version
	return update, nil
}

// ApplyReleaseNotesUpdate mutates profile with a normalized update and reports
// whether anything changed.
func ApplyReleaseNotesUpdate(profile *PlayerProfile, update ReleaseNotesUpdate) bool {
	changed := false
	if update.LastSeenVersion != nil && profile.LastSeenVersion != *update.LastSeenVersion {
		profile.LastSeenVersion = *update.LastSeenVersion
		changed = true
	}
	if update.HideReleaseNotes != nil && profile.HideReleaseNotes != *update.HideReleaseNotes {
		profile.HideReleaseNotes = *update.HideReleaseNotes
		changed = true
	}
	return changed
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
