package handlers

import (
    "encoding/json"
    "net/http"
)

// ResponseHandler handles all successful responses
type ResponseHandler struct{}

// Response represents the standard API response structure
type Response struct {
    Status  int         `json:"status"`
    Message string      `json:"message,omitempty"`
    Data    interface{} `json:"data,omitempty"`
}

// PaginatedResponse represents a response with pagination information
type PaginatedResponse struct {
    Response
    Pagination *PaginationInfo `json:"pagination,omitempty"`
}

// PaginationInfo contains pagination metadata
type PaginationInfo struct {
    CurrentPage  int `json:"current_page"`
    TotalPages   int `json:"total_pages"`
    ItemsPerPage int `json:"items_per_page"`
    TotalItems   int `json:"total_items"`
}

// NewResponseHandler creates a new response handler
func NewResponseHandler() *ResponseHandler {
    return &ResponseHandler{}
}

// JSON sends a JSON response
func (h *ResponseHandler) JSON(w http.ResponseWriter, code int, payload interface{}) {
    response, err := json.Marshal(payload)
    if err != nil {
        NewErrorHandler().HandleInternalError(w, "Error processing response")
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    w.Write(response)
}

// Success sends a success response with status 200
func (h *ResponseHandler) Success(w http.ResponseWriter, message string, data interface{}) {
    h.JSON(w, http.StatusOK, Response{
        Status:  http.StatusOK,
        Message: message,
        Data:    data,
    })
}

// Created sends a success response with status 201
func (h *ResponseHandler) Created(w http.ResponseWriter, message string, data interface{}) {
    h.JSON(w, http.StatusCreated, Response{
        Status:  http.StatusCreated,
        Message: message,
        Data:    data,
    })
}

// Paginated sends a paginated response
func (h *ResponseHandler) Paginated(w http.ResponseWriter, message string, data interface{}, page, limit, total int) {
    totalPages := (total + limit - 1) / limit // Ceiling division

    h.JSON(w, http.StatusOK, PaginatedResponse{
        Response: Response{
            Status:  http.StatusOK,
            Message: message,
            Data:    data,
        },
        Pagination: &PaginationInfo{
            CurrentPage:  page,
            TotalPages:   totalPages,
            ItemsPerPage: limit,
            TotalItems:   total,
        },
    })
}