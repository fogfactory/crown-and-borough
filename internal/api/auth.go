package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/fogfactory/crown-and-borough/internal/store"
)

type AuthHandler struct {
	profiles store.ProfileStore
	actor    ActorResolver
}

func NewAuthHandler(profiles store.ProfileStore, resolve ActorResolver) http.Handler {
	return &AuthHandler{profiles: profiles, actor: resolve}
}

type authMethod func(*AuthHandler, http.ResponseWriter, *http.Request, store.Actor)

type authRoute struct {
	allow   string
	methods map[string]authMethod
}

var authRoutes = map[string]authRoute{
	"/api/auth/me": {allow: "GET, PUT", methods: map[string]authMethod{
		http.MethodGet: (*AuthHandler).getProfile,
		http.MethodPut: (*AuthHandler).updateProfile,
	}},
	"/api/auth/me/release-notes": {allow: "PUT", methods: map[string]authMethod{
		http.MethodPut: (*AuthHandler).updateReleaseNotes,
	}},
	"/api/auth/me/notifications": {allow: "PUT", methods: map[string]authMethod{
		http.MethodPut: (*AuthHandler).updateNotifications,
	}},
}

func (h *AuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	route, ok := authRoutes[strings.TrimSuffix(r.URL.Path, "/")]
	if !ok {
		http.NotFound(w, r)
		return
	}
	if h.profiles == nil {
		writeAPIError(w, http.StatusInternalServerError, "profile_store_unavailable", "profile store is not configured")
		return
	}
	actor, err := actorFromRequest(r, h.actor)
	if err != nil {
		writeActorError(w, err)
		return
	}
	handle, ok := route.methods[r.Method]
	if !ok {
		w.Header().Set("Allow", route.allow)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	handle(h, w, r, actor)
}

func (h *AuthHandler) getProfile(w http.ResponseWriter, r *http.Request, actor store.Actor) {
	profile, err := h.profiles.EnsureProfile(r.Context(), actor)
	if err != nil {
		h.writeProfileError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, profileResponse{Player: newProfileView(profile)})
}

func (h *AuthHandler) updateProfile(w http.ResponseWriter, r *http.Request, actor store.Actor) {
	var body profileUpdateRequest
	if err := decodeJSONBody(w, r, 4<<10, &body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_profile_request", err.Error())
		return
	}
	if _, err := h.profiles.EnsureProfile(r.Context(), actor); err != nil {
		h.writeProfileError(w, err)
		return
	}
	profile, err := h.profiles.UpdateProfile(r.Context(), actor.ID, body.DisplayName)
	if err != nil {
		h.writeProfileError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, profileResponse{Player: newProfileView(profile)})
}

func (h *AuthHandler) updateReleaseNotes(w http.ResponseWriter, r *http.Request, actor store.Actor) {
	var body releaseNotesUpdateRequest
	if err := decodeJSONBody(w, r, 4<<10, &body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_release_notes_request", err.Error())
		return
	}
	if _, err := h.profiles.EnsureProfile(r.Context(), actor); err != nil {
		h.writeProfileError(w, err)
		return
	}
	profile, err := h.profiles.UpdateReleaseNotes(r.Context(), actor.ID, store.ReleaseNotesUpdate{
		LastSeenVersion:  body.LastSeenVersion,
		HideReleaseNotes: body.HideReleaseNotes,
	})
	if err != nil {
		h.writeProfileError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, profileResponse{Player: newProfileView(profile)})
}

func (h *AuthHandler) updateNotifications(w http.ResponseWriter, r *http.Request, actor store.Actor) {
	var body notificationsUpdateRequest
	if err := decodeJSONBody(w, r, 4<<10, &body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_notifications_request", err.Error())
		return
	}
	if _, err := h.profiles.EnsureProfile(r.Context(), actor); err != nil {
		h.writeProfileError(w, err)
		return
	}
	profile, err := h.profiles.UpdateNotifications(r.Context(), actor.ID, store.NotificationsUpdate{
		NewTurn:          body.NewTurn,
		PrivateMessage:   body.PrivateMessage,
		PublicMessage:    body.PublicMessage,
		ReduceWhenActive: body.ReduceWhenActive,
		Language:         body.Language,
	})
	if err != nil {
		h.writeProfileError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, profileResponse{Player: newProfileView(profile)})
}

func (h *AuthHandler) writeProfileError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrInvalidDisplayName):
		writeAPIError(w, http.StatusBadRequest, "invalid_display_name", "display name must contain between one and thirty-two characters")
	case errors.Is(err, store.ErrInvalidReleaseNotes):
		writeAPIError(w, http.StatusBadRequest, "invalid_release_notes", "release notes update needs a lastSeenVersion (SemVer) and/or hideReleaseNotes")
	case errors.Is(err, store.ErrInvalidNotifications):
		writeAPIError(w, http.StatusBadRequest, "invalid_notifications", "notification settings update needs at least one setting; language must be fr or en")
	case errors.Is(err, store.ErrProfileNotFound):
		writeAPIError(w, http.StatusNotFound, "profile_not_found", err.Error())
	default:
		writeAPIError(w, http.StatusInternalServerError, "profile_unavailable", "profile could not be loaded")
	}
}

type profileUpdateRequest struct {
	DisplayName string `json:"displayName"`
}

type releaseNotesUpdateRequest struct {
	LastSeenVersion  *string `json:"lastSeenVersion"`
	HideReleaseNotes *bool   `json:"hideReleaseNotes"`
}

// notificationsUpdateRequest changes the email notification settings; omitted
// fields are left untouched. Language is the app language (fr or en) the
// emails are written in.
type notificationsUpdateRequest struct {
	NewTurn          *bool   `json:"newTurn"`
	PrivateMessage   *bool   `json:"privateMessage"`
	PublicMessage    *bool   `json:"publicMessage"`
	ReduceWhenActive *bool   `json:"reduceWhenActive"`
	Language         *string `json:"language"`
}

type profileResponse struct {
	Player profileView `json:"player"`
}

type profileView struct {
	UID              string `json:"uid"`
	Email            string `json:"email"`
	DisplayName      string `json:"displayName"`
	LastSeenVersion  string `json:"lastSeenVersion"`
	HideReleaseNotes bool   `json:"hideReleaseNotes"`
	// Notifications is additive: clients that predate it ignore it.
	Notifications store.NotificationPreferences `json:"notifications"`
}

func newProfileView(profile store.PlayerProfile) profileView {
	return profileView{
		UID:              profile.UID,
		Email:            profile.Email,
		DisplayName:      profile.DisplayName,
		LastSeenVersion:  profile.LastSeenVersion,
		HideReleaseNotes: profile.HideReleaseNotes,
		Notifications:    profile.Notifications,
	}
}
