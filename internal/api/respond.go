package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/i18n"
)

func writeResolutionError(w http.ResponseWriter, err error, language i18n.Language) {
	var inputErrors *engine.InputErrors
	if errors.As(err, &inputErrors) {
		localized := engine.InputErrors{Errors: append([]engine.InputError(nil), inputErrors.Errors...)}
		for index := range localized.Errors {
			inputError := &localized.Errors[index]
			if inputError.MessageKey != "" {
				inputError.Message = i18n.Translate(language, i18n.Message{Key: inputError.MessageKey, Args: inputError.MessageArgs})
			}
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
