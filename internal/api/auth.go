package api

import (
	"encoding/json"
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

func (h *AuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/")
	if path != "/api/auth/me" {
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
	switch r.Method {
	case http.MethodGet:
		h.getProfile(w, r, actor)
	case http.MethodPut:
		h.updateProfile(w, r, actor)
	default:
		w.Header().Set("Allow", "GET, PUT")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AuthHandler) getProfile(w http.ResponseWriter, r *http.Request, actor store.Actor) {
	profile, err := h.profiles.EnsureProfile(r.Context(), actor)
	if err != nil {
		h.writeProfileError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, profileResponse{Player: profileView{UID: profile.UID, Email: profile.Email, DisplayName: profile.DisplayName}})
}

func (h *AuthHandler) updateProfile(w http.ResponseWriter, r *http.Request, actor store.Actor) {
	var body profileUpdateRequest
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
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
	writeJSON(w, http.StatusOK, profileResponse{Player: profileView{UID: profile.UID, Email: profile.Email, DisplayName: profile.DisplayName}})
}

func (h *AuthHandler) writeProfileError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrInvalidDisplayName):
		writeAPIError(w, http.StatusBadRequest, "invalid_display_name", "display name must contain between one and thirty-two characters")
	case errors.Is(err, store.ErrProfileNotFound):
		writeAPIError(w, http.StatusNotFound, "profile_not_found", err.Error())
	default:
		writeAPIError(w, http.StatusInternalServerError, "profile_unavailable", "profile could not be loaded")
	}
}

type profileUpdateRequest struct {
	DisplayName string `json:"displayName"`
}

type profileResponse struct {
	Player profileView `json:"player"`
}

type profileView struct {
	UID         string `json:"uid"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
}
