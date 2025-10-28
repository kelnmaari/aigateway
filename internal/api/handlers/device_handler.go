// Package handlers provides HTTP handlers for device management
package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/models"
	"aigateway/internal/storage"
)

// DeviceHandler handles device-related operations (Version 2.4.0+)
type DeviceHandler struct {
	db     storage.Database
	logger *logrus.Logger
}

// NewDeviceHandler creates a new DeviceHandler
func NewDeviceHandler(db storage.Database, logger *logrus.Logger) *DeviceHandler {
	return &DeviceHandler{
		db:     db,
		logger: logger,
	}
}

// RegisterDevice handles POST /api/auth/devices/register
//
// @Summary Register desktop device and get API key
// @Description Creates or returns existing API key for desktop device based on fingerprint
// @Tags Devices
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.DeviceRegistrationRequest true "Device registration details"
// @Success 201 {object} models.DeviceRegistrationResponse "New device registered"
// @Success 200 {object} models.DeviceRegistrationResponse "Existing device found"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 429 {object} ErrorResponse "Rate limit exceeded"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/auth/devices/register [post]
func (h *DeviceHandler) RegisterDevice(c *gin.Context) {
	// 1. Extract user from context (set by JWT middleware)
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		h.logger.Warn("Device registration: user_id not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, ok := userIDInterface.(string)
	if !ok || userID == "" {
		h.logger.Warn("Device registration: invalid user_id in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// 2. Parse request
	var req models.DeviceRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("Device registration: invalid request")
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request: %v", err)})
		return
	}

	// 3. Validate request
	if err := req.Validate(); err != nil {
		h.logger.WithError(err).Warn("Device registration: validation failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 4. Check rate limit (max 5 device registrations per user per day)
	// TODO: Implement rate limiting in Phase 2
	// For now, skip rate limiting

	// 5. Check if device already exists by fingerprint
	ctx := c.Request.Context()
	existingKey, err := h.db.FindAPIKeyByDeviceFingerprint(ctx, userID, req.DeviceFingerprint)
	if err != nil && err != storage.ErrNotFound {
		h.logger.WithError(err).Error("Device registration: failed to query existing device")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// 6. If device exists, return existing API key
	if existingKey != nil {
		h.logger.WithFields(logrus.Fields{
			"user_id":            userID,
			"key_id":             existingKey.ID,
			"device_fingerprint": req.DeviceFingerprint[:16],
		}).Info("Device registration: returning existing device")

		// Return existing device (without plain key)
		response := models.DeviceRegistrationResponse{
			APIKey:      fmt.Sprintf("sk-existing-%s", existingKey.ID), // Placeholder, client should already have it
			KeyID:       existingKey.ID,
			DeviceName:  safeStringValue(existingKey.DeviceName),
			DeviceOS:    safeStringValue(existingKey.DeviceOS),
			CreatedAt:   existingKey.CreatedAt,
			ExpiresAt:   existingKey.AutoExpireAt,
			LastSeenAt:  existingKey.LastSeenAt,
			IsNewDevice: false,
			Message:     "Device already registered. Using existing API key.",
		}

		c.JSON(http.StatusOK, response)
		return
	}

	// 7. Generate device name if not provided
	deviceName := req.DeviceName
	if deviceName == "" {
		deviceName = models.GenerateDeviceName(req.DeviceOS, req.DeviceHostname)
	}

	// 8. Calculate auto-expiry
	var autoExpireAt *time.Time
	expireDays := req.AutoExpireDays
	if expireDays == 0 {
		expireDays = 90 // Default: 90 days
	}
	if expireDays > 0 {
		expiry := time.Now().Add(time.Duration(expireDays) * 24 * time.Hour)
		autoExpireAt = &expiry
	}

	// 9. Create new API key with device metadata
	createReq := models.CreateAPIKeyRequest{
		Name:        deviceName,
		Description: fmt.Sprintf("Desktop device (%s, %s)", req.DeviceOS, req.DeviceHostname),
		Models:      []string{"*"}, // Full access to all models
		Permissions: []string{"chat", "models", "embeddings"},
		RateLimits: &models.RateLimits{
			RequestsPerMinute: 60,
			RequestsPerHour:   1000,
			RequestsPerDay:    10000,
			TokensPerMinute:   50000,
			TokensPerDay:      1000000,
		},
		ExpiresAt: autoExpireAt,
	}

	apiKey, plainKey, err := models.NewAPIKey(createReq)
	if err != nil {
		h.logger.WithError(err).Error("Device registration: failed to generate API key")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate API key"})
		return
	}

	// 10. Set device metadata
	apiKey.UserID = &userID
	apiKey.Scope = models.APIKeyScopePersonal
	apiKey.DeviceName = &deviceName
	apiKey.DeviceOS = &req.DeviceOS
	apiKey.DeviceHostname = &req.DeviceHostname
	apiKey.DeviceVersion = &req.DeviceVersion
	apiKey.DeviceFingerprint = &req.DeviceFingerprint
	apiKey.AutoExpireAt = autoExpireAt

	// 11. Save to database
	if err := h.db.CreateAPIKey(ctx, apiKey); err != nil {
		h.logger.WithError(err).Error("Device registration: failed to save API key")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save API key"})
		return
	}

	// 12. Log successful registration
	h.logger.WithFields(logrus.Fields{
		"event":              "device_registered",
		"user_id":            userID,
		"key_id":             apiKey.ID,
		"device_os":          req.DeviceOS,
		"device_hostname":    req.DeviceHostname,
		"device_fingerprint": req.DeviceFingerprint[:16],
		"auto_expire_days":   expireDays,
	}).Info("Device registered successfully")

	// 13. Return response with plain API key (only time it's shown!)
	response := models.DeviceRegistrationResponse{
		APIKey:      plainKey,
		KeyID:       apiKey.ID,
		DeviceName:  deviceName,
		DeviceOS:    req.DeviceOS,
		CreatedAt:   apiKey.CreatedAt,
		ExpiresAt:   autoExpireAt,
		LastSeenAt:  nil, // New device, not used yet
		IsNewDevice: true,
		Message:     "Device registered successfully",
	}

	c.JSON(http.StatusCreated, response)
}

// GenerateDeviceFingerprint generates device fingerprint for server-side validation
// This is a helper function that clients can use as reference
func GenerateDeviceFingerprint(userID, os, hostname, macAddr, cpuModel, diskSerial string) string {
	data := fmt.Sprintf("%s:%s:%s:%s:%s:%s", userID, os, hostname, macAddr, cpuModel, diskSerial)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// safeStringValue returns empty string if pointer is nil
func safeStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ListDevices handles GET /api/auth/devices
//
// @Summary List user's registered devices
// @Description Get list of all registered devices (API keys with device metadata)
// @Tags Devices
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status query string false "Status filter: active, expired, all" default(active)
// @Param sort query string false "Sort by: last_seen, created_at, name" default(last_seen)
// @Param order query string false "Sort order: asc, desc" default(desc)
// @Success 200 {object} models.ListDevicesResponse
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/auth/devices [get]
func (h *DeviceHandler) ListDevices(c *gin.Context) {
	// 1. Extract user from context (set by JWT middleware)
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		h.logger.Warn("List devices: user_id not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, ok := userIDInterface.(string)
	if !ok || userID == "" {
		h.logger.Warn("List devices: invalid user_id in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// 2. Parse query parameters
	status := c.DefaultQuery("status", "active")
	sort := c.DefaultQuery("sort", "last_seen")
	order := c.DefaultQuery("order", "desc")

	filters := models.DeviceFilters{
		Status: status,
		Sort:   sort,
		Order:  order,
	}

	// 3. Get current key ID for marking current device
	currentKeyID := ""
	if keyIDInterface, exists := c.Get("key_id"); exists {
		if keyID, ok := keyIDInterface.(string); ok {
			currentKeyID = keyID
		}
	}

	// 4. List device API keys from database
	ctx := c.Request.Context()
	keys, err := h.db.ListDeviceAPIKeys(ctx, userID, filters)
	if err != nil {
		h.logger.WithError(err).Error("List devices: failed to query device keys")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// 5. Convert to DeviceInfo
	devices := make([]models.DeviceInfo, 0, len(keys))
	activeCount := 0

	for _, key := range keys {
		deviceInfo := key.ToDeviceInfo(currentKeyID)
		devices = append(devices, deviceInfo)

		if deviceInfo.Status == "active" {
			activeCount++
		}
	}

	// 6. Return response
	response := models.ListDevicesResponse{
		Devices:      devices,
		Total:        len(devices),
		CurrentCount: activeCount,
	}

	h.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"total":   len(devices),
		"active":  activeCount,
	}).Info("Listed user devices")

	c.JSON(http.StatusOK, response)
}

// GetDevice handles GET /api/auth/devices/:id
//
// @Summary Get device details
// @Description Get detailed information about a specific device
// @Tags Devices
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Device ID (API Key ID)"
// @Success 200 {object} models.DeviceInfo
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden - device doesn't belong to user"
// @Failure 404 {object} ErrorResponse "Device not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/auth/devices/{id} [get]
func (h *DeviceHandler) GetDevice(c *gin.Context) {
	// 1. Extract user from context
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		h.logger.Warn("Get device: user_id not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, ok := userIDInterface.(string)
	if !ok || userID == "" {
		h.logger.Warn("Get device: invalid user_id in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// 2. Extract device ID from path
	deviceID := c.Param("id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Device ID is required"})
		return
	}

	// 3. Get current key ID
	currentKeyID := ""
	if keyIDInterface, exists := c.Get("key_id"); exists {
		if keyID, ok := keyIDInterface.(string); ok {
			currentKeyID = keyID
		}
	}

	// 4. Get device from database
	ctx := c.Request.Context()
	key, err := h.db.GetAPIKey(ctx, deviceID)
	if err != nil {
		if err == storage.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
			return
		}
		h.logger.WithError(err).Error("Get device: failed to query device")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// 5. Verify ownership
	if key.UserID == nil || *key.UserID != userID {
		h.logger.WithFields(logrus.Fields{
			"device_id":   deviceID,
			"user_id":     userID,
			"device_user": key.UserID,
		}).Warn("Get device: access denied - device doesn't belong to user")
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// 6. Verify it's a device key
	if !key.IsDeviceKey() {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not a device key"})
		return
	}

	// 7. Convert to DeviceInfo with usage
	deviceInfo := key.ToDeviceInfoWithUsage(currentKeyID)

	h.logger.WithFields(logrus.Fields{
		"user_id":   userID,
		"device_id": deviceID,
	}).Info("Get device details")

	c.JSON(http.StatusOK, deviceInfo)
}

// DeleteDevice handles DELETE /api/auth/devices/:id
//
// @Summary Revoke device (delete API key)
// @Description Revoke device API key. Cannot revoke current device (use logout instead)
// @Tags Devices
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Device ID (API Key ID)"
// @Param request body object false "Optional revoke reason"
// @Success 200 {object} object "Device revoked successfully"
// @Failure 400 {object} ErrorResponse "Cannot revoke current device"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden - device doesn't belong to user"
// @Failure 404 {object} ErrorResponse "Device not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/auth/devices/{id} [delete]
func (h *DeviceHandler) DeleteDevice(c *gin.Context) {
	// 1. Extract user from context
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		h.logger.Warn("Delete device: user_id not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, ok := userIDInterface.(string)
	if !ok || userID == "" {
		h.logger.Warn("Delete device: invalid user_id in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// 2. Extract device ID from path
	deviceID := c.Param("id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Device ID is required"})
		return
	}

	// 3. Get current key ID
	currentKeyID := ""
	if keyIDInterface, exists := c.Get("key_id"); exists {
		if keyID, ok := keyIDInterface.(string); ok {
			currentKeyID = keyID
		}
	}

	// 4. Prevent self-revoke
	if currentKeyID == deviceID {
		h.logger.WithFields(logrus.Fields{
			"device_id": deviceID,
			"user_id":   userID,
		}).Warn("Delete device: attempt to revoke current device")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Cannot revoke current device",
			"message": "Please use logout to revoke current device",
		})
		return
	}

	// 5. Parse request (optional reason)
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)

	if req.Reason == "" {
		req.Reason = "Revoked by user"
	}

	// 6. Get device from database (verify ownership)
	ctx := c.Request.Context()
	key, err := h.db.GetAPIKey(ctx, deviceID)
	if err != nil {
		if err == storage.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
			return
		}
		h.logger.WithError(err).Error("Delete device: failed to query device")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// 7. Verify ownership
	if key.UserID == nil || *key.UserID != userID {
		h.logger.WithFields(logrus.Fields{
			"device_id":   deviceID,
			"user_id":     userID,
			"device_user": key.UserID,
		}).Warn("Delete device: access denied - device doesn't belong to user")
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// 8. Verify it's a device key
	if !key.IsDeviceKey() {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not a device key"})
		return
	}

	// 9. Revoke the API key
	if err := h.db.RevokeAPIKey(ctx, deviceID, req.Reason); err != nil {
		h.logger.WithError(err).Error("Delete device: failed to revoke device")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// 10. Log success
	h.logger.WithFields(logrus.Fields{
		"event":       "device_revoked",
		"user_id":     userID,
		"device_id":   deviceID,
		"device_name": safeStringValue(key.DeviceName),
		"reason":      req.Reason,
	}).Info("Device revoked successfully")

	c.JSON(http.StatusOK, gin.H{
		"message":     "Device revoked successfully",
		"device_id":   deviceID,
		"device_name": safeStringValue(key.DeviceName),
	})
}

// UpdateDeviceName handles PATCH /api/auth/devices/:id
//
// @Summary Update device name
// @Description Update the user-friendly name of a device
// @Tags Devices
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Device ID (API Key ID)"
// @Param request body models.UpdateDeviceNameRequest true "New device name"
// @Success 200 {object} object "Device name updated successfully"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden - device doesn't belong to user"
// @Failure 404 {object} ErrorResponse "Device not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/auth/devices/{id} [patch]
func (h *DeviceHandler) UpdateDeviceName(c *gin.Context) {
	// 1. Extract user from context
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		h.logger.Warn("Update device name: user_id not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID, ok := userIDInterface.(string)
	if !ok || userID == "" {
		h.logger.Warn("Update device name: invalid user_id in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// 2. Extract device ID from path
	deviceID := c.Param("id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Device ID is required"})
		return
	}

	// 3. Parse request
	var req models.UpdateDeviceNameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("Update device name: invalid request")
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request: %v", err)})
		return
	}

	// 4. Update device name in database
	ctx := c.Request.Context()
	if err := h.db.UpdateDeviceName(ctx, deviceID, userID, req.DeviceName); err != nil {
		if err == storage.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Device not found or access denied"})
			return
		}
		h.logger.WithError(err).Error("Update device name: failed to update")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// 5. Log success
	h.logger.WithFields(logrus.Fields{
		"event":       "device_renamed",
		"user_id":     userID,
		"device_id":   deviceID,
		"new_name":    req.DeviceName,
	}).Info("Device name updated successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "Device name updated successfully",
		"device": gin.H{
			"id":          deviceID,
			"device_name": req.DeviceName,
		},
	})
}

