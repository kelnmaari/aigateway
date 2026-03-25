//go:build !windows

// Package metrics provides GPU monitoring using nvidia-smi (no CGO required)
package metrics

import (
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// GPUMonitor отслеживает метрики NVIDIA GPU через nvidia-smi
type GPUMonitor struct {
	logger        *logrus.Logger
	metricsLogger *logrus.Logger // Отдельный логгер для метрик (в metrics.log)
	mu            sync.RWMutex
	enabled       bool
	interval      time.Duration
	stopChan      chan struct{}
}

// SetMetricsLogger устанавливает отдельный логгер для метрик
func (m *GPUMonitor) SetMetricsLogger(logger *logrus.Logger) {
	if m != nil {
		m.metricsLogger = logger
	}
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

// NewGPUMonitor создает новый GPU монитор
func NewGPUMonitor(logger *logrus.Logger, collectionInterval time.Duration) (*GPUMonitor, error) {
	monitor := &GPUMonitor{
		logger:   logger,
		enabled:  false,
		interval: collectionInterval,
		stopChan: make(chan struct{}),
	}

	// Проверяем доступность nvidia-smi
	if err := exec.Command("nvidia-smi", "--version").Run(); err != nil {
		logger.Info("nvidia-smi not found - GPU monitoring disabled")
		return monitor, nil
	}

	// Проверяем что есть GPU
	metrics, err := monitor.queryNvidiaSMI()
	if err != nil {
		logger.WithError(err).Warn("Failed to query nvidia-smi - GPU monitoring disabled")
		return monitor, nil
	}

	if len(metrics.Devices) == 0 {
		logger.Info("No NVIDIA GPUs found - GPU monitoring disabled")
		return monitor, nil
	}

	monitor.enabled = true
	logger.WithField("device_count", len(metrics.Devices)).Info("✅ GPU Monitor initialized successfully (nvidia-smi)")

	return monitor, nil
}

// Start запускает мониторинг GPU
func (m *GPUMonitor) Start() {
	if !m.enabled {
		m.logger.Info("GPU monitoring is disabled")
		return
	}

	go m.collectLoop()
	m.logger.Info("GPU Monitor started")
}

// Stop останавливает мониторинг
func (m *GPUMonitor) Stop() {
	if !m.enabled {
		return
	}

	close(m.stopChan)
	m.logger.Info("GPU Monitor stopped")
}

// GetMetrics возвращает текущие метрики GPU
func (m *GPUMonitor) GetMetrics() (*GPUMetrics, error) {
	if !m.enabled {
		return &GPUMetrics{DeviceCount: 0}, nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.queryNvidiaSMI()
}

// queryNvidiaSMI выполняет nvidia-smi и парсит результат
func (m *GPUMonitor) queryNvidiaSMI() (*GPUMetrics, error) {
	// Запускаем nvidia-smi с XML выводом для надежного парсинга
	cmd := exec.Command("nvidia-smi", "-q", "-x")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute nvidia-smi: %w", err)
	}

	// Используем простой парсинг вместо полноценного XML для минимизации зависимостей
	metrics := &GPUMetrics{
		Devices: []DeviceMetrics{},
	}

	// Для простоты используем CSV формат
	cmd = exec.Command("nvidia-smi",
		"--query-gpu=index,name,uuid,temperature.gpu,power.draw,power.limit,utilization.gpu,utilization.memory,memory.total,memory.used,memory.free,fan.speed,clocks.current.graphics,clocks.current.memory",
		"--format=csv,noheader,nounits")

	output, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute nvidia-smi: %w", err)
	}

	lines := splitLines(string(output))

	// Используем metricsLogger для debug логов если настроен
	debugLogger := m.metricsLogger
	if debugLogger == nil {
		debugLogger = m.logger
	}

	debugLogger.WithFields(logrus.Fields{
		"output_length": len(output),
		"lines_count":   len(lines),
	}).Debug("nvidia-smi output received")

	for _, line := range lines {
		if line == "" || len(line) < 10 {
			continue
		}

		fields := splitCSV(line)

		debugLogger.WithFields(logrus.Fields{
			"fields_count": len(fields),
			"raw_line":     line,
		}).Debug("Parsing nvidia-smi line")

		if len(fields) < 14 {
			m.logger.WithField("fields_count", len(fields)).Warn("Insufficient fields in nvidia-smi output")
			continue
		}

		device := DeviceMetrics{}
		fmt.Sscanf(fields[0], "%d", &device.Index)
		device.Name = trimSpace(fields[1])
		device.UUID = trimSpace(fields[2])

		var temp, utilGPU, utilMem, fan, gfxClock, memClock int
		var power, powerLim, memTot, memUse, memFr float64

		fmt.Sscanf(fields[3], "%d", &temp)
		fmt.Sscanf(fields[4], "%f", &power)
		fmt.Sscanf(fields[5], "%f", &powerLim)
		fmt.Sscanf(fields[6], "%d", &utilGPU)
		fmt.Sscanf(fields[7], "%d", &utilMem)
		fmt.Sscanf(fields[8], "%f", &memTot)
		fmt.Sscanf(fields[9], "%f", &memUse)
		fmt.Sscanf(fields[10], "%f", &memFr)
		fmt.Sscanf(fields[11], "%d", &fan)
		fmt.Sscanf(fields[12], "%d", &gfxClock)
		fmt.Sscanf(fields[13], "%d", &memClock)

		device.TemperatureC = uint32(temp)
		device.PowerUsageW = power
		device.PowerLimitW = powerLim
		device.UtilizationGPU = uint32(utilGPU)
		device.UtilizationMem = uint32(utilMem)
		device.MemoryTotalMB = memTot
		device.MemoryUsedMB = memUse
		device.MemoryFreeMB = memFr
		device.FanSpeedPercent = uint32(fan)
		device.ClockGraphicsMHz = uint32(gfxClock)
		device.ClockMemoryMHz = uint32(memClock)

		if memTot > 0 {
			device.MemoryUsage = (memUse / memTot) * 100
		}

		debugLogger.WithFields(logrus.Fields{
			"index":        device.Index,
			"name":         device.Name,
			"temp":         device.TemperatureC,
			"power":        device.PowerUsageW,
			"gpu_util":     device.UtilizationGPU,
			"mem_util":     device.UtilizationMem,
			"mem_used_mb":  device.MemoryUsedMB,
			"mem_total_mb": device.MemoryTotalMB,
		}).Debug("GPU device parsed")

		metrics.Devices = append(metrics.Devices, device)
		metrics.TotalMemoryMB += memTot
		metrics.UsedMemoryMB += memUse
	}

	metrics.DeviceCount = len(metrics.Devices)
	if metrics.TotalMemoryMB > 0 {
		metrics.MemoryUsage = (metrics.UsedMemoryMB / metrics.TotalMemoryMB) * 100
	}

	return metrics, nil
}

// collectLoop периодически собирает метрики
func (m *GPUMonitor) collectLoop() {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			metrics, err := m.GetMetrics()
			if err != nil {
				m.logger.WithError(err).Error("Failed to collect GPU metrics")
				continue
			}

			// Логируем метрики в отдельный файл (если настроен) или в основной
			logTarget := m.metricsLogger
			if logTarget == nil {
				logTarget = m.logger
			}
			logTarget.WithFields(logrus.Fields{
				"device_count":    metrics.DeviceCount,
				"total_memory_mb": fmt.Sprintf("%.2f", metrics.TotalMemoryMB),
				"used_memory_mb":  fmt.Sprintf("%.2f", metrics.UsedMemoryMB),
				"memory_usage":    fmt.Sprintf("%.2f%%", metrics.MemoryUsage),
			}).Debug("GPU metrics collected")

		case <-m.stopChan:
			return
		}
	}
}

// splitLines splits string by newlines
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// splitCSV splits CSV line by commas
func splitCSV(line string) []string {
	var fields []string
	start := 0
	for i, c := range line {
		if c == ',' {
			fields = append(fields, line[start:i])
			start = i + 1
		}
	}
	if start < len(line) {
		fields = append(fields, line[start:])
	}
	return fields
}

// trimSpace removes leading/trailing whitespace
func trimSpace(s string) string {
	start := 0
	end := len(s)

	// Trim leading spaces
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}

	// Trim trailing spaces
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\r' || s[end-1] == '\n') {
		end--
	}

	return s[start:end]
}
