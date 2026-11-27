package api

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/fogfactory/crown-and-borough/internal/notify"
	"github.com/fogfactory/crown-and-borough/internal/store"
)

// discordWebhookActive tells whether the game announces to Discord, for the
// game view every player reads. A read failure reports false: the view must
// not fail because of it.
func (h *GamesHandler) discordWebhookActive(ctx context.Context, id store.GameID) bool {
	if h.discord == nil {
		return false
	}
	active, err := h.discord.Active(ctx, id)
	if err != nil {
		log.Printf("discord webhook: read state game=%s: %v", id, err)
		return false
	}
	return active
}

// discordEnabled answers 404 when the channel is not mounted (hotseat).
func (h *GamesHandler) discordEnabled(w http.ResponseWriter) bool {
	if h.discord == nil {
		writeAPIError(w, http.StatusNotFound, "discord_disabled", "Discord webhooks are not available in this mode")
		return false
	}
	return true
}

// hostOnly loads the game and requires the actor to be its creator. On
// failure it has written the error response.
func (h *GamesHandler) hostOnly(w http.ResponseWriter, r *http.Request, actor store.Actor, id store.GameID) (store.GameSnapshot, bool) {
	if !h.discordEnabled(w) {
		return store.GameSnapshot{}, false
	}
	snapshot, err := h.store.Get(r.Context(), actor, id)
	if err != nil {
		h.writeStoreError(w, err)
		return store.GameSnapshot{}, false
	}
	if creator := strings.TrimSpace(snapshot.CreatedBy); creator == "" || creator != strings.TrimSpace(actor.ID) {
		h.writeStoreError(w, store.ErrNotCreator)
		return store.GameSnapshot{}, false
	}
	return snapshot, true
}

// discordSaveBody is the PUT body. URL is optional once a webhook is
// configured: left out, the stored URL is kept and only the settings change.
type discordSaveBody struct {
	URL       string                 `json:"url"`
	Language  string                 `json:"language"`
	Templates store.DiscordTemplates `json:"templates"`
}

type discordTestBody struct {
	URL      string `json:"url"`
	Language string `json:"language"`
}

type discordPreviewBody struct {
	// Event is "newTurn" or "publicMessage".
	Event    string `json:"event"`
	Language string `json:"language"`
	Template string `json:"template"`
}

func (h *GamesHandler) getDiscordWebhook(w http.ResponseWriter, r *http.Request, actor store.Actor, id store.GameID) {
	if _, ok := h.hostOnly(w, r, actor, id); !ok {
		return
	}
	status, err := h.discord.Status(r.Context(), id)
	if err != nil {
		h.writeDiscordError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *GamesHandler) putDiscordWebhook(w http.ResponseWriter, r *http.Request, actor store.Actor, id store.GameID) {
	if _, ok := h.hostOnly(w, r, actor, id); !ok {
		return
	}
	var body discordSaveBody
	if err := decodeJSONBody(w, r, 8<<10, &body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_discord_request", "invalid request body")
		return
	}
	status, err := h.discord.Save(r.Context(), id, notify.DiscordSave{URL: body.URL, Language: body.Language, Templates: body.Templates})
	if err != nil {
		h.writeDiscordError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *GamesHandler) deleteDiscordWebhook(w http.ResponseWriter, r *http.Request, actor store.Actor, id store.GameID) {
	if _, ok := h.hostOnly(w, r, actor, id); !ok {
		return
	}
	if err := h.discord.Delete(r.Context(), id); err != nil {
		h.writeDiscordError(w, err)
		return
	}
	status, err := h.discord.Status(r.Context(), id)
	if err != nil {
		h.writeDiscordError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *GamesHandler) testDiscordWebhook(w http.ResponseWriter, r *http.Request, actor store.Actor, id store.GameID) {
	snapshot, ok := h.hostOnly(w, r, actor, id)
	if !ok {
		return
	}
	// The body is optional: with a URL, the test goes to the webhook of the
	// form and stores nothing.
	var body discordTestBody
	if err := decodeJSONBody(w, r, 8<<10, &body); err != nil && !errors.Is(err, io.EOF) {
		writeAPIError(w, http.StatusBadRequest, "invalid_discord_request", "invalid request body")
		return
	}
	turn, season := snapshotCalendar(snapshot)
	err := h.discord.Test(r.Context(), notify.DiscordGame{ID: id, Name: snapshot.Name, Turn: turn, Season: season}, body.URL, body.Language)
	if err != nil {
		h.writeDiscordError(w, err)
		return
	}
	status, err := h.discord.Status(r.Context(), id)
	if err != nil {
		h.writeDiscordError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

// previewDiscordWebhook renders a template with example values; it neither
// reads nor sends anything.
func (h *GamesHandler) previewDiscordWebhook(w http.ResponseWriter, r *http.Request, actor store.Actor, id store.GameID) {
	if _, ok := h.hostOnly(w, r, actor, id); !ok {
		return
	}
	var body discordPreviewBody
	if err := decodeJSONBody(w, r, 8<<10, &body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_discord_request", "invalid request body")
		return
	}
	preview, err := h.discord.Preview(body.Event, body.Language, body.Template)
	if err != nil {
		h.writeDiscordError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

// writeDiscordError maps the errors of the Discord service. Messages never
// carry the webhook URL.
func (h *GamesHandler) writeDiscordError(w http.ResponseWriter, err error) {
	var unknown *notify.UnknownVariableError
	switch {
	case errors.As(err, &unknown):
		writeJSON(w, http.StatusBadRequest, struct {
			Error    string `json:"error"`
			Message  string `json:"message"`
			Variable string `json:"variable"`
		}{"invalid_template", err.Error(), unknown.Name})
	case errors.Is(err, notify.ErrTemplateTooLong):
		writeAPIError(w, http.StatusBadRequest, "template_too_long", "the message could exceed 2000 characters once its variables are replaced")
	case errors.Is(err, notify.ErrInvalidWebhookURL):
		writeAPIError(w, http.StatusBadRequest, "invalid_webhook_url", "not a Discord webhook URL (https://discord.com/api/webhooks/{id}/{token})")
	case errors.Is(err, notify.ErrInvalidLanguage):
		writeAPIError(w, http.StatusBadRequest, "invalid_language", "language must be fr or en")
	case errors.Is(err, notify.ErrUnknownEvent):
		writeAPIError(w, http.StatusBadRequest, "invalid_event", "event must be newTurn or publicMessage")
	case errors.Is(err, notify.ErrWebhookNotConfigured):
		writeAPIError(w, http.StatusConflict, "webhook_not_configured", "no Discord webhook is configured for this game")
	case errors.Is(err, notify.ErrWebhookGone):
		writeAPIError(w, http.StatusConflict, "webhook_gone", "Discord refused the webhook (deleted or invalid): it is disabled")
	case errors.Is(err, notify.ErrTestRateLimited), errors.Is(err, notify.ErrDiscordRateLimited):
		writeAPIError(w, http.StatusTooManyRequests, "discord_rate_limited", "too many messages, try again in a moment")
	case errors.Is(err, notify.ErrDiscordFailed), errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		writeAPIError(w, http.StatusBadGateway, "discord_unreachable", "Discord did not accept the message")
	default:
		log.Printf("discord webhook: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "discord_unavailable", "the Discord webhook could not be read or saved")
	}
}
