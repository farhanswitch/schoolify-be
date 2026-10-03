package utilities

import (
	"encoding/json"
	"net/http"
	"strings"

	errUtility "schoolify_be/utilities/errors"
)

// WriteJSON is the primary helper for sending JSON HTTP responses.
func WriteJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// SimpleValidationError writes a validation error response (HTTP 400) with a validation message.
func SimpleValidationError(errorName string, errorData error, w http.ResponseWriter) {
	errPayload := map[string]interface{}{
		"message": errorName,
		"errors": map[string]string{
			"validation": errorData.Error(),
		},
	}
	WriteJSON(w, http.StatusBadRequest, errPayload)
}

// SimpleUnhandledError writes a generic internal server error response (HTTP 500) to hide raw errors from clients.
func SimpleUnhandledError(w http.ResponseWriter) {
	errPayload := map[string]interface{}{
		"errors": "Internal Server Error. Please contact our administrator",
	}
	WriteJSON(w, http.StatusInternalServerError, errPayload)
}

// WriteErrorResponse handles database and Go errors, returning HTTP 500 for unexpected database errors.
func WriteErrorResponse(w http.ResponseWriter, err error, defaultStatus int) {
	sanitized := errUtility.SanitizeError(err)
	status := defaultStatus

	if errUtility.IsUnexpectedDBError(err) {
		status = http.StatusInternalServerError
	} else {
		errStr := err.Error()
		if errStr == "product is temporarily locked by another transaction, please try again" {
			status = http.StatusConflict
		} else if strings.Contains(errStr, "not found") {
			status = http.StatusNotFound
		} else if strings.Contains(errStr, "unauthorized") || strings.Contains(errStr, "invalid token") || strings.Contains(errStr, "expired") {
			status = http.StatusUnauthorized
		}
	}
	WriteJSON(w, status, map[string]string{"errors": sanitized})
}
