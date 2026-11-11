package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/fogfactory/crown-and-borough/internal/store"
)

type chatReadRequest struct {
	Channel string `json:"channel"`
}

type chatHistoryResponse struct {
	Messages []store.ChatMessage `json:"messages"`
	// NextBefore is the cursor of the following page, to send back as
	// before and beforeId: the createdAt and ID of the oldest message
	// returned. Absent when the page is not full.
	NextBefore   string `json:"nextBefore,omitempty"`
	NextBeforeID string `json:"nextBeforeId,omitempty"`
}

// chatEnabled answers 404 when the chat is not mounted (hotseat).
func (h *GamesHandler) chatEnabled(w http.ResponseWriter) bool {
	if h.chat == nil {
		writeAPIError(w, http.StatusNotFound, "chat_disabled", "the chat is not available in this mode")
		return false
	}
	return true
}

func (h *GamesHandler) postMessage(w http.ResponseWriter, r *http.Request, actor store.Actor, id store.GameID) {
	if !h.chatEnabled(w) {
		return
	}
	// Channel is "public" or the recipient house; BatchID, Part and Parts tag
	// the pieces of a long text split by the client.
	var request store.ChatPostRequest
	if err := decodeJSONBody(w, r, 16<<10, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_message_request", err.Error())
		return
	}
	result, err := h.chat.Post(r.Context(), actor, id, request)
	if err != nil {
		h.writeChatError(w, err)
		return
	}
	h.closeAfterNotifications(w)
	writeJSONFlushed(w, http.StatusCreated, result.Message)
	h.afterResponse(r, func(ctx context.Context) { h.notifier.MessagePosted(ctx, result) })
}

func (h *GamesHandler) markChannelRead(w http.ResponseWriter, r *http.Request, actor store.Actor, id store.GameID) {
	if !h.chatEnabled(w) {
		return
	}
	var request chatReadRequest
	if err := decodeJSONBody(w, r, 1<<10, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_message_request", err.Error())
		return
	}
	cursor, err := h.chat.MarkRead(r.Context(), actor, id, request.Channel)
	if err != nil {
		h.writeChatError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cursor)
}

func (h *GamesHandler) listMessages(w http.ResponseWriter, r *http.Request, actor store.Actor, id store.GameID) {
	if !h.chatEnabled(w) {
		return
	}
	query := r.URL.Query()
	request := store.ChatHistoryRequest{Channel: query.Get("channel")}
	if value := query.Get("before"); value != "" {
		before, err := time.Parse(time.RFC3339Nano, value)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_message_request", "before must be an RFC 3339 timestamp")
			return
		}
		request.Before = store.ChatCursor{CreatedAt: before, ID: query.Get("beforeId")}
	}
	if value := query.Get("limit"); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil || limit < 1 {
			writeAPIError(w, http.StatusBadRequest, "invalid_message_request", "limit must be a positive integer")
			return
		}
		request.Limit = limit
	}
	messages, err := h.chat.History(r.Context(), actor, id, request)
	if err != nil {
		h.writeChatError(w, err)
		return
	}
	response := chatHistoryResponse{Messages: messages}
	if len(messages) > 0 && len(messages) >= store.ChatPageSize(request.Limit) {
		oldest := messages[len(messages)-1]
		response.NextBefore = oldest.CreatedAt.UTC().Format(time.RFC3339Nano)
		response.NextBeforeID = oldest.ID
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *GamesHandler) writeChatError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrChatRateLimited):
		w.Header().Set("Retry-After", strconv.Itoa(int(store.ChatMinInterval.Seconds())))
		writeAPIError(w, http.StatusTooManyRequests, "rate_limited", "at most one message per second")
	case errors.Is(err, store.ErrChatEmptyBody):
		writeAPIError(w, http.StatusBadRequest, "empty_message", "the message is empty")
	case errors.Is(err, store.ErrChatBodyTooLong):
		writeAPIError(w, http.StatusBadRequest, "message_too_long", fmt.Sprintf("a message holds at most %d characters", store.ChatMaxBodyRunes))
	case errors.Is(err, store.ErrChatInvalidBatch):
		writeAPIError(w, http.StatusBadRequest, "invalid_batch", "batchId, part and parts must come together with 1 <= part <= parts <= 8")
	case errors.Is(err, store.ErrChatInvalidChannel):
		writeAPIError(w, http.StatusBadRequest, "invalid_channel", "the channel must be public or a house of this game")
	case errors.Is(err, store.ErrChatInvalidRecipient):
		writeAPIError(w, http.StatusBadRequest, "invalid_recipient", "the recipient must be another house still in play, or one already in conversation with you")
	case errors.Is(err, store.ErrChatNoHouse):
		writeAPIError(w, http.StatusForbidden, "no_house", "only a house can take part in the chat")
	default:
		h.writeStoreError(w, err)
	}
}
