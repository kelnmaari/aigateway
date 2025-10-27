package models

// ChangePasswordRequest запрос на смену пароля
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required,min=8"`
	NewPassword     string `json:"new_password" binding:"required,min=8,max=100"`
}

// ChangePasswordResponse ответ на смену пароля
type ChangePasswordResponse struct {
	Message string `json:"message"`
}

