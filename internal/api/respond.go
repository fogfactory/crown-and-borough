package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/i18n"
)

func writeResolutionError(w http.ResponseWriter, err error, language i18n.Language) {
	var inputErrors *engine.InputErrors
	if errors.As(err, &inputErrors) {
		localized := engine.InputErrors{Errors: make([]engine.InputError, len(inputErrors.Errors))}
		for index, inputError := range inputErrors.Errors {
			localized.Errors[index] = localizedInputError(inputError, language)
		}
		message := i18n.Translate(language, i18n.Message{Key: i18n.ErrorPlayerRequired})
		if len(localized.Errors) > 0 {
			message = localized.Errors[0].Message
		}
		writeJSON(w, http.StatusBadRequest, struct {
			Error   string              `json:"error"`
			Message string              `json:"message"`
			Errors  []engine.InputError `json:"errors"`
		}{Error: "invalid_orders", Message: message, Errors: localized.Errors})
		return
	}
	writeAPIError(w, http.StatusInternalServerError, "resolution_failed", err.Error())
}

func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}{Error: code, Message: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// writeJSONFlushed writes the response with an explicit Content-Length and
// flushes it, so the client has the complete answer before the handler goes on
// with work that follows (the notifications).
func writeJSONFlushed(w http.ResponseWriter, status int, value any) {
	var buffer bytes.Buffer
	if err := json.NewEncoder(&buffer).Encode(value); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "encoding_failed", "response could not be encoded")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(buffer.Len()))
	w.WriteHeader(status)
	_, _ = w.Write(buffer.Bytes())
	_ = http.NewResponseController(w).Flush()
}

// decodeJSONBody decodes a JSON request body of at most limit bytes into
// target, rejecting unknown fields.
func decodeJSONBody(w http.ResponseWriter, r *http.Request, limit int64, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}
