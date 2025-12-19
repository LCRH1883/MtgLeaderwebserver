package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"MtgLeaderwebserver/internal/domain"
)

type errorEnvelope struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, errorEnvelope{Error: apiError{Code: code, Message: message}})
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func WriteDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrValidation):
		WriteError(w, http.StatusBadRequest, "validation_error", "invalid request")
	case errors.Is(err, domain.ErrUsernameTaken):
		WriteError(w, http.StatusConflict, "username_taken", "username already taken")
	case errors.Is(err, domain.ErrEmailTaken):
		WriteError(w, http.StatusConflict, "email_taken", "email already taken")
	case errors.Is(err, domain.ErrInvalidCredentials):
		WriteError(w, http.StatusUnauthorized, "invalid_credentials", "invalid login or password")
	case errors.Is(err, domain.ErrUnauthorized):
		WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
	case errors.Is(err, domain.ErrForbidden):
		WriteError(w, http.StatusForbidden, "forbidden", "forbidden")
	case errors.Is(err, domain.ErrUserDisabled):
		WriteError(w, http.StatusForbidden, "user_disabled", "user is disabled")
	case errors.Is(err, domain.ErrFriendshipExists):
		WriteError(w, http.StatusConflict, "friendship_exists", "friend request already exists")
	case errors.Is(err, domain.ErrNotFound):
		WriteError(w, http.StatusNotFound, "not_found", "not found")
	default:
		WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
