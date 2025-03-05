package utils

import (
	"encoding/json"
	"net/http"
)

// ErrorHandler handles all error responses
type ErrorHandler struct{}

// ErrorResponse represents an error response structure
type ErrorResponse struct {
	Status  int           `json:"status"`
	Message string        `json:"message"`
	Errors  []ErrorDetail `json:"errors,omitempty"`
}

// ErrorDetail represents detailed error information
type ErrorDetail struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

// NewErrorHandler creates a new error handler
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{}
}

// HandleError sends a generic error response
func (h *ErrorHandler) HandleError(w http.ResponseWriter, code int, message string) {
	response, _ := json.Marshal(ErrorResponse{
		Status:  code,
		Message: message,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

// HandleValidationError sends a validation error response
func (h *ErrorHandler) HandleValidationError(w http.ResponseWriter, errors []ErrorDetail) {
	response, _ := json.Marshal(ErrorResponse{
		Status:  http.StatusBadRequest,
		Message: "Validation failed",
		Errors:  errors,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	w.Write(response)
}

// HandleBadRequest sends a 400 Bad Request response
func (h *ErrorHandler) HandleBadRequest(w http.ResponseWriter, message string) {
	h.HandleError(w, http.StatusBadRequest, message)
}

// HandleUnauthorized sends a 401 Unauthorized response
func (h *ErrorHandler) HandleUnauthorized(w http.ResponseWriter, message string) {
	h.HandleError(w, http.StatusUnauthorized, message)
}

// HandleForbidden sends a 403 Forbidden response
func (h *ErrorHandler) HandleForbidden(w http.ResponseWriter, message string) {
	h.HandleError(w, http.StatusForbidden, message)
}

// HandleNotFound sends a 404 Not Found response
func (h *ErrorHandler) HandleNotFound(w http.ResponseWriter, message string) {
	h.HandleError(w, http.StatusNotFound, message)
}

// HandleInternalError sends a 500 Internal Server Error response
func (h *ErrorHandler) HandleInternalError(w http.ResponseWriter, message string) {
	h.HandleError(w, http.StatusInternalServerError, message)
}
