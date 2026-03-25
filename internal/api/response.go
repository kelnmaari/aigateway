// Package api provides generic API response wrappers for type-safe HTTP responses.
package api

import (
	"github.com/gin-gonic/gin"
)

// APIResponse[T] is a generic wrapper for successful API responses with typed data.
//
// Example usage:
//
//	response := NewSuccess(user)
//	c.JSON(http.StatusOK, response)
//
// Output:
//
//	{
//	  "success": true,
//	  "data": {...user object...}
//	}
type APIResponse[T any] struct {
	Success bool   `json:"success"`
	Data    T      `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
}

// PaginatedResponse[T] extends APIResponse with pagination metadata.
//
// Example usage:
//
//	response := NewPaginated(users, PaginationMeta{
//	    Page:       1,
//	    PageSize:   20,
//	    TotalItems: 150,
//	    TotalPages: 8,
//	})
//	c.JSON(http.StatusOK, response)
type PaginatedResponse[T any] struct {
	Success    bool           `json:"success"`
	Data       []T            `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
	Message    string         `json:"message,omitempty"`
}

// PaginationMeta contains pagination metadata.
type PaginationMeta struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

// ErrorResponse represents an error response.
//
// Example usage:
//
//	response := NewError("User not found", "USER_NOT_FOUND")
//	c.JSON(http.StatusNotFound, response)
type ErrorResponse struct {
	Success bool           `json:"success"` // Always false
	Error   ErrorDetail    `json:"error"`
	Details map[string]any `json:"details,omitempty"`
}

// ErrorDetail contains error information.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// NewSuccess creates a successful response with typed data.
//
// Example:
//
//	response := NewSuccess(user)
//	// { "success": true, "data": {...} }
func NewSuccess[T any](data T) APIResponse[T] {
	return APIResponse[T]{
		Success: true,
		Data:    data,
	}
}

// NewSuccessWithMessage creates a successful response with data and message.
//
// Example:
//
//	response := NewSuccessWithMessage(user, "User created successfully")
//	// { "success": true, "data": {...}, "message": "User created successfully" }
func NewSuccessWithMessage[T any](data T, message string) APIResponse[T] {
	return APIResponse[T]{
		Success: true,
		Data:    data,
		Message: message,
	}
}

// NewPaginated creates a paginated response with typed items.
//
// Example:
//
//	response := NewPaginated(users, PaginationMeta{
//	    Page:       1,
//	    PageSize:   20,
//	    TotalItems: 150,
//	    TotalPages: 8,
//	})
func NewPaginated[T any](data []T, pagination PaginationMeta) PaginatedResponse[T] {
	if data == nil {
		data = []T{} // Return empty array instead of null
	}
	return PaginatedResponse[T]{
		Success:    true,
		Data:       data,
		Pagination: pagination,
	}
}

// NewPaginatedWithMessage creates a paginated response with message.
func NewPaginatedWithMessage[T any](data []T, pagination PaginationMeta, message string) PaginatedResponse[T] {
	response := NewPaginated(data, pagination)
	response.Message = message
	return response
}

// NewError creates an error response.
//
// Example:
//
//	response := NewError("User not found", "USER_NOT_FOUND")
//	// { "success": false, "error": {"code": "USER_NOT_FOUND", "message": "User not found"} }
func NewError(message, code string) ErrorResponse {
	return ErrorResponse{
		Success: false,
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	}
}

// NewErrorWithDetails creates an error response with additional details.
//
// Example:
//
//	response := NewErrorWithDetails("Validation failed", "VALIDATION_ERROR", map[string]interface{}{
//	    "field": "email",
//	    "reason": "invalid format",
//	})
func NewErrorWithDetails(message, code string, details map[string]any) ErrorResponse {
	return ErrorResponse{
		Success: false,
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
		Details: details,
	}
}

// RespondSuccess sends a successful response with typed data.
//
// Example:
//
//	RespondSuccess(c, http.StatusOK, user)
func RespondSuccess[T any](c *gin.Context, statusCode int, data T) {
	c.JSON(statusCode, NewSuccess(data))
}

// RespondSuccessWithMessage sends a successful response with data and message.
func RespondSuccessWithMessage[T any](c *gin.Context, statusCode int, data T, message string) {
	c.JSON(statusCode, NewSuccessWithMessage(data, message))
}

// RespondPaginated sends a paginated response with typed data.
//
// Example:
//
//	RespondPaginated(c, http.StatusOK, users, PaginationMeta{...})
func RespondPaginated[T any](c *gin.Context, statusCode int, data []T, pagination PaginationMeta) {
	c.JSON(statusCode, NewPaginated(data, pagination))
}

// RespondError sends an error response.
//
// Example:
//
//	RespondError(c, http.StatusNotFound, "User not found", "USER_NOT_FOUND")
func RespondError(c *gin.Context, statusCode int, message, code string) {
	c.JSON(statusCode, NewError(message, code))
}

// RespondErrorWithDetails sends an error response with additional details.
func RespondErrorWithDetails(c *gin.Context, statusCode int, message, code string, details map[string]any) {
	c.JSON(statusCode, NewErrorWithDetails(message, code, details))
}

// CalculatePagination calculates pagination metadata from total items, page, and page size.
//
// Example:
//
//	pagination := CalculatePagination(150, 1, 20)
//	// PaginationMeta{Page: 1, PageSize: 20, TotalItems: 150, TotalPages: 8}
func CalculatePagination(totalItems, page, pageSize int) PaginationMeta {
	totalPages := max((totalItems+pageSize-1)/pageSize, 1)

	return PaginationMeta{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}
}
