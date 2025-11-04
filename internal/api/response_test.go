package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// Test models
type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Product struct {
	SKU   string  `json:"sku"`
	Price float64 `json:"price"`
}

func TestNewSuccess(t *testing.T) {
	t.Attr("category", "api")
	t.Attr("type", "unit")
	t.Attr("go_version", "1.25")

	user := User{ID: "123", Name: "Test User"}
	response := NewSuccess(user)

	if !response.Success {
		t.Error("Expected Success to be true")
	}
	if response.Data.ID != "123" {
		t.Errorf("Expected user ID 123, got %s", response.Data.ID)
	}
	if response.Message != "" {
		t.Error("Expected empty message")
	}
}

func TestNewSuccessWithMessage(t *testing.T) {
	t.Attr("category", "api")
	t.Attr("type", "unit")

	product := Product{SKU: "ABC123", Price: 99.99}
	message := "Product created successfully"
	response := NewSuccessWithMessage(product, message)

	if !response.Success {
		t.Error("Expected Success to be true")
	}
	if response.Data.SKU != "ABC123" {
		t.Errorf("Expected SKU ABC123, got %s", response.Data.SKU)
	}
	if response.Message != message {
		t.Errorf("Expected message %q, got %q", message, response.Message)
	}
}

func TestNewPaginated(t *testing.T) {
	users := []User{
		{ID: "1", Name: "User 1"},
		{ID: "2", Name: "User 2"},
	}
	pagination := PaginationMeta{
		Page:       1,
		PageSize:   20,
		TotalItems: 100,
		TotalPages: 5,
	}

	response := NewPaginated(users, pagination)

	if !response.Success {
		t.Error("Expected Success to be true")
	}
	if len(response.Data) != 2 {
		t.Errorf("Expected 2 items, got %d", len(response.Data))
	}
	if response.Pagination.TotalItems != 100 {
		t.Errorf("Expected TotalItems 100, got %d", response.Pagination.TotalItems)
	}
}

func TestNewPaginated_NilSlice(t *testing.T) {
	var users []User // nil slice
	pagination := PaginationMeta{
		Page:       1,
		PageSize:   20,
		TotalItems: 0,
		TotalPages: 1,
	}

	response := NewPaginated(users, pagination)

	if response.Data == nil {
		t.Error("Expected empty array, got nil")
	}
	if len(response.Data) != 0 {
		t.Errorf("Expected 0 items, got %d", len(response.Data))
	}
}

func TestNewError(t *testing.T) {
	message := "User not found"
	code := "USER_NOT_FOUND"
	response := NewError(message, code)

	if response.Success {
		t.Error("Expected Success to be false")
	}
	if response.Error.Message != message {
		t.Errorf("Expected message %q, got %q", message, response.Error.Message)
	}
	if response.Error.Code != code {
		t.Errorf("Expected code %q, got %q", code, response.Error.Code)
	}
	if response.Details != nil {
		t.Error("Expected nil details")
	}
}

func TestNewErrorWithDetails(t *testing.T) {
	message := "Validation failed"
	code := "VALIDATION_ERROR"
	details := map[string]interface{}{
		"field":  "email",
		"reason": "invalid format",
	}

	response := NewErrorWithDetails(message, code, details)

	if response.Success {
		t.Error("Expected Success to be false")
	}
	if response.Error.Message != message {
		t.Errorf("Expected message %q, got %q", message, response.Error.Message)
	}
	if response.Details["field"] != "email" {
		t.Errorf("Expected field email, got %v", response.Details["field"])
	}
}

func TestCalculatePagination(t *testing.T) {
	tests := []struct {
		name       string
		totalItems int
		page       int
		pageSize   int
		wantPages  int
	}{
		{"Exact pages", 100, 1, 20, 5},
		{"With remainder", 105, 1, 20, 6},
		{"Single page", 15, 1, 20, 1},
		{"Zero items", 0, 1, 20, 1},
		{"Large dataset", 1000, 3, 50, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pagination := CalculatePagination(tt.totalItems, tt.page, tt.pageSize)
			if pagination.TotalPages != tt.wantPages {
				t.Errorf("Expected %d total pages, got %d", tt.wantPages, pagination.TotalPages)
			}
			if pagination.Page != tt.page {
				t.Errorf("Expected page %d, got %d", tt.page, pagination.Page)
			}
			if pagination.TotalItems != tt.totalItems {
				t.Errorf("Expected %d total items, got %d", tt.totalItems, pagination.TotalItems)
			}
		})
	}
}

func TestRespondSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	user := User{ID: "123", Name: "Test"}
	RespondSuccess(c, http.StatusOK, user)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response APIResponse[User]
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Error("Expected Success to be true")
	}
	if response.Data.ID != "123" {
		t.Errorf("Expected user ID 123, got %s", response.Data.ID)
	}
}

func TestRespondPaginated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	users := []User{{ID: "1", Name: "User 1"}}
	pagination := PaginationMeta{Page: 1, PageSize: 20, TotalItems: 1, TotalPages: 1}
	RespondPaginated(c, http.StatusOK, users, pagination)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response PaginatedResponse[User]
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Error("Expected Success to be true")
	}
	if len(response.Data) != 1 {
		t.Errorf("Expected 1 item, got %d", len(response.Data))
	}
}

func TestRespondError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	RespondError(c, http.StatusNotFound, "User not found", "USER_NOT_FOUND")

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Success {
		t.Error("Expected Success to be false")
	}
	if response.Error.Code != "USER_NOT_FOUND" {
		t.Errorf("Expected code USER_NOT_FOUND, got %s", response.Error.Code)
	}
}

// Benchmarks
func BenchmarkNewSuccess(b *testing.B) {
	b.Attr("category", "benchmarks")
	b.Attr("type", "performance")

	user := User{ID: "123", Name: "Test User"}
	for i := 0; i < b.N; i++ {
		_ = NewSuccess(user)
	}
}

func BenchmarkNewPaginated(b *testing.B) {
	users := []User{{ID: "1", Name: "User 1"}}
	pagination := PaginationMeta{Page: 1, PageSize: 20, TotalItems: 1, TotalPages: 1}
	for i := 0; i < b.N; i++ {
		_ = NewPaginated(users, pagination)
	}
}

func BenchmarkCalculatePagination(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = CalculatePagination(1000, 3, 50)
	}
}
