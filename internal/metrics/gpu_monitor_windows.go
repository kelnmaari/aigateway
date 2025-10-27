//go:build windows
// +build windows

// Package metrics provides GPU monitoring stub for Windows
package metrics

import (
	"time"

	"github.com/sirupsen/logrus"
)

// GPUMonitor заглушка для Windows (NVML недоступен)
type GPUMonitor struct {
	logger  *logrus.Logger
	enabled bool
}

// GPUMetrics содержит метрики GPU
type GPUMetrics struct {
	DeviceCount   int             `json:"device_count"`
	Devices       []DeviceMetrics `json:"devices"`
	TotalMemoryMB float64         `json:"total_memory_mb"`
	UsedMemoryMB  float64         `json:"used_memory_mb"`
	MemoryUsage   float64         `json:"memory_usage_percent"`
}

// DeviceMetrics содержит метрики отдельного GPU
type DeviceMetrics struct {
	Index            int     `json:"index"`
	Name             string  `json:"name"`
	UUID             string  `json:"uuid"`
	TemperatureC     uint32  `json:"temperature_c"`
	PowerUsageW      float64 `json:"power_usage_w"`
	PowerLimitW      float64 `json:"power_limit_w"`
	UtilizationGPU   uint32  `json:"utilization_gpu_percent"`
	UtilizationMem   uint32  `json:"utilization_memory_percent"`
	MemoryTotalMB    float64 `json:"memory_total_mb"`
	MemoryUsedMB     float64 `json:"memory_used_mb"`
	MemoryFreeMB     float64 `json:"memory_free_mb"`
	MemoryUsage      float64 `json:"memory_usage_percent"`
	FanSpeedPercent  uint32  `json:"fan_speed_percent"`
	ClockGraphicsMHz uint32  `json:"clock_graphics_mhz"`
	ClockMemoryMHz   uint32  `json:"clock_memory_mhz"`
}

// NewGPUMonitor создает заглушку GPU монитора для Windows
func NewGPUMonitor(logger *logrus.Logger, collectionInterval time.Duration) (*GPUMonitor, error) {
	logger.Info("GPU monitoring is not supported on Windows (NVML requires Linux/Unix)")
	return &GPUMonitor{
		logger:  logger,
		enabled: false,
	}, nil
}

// Start ничего не делает на Windows
func (m *GPUMonitor) Start() {
	// No-op on Windows
}

// Stop ничего не делает на Windows
func (m *GPUMonitor) Stop() {
	// No-op on Windows
}

// GetMetrics возвращает пустые метрики на Windows
func (m *GPUMonitor) GetMetrics() (*GPUMetrics, error) {
	return &GPUMetrics{
		DeviceCount: 0,
		Devices:     []DeviceMetrics{},
	}, nil
}


