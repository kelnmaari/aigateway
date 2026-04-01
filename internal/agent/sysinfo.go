package agent

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

// collectGPUDevices queries nvidia-smi for GPU info.
// Returns empty slice on CPU-only nodes or when nvidia-smi is unavailable.
func collectGPUDevices(logger *logrus.Logger) []GPUDevice {
	cmd := exec.Command("nvidia-smi",
		"--query-gpu=index,name,memory.total,memory.used,memory.free,utilization.gpu,temperature.gpu,power.draw,power.limit",
		"--format=csv,noheader,nounits",
	)
	out, err := cmd.Output()
	if err != nil {
		// nvidia-smi not available — CPU-only node
		return nil
	}

	var devices []GPUDevice
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, ", ")
		if len(fields) < 7 {
			continue
		}
		idx, _ := strconv.Atoi(strings.TrimSpace(fields[0]))
		memTotal, _ := strconv.ParseFloat(strings.TrimSpace(fields[2]), 64)
		memUsed, _ := strconv.ParseFloat(strings.TrimSpace(fields[3]), 64)
		memFree, _ := strconv.ParseFloat(strings.TrimSpace(fields[4]), 64)
		utilGPU, _ := strconv.ParseUint(strings.TrimSpace(fields[5]), 10, 32)
		tempC, _ := strconv.ParseUint(strings.TrimSpace(fields[6]), 10, 32)

		dev := GPUDevice{
			Index:          idx,
			Name:           strings.TrimSpace(fields[1]),
			MemoryTotalMB:  memTotal,
			MemoryUsedMB:   memUsed,
			MemoryFreeMB:   memFree,
			UtilizationGPU: uint32(utilGPU),
			TemperatureC:   uint32(tempC),
		}
		if len(fields) >= 9 {
			dev.PowerDrawW, _ = strconv.ParseFloat(strings.TrimSpace(fields[7]), 64)
			dev.PowerLimitW, _ = strconv.ParseFloat(strings.TrimSpace(fields[8]), 64)
		}
		devices = append(devices, dev)
	}
	return devices
}

// collectCPUInfo returns basic CPU information.
func collectCPUInfo() *CPUInfo {
	info := &CPUInfo{
		Cores:   runtime.NumCPU(),
		Threads: runtime.NumCPU(),
	}

	// Try to get CPU model from lscpu on Linux
	if runtime.GOOS == "linux" {
		cmd := exec.Command("lscpu")
		out, err := cmd.Output()
		if err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				if strings.HasPrefix(line, "Model name:") {
					info.Model = strings.TrimSpace(strings.TrimPrefix(line, "Model name:"))
				}
				if strings.HasPrefix(line, "Thread(s) per core:") {
					threads, _ := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Thread(s) per core:")))
					if threads > 0 {
						info.Threads = info.Cores * threads
					}
				}
			}
		}
	}

	return info
}

// collectMemoryInfo returns system RAM info.
func collectMemoryInfo() *MemoryInfo {
	if runtime.GOOS != "linux" {
		return nil
	}

	cmd := exec.Command("free", "-m")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "Mem:") {
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				total, _ := strconv.ParseFloat(fields[1], 64)
				used, _ := strconv.ParseFloat(fields[2], 64)
				free, _ := strconv.ParseFloat(fields[3], 64)
				return &MemoryInfo{
					TotalMB: total,
					UsedMB:  used,
					FreeMB:  free,
				}
			}
		}
	}
	return nil
}
