package inference

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/docker/go-units"
	"github.com/sirupsen/logrus"
)

// DockerRuntime starts/stops provider containers via Docker API if reachable, otherwise docker CLI.
// Notes:
// - Uses host networking via explicit -p mappings (one container = one model).
type DockerRuntime struct {
	dockerBin string
	logger    *logrus.Logger
	logsDir   string // directory for container log files

	mu      sync.Mutex
	handles map[string]*ContainerHandle // containerName -> handle

	useAPI bool
	api    *client.Client

	// Log streaming goroutine cancellation
	logCancels map[string]context.CancelFunc
}

// DockerRuntimeConfig holds runtime options.
type DockerRuntimeConfig struct {
	DockerBin string
	Logger    *logrus.Logger
	LogsDir   string // directory where container logs will be written
}

// NewDockerRuntime creates DockerRuntime with defaults.
func NewDockerRuntime(cfg DockerRuntimeConfig) *DockerRuntime {
	bin := cfg.DockerBin
	if bin == "" {
		bin = "docker"
	}
	if cfg.Logger == nil {
		cfg.Logger = logrus.New()
	}
	logsDir := cfg.LogsDir
	if logsDir == "" {
		logsDir = "logs/containers"
	}
	// Ensure logs directory exists
	_ = os.MkdirAll(logsDir, 0755)

	r := &DockerRuntime{
		dockerBin:  bin,
		logger:     cfg.Logger,
		logsDir:    logsDir,
		handles:    make(map[string]*ContainerHandle),
		logCancels: make(map[string]context.CancelFunc),
	}

	if cli, err := newDockerAPIClient(); err == nil {
		if _, err := cli.Ping(context.Background()); err == nil {
			r.useAPI = true
			r.api = cli
			r.logger.Info("DockerRuntime: using Docker API (socket)")
		} else {
			r.logger.WithError(err).Warn("DockerRuntime: Docker API ping failed, fallback to CLI")
		}
	} else {
		r.logger.WithError(err).Warn("DockerRuntime: Docker API client init failed, fallback to CLI")
	}

	return r
}

// Start launches a container and returns a handle with mapped endpoint.
func (r *DockerRuntime) Start(ctx context.Context, req ContainerStartRequest) (*ContainerHandle, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	containerName := fmt.Sprintf("aigw-%s-%d", sanitize(req.ModelAlias), time.Now().UnixNano())

	hostPorts := make(map[string]int, len(req.Ports))
	for name, cport := range req.Ports {
		hp, err := pickFreePort()
		if err != nil {
			return nil, fmt.Errorf("allocate port for %s: %w", name, err)
		}
		hostPorts[name] = hp
		req.Ports[name] = cport
	}

	// Use background context for Docker operations to prevent cancellation from HTTP request
	pullCtx, pullCancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer pullCancel()

	if r.useAPI && r.api != nil {
		if err := r.pullImageAPI(pullCtx, req.Image); err != nil {
			return nil, err
		}
		return r.startAPI(ctx, containerName, req, hostPorts)
	}
	if err := r.pullImageCLI(pullCtx, req.Image); err != nil {
		return nil, err
	}
	return r.startCLI(ctx, containerName, req, hostPorts)
}

func (r *DockerRuntime) startAPI(ctx context.Context, containerName string, req ContainerStartRequest, hostPorts map[string]int) (*ContainerHandle, error) {
	// Use independent context for Docker operations to avoid cancellation from HTTP request
	// Container creation should complete even if client disconnects
	dockerCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	_ = ctx // original context preserved for future use if needed

	portBindings := nat.PortMap{}
	exposed := nat.PortSet{}
	for name, cport := range req.Ports {
		port := nat.Port(fmt.Sprintf("%d/tcp", cport))
		exposed[port] = struct{}{}
		portBindings[port] = []nat.PortBinding{{
			HostIP:   "127.0.0.1", // bind to localhost only for security
			HostPort: fmt.Sprintf("%d", hostPorts[name]),
		}}
	}

	var mounts []mount.Mount
	for _, m := range req.Mounts {
		mounts = append(mounts, mount.Mount{
			Type:     mount.TypeBind,
			Source:   m.HostPath,
			Target:   m.ContainerPath,
			ReadOnly: m.ReadOnly,
		})
	}

	env := make([]string, 0, len(req.Env))
	for k, v := range req.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// Build GPU device request - specific device(s) or all
	gpuRequest := container.DeviceRequest{
		Capabilities: [][]string{{"gpu"}},
	}
	if req.GPUDevice != "" {
		// Use specific GPU(s), e.g., "0" or "0,1"
		gpuRequest.DeviceIDs = strings.Split(req.GPUDevice, ",")
	} else {
		// Use all GPUs
		gpuRequest.Count = -1
	}

	// Build host config with GPU and multi-GPU support
	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		Mounts:       mounts,
		AutoRemove:   true,
		Resources: container.Resources{
			DeviceRequests: []container.DeviceRequest{gpuRequest},
		},
	}

	// For multi-GPU setups (tensor parallel > 1), add NCCL requirements
	if strings.Contains(req.GPUDevice, ",") || req.GPUDevice == "" {
		// IPC host mode required for NCCL inter-GPU communication
		hostConfig.IpcMode = "host"
		// Increase shared memory for NCCL (16GB)
		hostConfig.ShmSize = 16 * 1024 * 1024 * 1024
		// Remove memlock limits for NCCL
		hostConfig.Ulimits = []*units.Ulimit{
			{Name: "memlock", Soft: -1, Hard: -1},
		}
	}

	resp, err := r.api.ContainerCreate(dockerCtx,
		&container.Config{
			Image:        req.Image,
			Cmd:          req.Command,
			Env:          env,
			ExposedPorts: exposed,
		},
		hostConfig,
		nil, nil, containerName,
	)
	if err != nil {
		return nil, fmt.Errorf("docker api create: %w", err)
	}

	if err := r.api.ContainerStart(dockerCtx, resp.ID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("docker api start: %w", err)
	}

	endpoint := ""
	if len(hostPorts) > 0 {
		keys := make([]string, 0, len(hostPorts))
		for k := range hostPorts {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		endpoint = fmt.Sprintf("http://127.0.0.1:%d", hostPorts[keys[0]])
	}

	handle := &ContainerHandle{
		ID:         resp.ID,
		Provider:   req.Provider,
		ModelAlias: req.ModelAlias,
		Endpoint:   endpoint,
	}
	r.handles[containerName] = handle

	// Start streaming logs to file
	r.startLogStreaming(req.ModelAlias, resp.ID)

	return handle, nil
}

func (r *DockerRuntime) startCLI(ctx context.Context, containerName string, req ContainerStartRequest, hostPorts map[string]int) (*ContainerHandle, error) {
	// Use independent context for Docker operations to avoid cancellation from HTTP request
	dockerCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	_ = ctx // original context preserved for future use if needed

	args := []string{"run", "-d", "--rm", "--name", containerName}

	// GPU access via --gpus all + NVIDIA_VISIBLE_DEVICES for specific devices
	// This avoids the "cannot set both Count and DeviceIDs" error
	isMultiGPU := strings.Contains(req.GPUDevice, ",") || req.GPUDevice == ""
	args = append(args, "--gpus", "all")
	if req.GPUDevice != "" {
		// Restrict to specific GPUs via environment variable
		args = append(args, "-e", fmt.Sprintf("NVIDIA_VISIBLE_DEVICES=%s", req.GPUDevice))
	}

	// For multi-GPU setups, add NCCL requirements
	if isMultiGPU {
		args = append(args, "--ipc=host")                // Required for NCCL inter-GPU communication
		args = append(args, "--shm-size=16g")            // Increase shared memory for NCCL
		args = append(args, "--ulimit", "memlock=-1:-1") // Remove memlock limits
	}

	for name, cport := range req.Ports {
		hp := hostPorts[name]
		args = append(args, "-p", fmt.Sprintf("127.0.0.1:%d:%d", hp, cport)) // bind to localhost only
	}

	for k, v := range req.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	for _, m := range req.Mounts {
		mode := "rw"
		if m.ReadOnly {
			mode = "ro"
		}
		args = append(args, "-v", fmt.Sprintf("%s:%s:%s", m.HostPath, m.ContainerPath, mode))
	}

	args = append(args, req.Image)
	args = append(args, req.Command...)

	cmd := exec.CommandContext(dockerCtx, r.dockerBin, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker run failed: %w: %s", err, string(out))
	}

	endpoint := ""
	if len(hostPorts) > 0 {
		keys := make([]string, 0, len(hostPorts))
		for k := range hostPorts {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		endpoint = fmt.Sprintf("http://127.0.0.1:%d", hostPorts[keys[0]])
	}

	handle := &ContainerHandle{
		ID:         strings.TrimSpace(string(out)),
		Provider:   req.Provider,
		ModelAlias: req.ModelAlias,
		Endpoint:   endpoint,
	}
	r.handles[containerName] = handle

	// Start streaming logs to file
	r.startLogStreaming(req.ModelAlias, handle.ID)

	return handle, nil
}

// Stop removes the container by ID.
func (r *DockerRuntime) Stop(ctx context.Context, handleID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Stop log streaming for this container
	r.stopLogStreaming(handleID)

	if r.useAPI && r.api != nil {
		if err := r.api.ContainerRemove(ctx, handleID, container.RemoveOptions{Force: true}); err != nil {
			// Ignore "no such container" errors - container already dead
			if !strings.Contains(err.Error(), "No such container") &&
				!strings.Contains(err.Error(), "not found") {
				return fmt.Errorf("docker api rm failed: %w", err)
			}
			r.logger.WithField("container", handleID).Debug("Container already removed, ignoring")
		}
		return nil
	}

	cmd := exec.CommandContext(ctx, r.dockerBin, "rm", "-f", handleID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Ignore "no such container" errors - container already dead
		outStr := string(out)
		if !strings.Contains(outStr, "No such container") &&
			!strings.Contains(outStr, "not found") {
			return fmt.Errorf("docker rm failed: %w: %s", err, outStr)
		}
		r.logger.WithField("container", handleID).Debug("Container already removed, ignoring")
	}
	return nil
}

func pickFreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	addr := l.Addr().(*net.TCPAddr)
	return addr.Port, nil
}

func sanitize(alias string) string {
	alias = strings.ReplaceAll(alias, "/", "-")
	alias = strings.ReplaceAll(alias, " ", "-")
	return alias
}

func defaultDockerHost() string {
	if runtime.GOOS == "windows" {
		return "npipe:////./pipe/docker_engine"
	}
	return "unix:///var/run/docker.sock"
}

func newDockerAPIClient() (*client.Client, error) {
	host := defaultDockerHost()
	// Note: assumes socket permissions are restricted to trusted users.
	return client.NewClientWithOpts(client.WithHost(host), client.WithAPIVersionNegotiation())
}

func (r *DockerRuntime) pullImageAPI(ctx context.Context, image string) error {
	if image == "" {
		return fmt.Errorf("image is empty")
	}
	r.logger.WithField("image", image).Info("Pulling Docker image...")
	out, err := r.api.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("docker api pull: %w", err)
	}
	defer out.Close()
	// Must read the entire response to wait for pull completion
	buf := make([]byte, 8192)
	for {
		_, readErr := out.Read(buf)
		if readErr != nil {
			break
		}
	}
	r.logger.WithField("image", image).Info("Docker image pulled successfully")
	return nil
}

func (r *DockerRuntime) pullImageCLI(ctx context.Context, image string) error {
	if image == "" {
		return fmt.Errorf("image is empty")
	}
	r.logger.WithField("image", image).Info("Pulling Docker image (CLI)...")
	cmd := exec.CommandContext(ctx, r.dockerBin, "pull", image)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("docker pull failed: %w: %s", err, string(out))
	}
	r.logger.WithField("image", image).Info("Docker image pulled successfully")
	return nil
}

// Logs returns recent stdout/stderr logs from container.
func (r *DockerRuntime) Logs(ctx context.Context, handleID string, tailLines int) (string, error) {
	if tailLines <= 0 {
		tailLines = 100
	}
	if r.useAPI && r.api != nil {
		opts := container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Tail:       fmt.Sprintf("%d", tailLines),
		}
		rc, err := r.api.ContainerLogs(ctx, handleID, opts)
		if err != nil {
			return "", fmt.Errorf("docker api logs: %w", err)
		}
		defer rc.Close()
		// Read all logs, not just first 64KB
		data, err := io.ReadAll(rc)
		if err != nil && err != io.EOF {
			return "", fmt.Errorf("docker api logs read: %w", err)
		}
		// Docker multiplexed stream has 8-byte header per frame
		// Clean it up for display
		return demuxDockerLogs(data), nil
	}
	// CLI fallback
	cmd := exec.CommandContext(ctx, r.dockerBin, "logs", "--tail", fmt.Sprintf("%d", tailLines), handleID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker logs failed: %w: %s", err, string(out))
	}
	return string(out), nil
}

// demuxDockerLogs removes Docker multiplexed stream headers.
// Docker logs API returns multiplexed stdout/stderr with 8-byte headers.
func demuxDockerLogs(data []byte) string {
	var result strings.Builder
	for len(data) >= 8 {
		// Header: [stream_type(1), 0, 0, 0, size(4 big-endian)]
		size := int(data[4])<<24 | int(data[5])<<16 | int(data[6])<<8 | int(data[7])
		if size <= 0 || len(data) < 8+size {
			break
		}
		result.Write(data[8 : 8+size])
		data = data[8+size:]
	}
	// If demux failed (e.g., tty mode), return as-is
	if result.Len() == 0 && len(data) > 0 {
		return string(data)
	}
	return result.String()
}

// startLogStreaming starts a goroutine that streams container logs to a file.
func (r *DockerRuntime) startLogStreaming(alias, containerID string) {
	if r.logsDir == "" {
		return
	}

	// Create log file
	logPath := filepath.Join(r.logsDir, alias+".log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		r.logger.WithError(err).WithField("path", logPath).Error("Failed to create container log file")
		return
	}

	// Write header
	fmt.Fprintf(f, "=== Container logs for %s (container: %s) ===\n", alias, containerID)
	fmt.Fprintf(f, "=== Started: %s ===\n\n", time.Now().Format(time.RFC3339))

	ctx, cancel := context.WithCancel(context.Background())
	r.logCancels[containerID] = cancel

	go func() {
		defer f.Close()
		defer func() {
			fmt.Fprintf(f, "\n=== Log streaming ended: %s ===\n", time.Now().Format(time.RFC3339))
		}()

		if r.useAPI && r.api != nil {
			r.streamLogsAPI(ctx, containerID, f)
		} else {
			r.streamLogsCLI(ctx, containerID, f)
		}
	}()

	r.logger.WithFields(logrus.Fields{
		"alias":     alias,
		"container": containerID,
		"log_file":  logPath,
	}).Info("Started container log streaming")
}

// stopLogStreaming cancels log streaming for a container.
func (r *DockerRuntime) stopLogStreaming(containerID string) {
	if cancel, ok := r.logCancels[containerID]; ok {
		cancel()
		delete(r.logCancels, containerID)
	}
}

// streamLogsAPI streams logs using Docker API.
func (r *DockerRuntime) streamLogsAPI(ctx context.Context, containerID string, w io.Writer) {
	opts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: true,
	}
	rc, err := r.api.ContainerLogs(ctx, containerID, opts)
	if err != nil {
		fmt.Fprintf(w, "ERROR: Failed to get container logs: %v\n", err)
		return
	}
	defer rc.Close()

	// Docker multiplexes stdout/stderr with 8-byte header:
	// [0]: stream type (1=stdout, 2=stderr)
	// [1-3]: reserved
	// [4-7]: payload size (big endian uint32)
	header := make([]byte, 8)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Read 8-byte header
		_, err := io.ReadFull(rc, header)
		if err != nil {
			if err != io.EOF && ctx.Err() == nil {
				fmt.Fprintf(w, "ERROR: Log read error: %v\n", err)
			}
			return
		}

		// Parse payload size from header[4:8] (big endian)
		size := uint32(header[4])<<24 | uint32(header[5])<<16 | uint32(header[6])<<8 | uint32(header[7])
		if size == 0 {
			continue
		}

		// Read payload
		payload := make([]byte, size)
		_, err = io.ReadFull(rc, payload)
		if err != nil {
			if err != io.EOF && ctx.Err() == nil {
				fmt.Fprintf(w, "ERROR: Log read error: %v\n", err)
			}
			return
		}

		// Write clean payload
		w.Write(payload)

		// Flush
		if f, ok := w.(*os.File); ok {
			f.Sync()
		}
	}
}

// streamLogsCLI streams logs using docker CLI.
func (r *DockerRuntime) streamLogsCLI(ctx context.Context, containerID string, w io.Writer) {
	cmd := exec.CommandContext(ctx, r.dockerBin, "logs", "-f", "--timestamps", containerID)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(w, "ERROR: Failed to get stdout pipe: %v\n", err)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Fprintf(w, "ERROR: Failed to get stderr pipe: %v\n", err)
		return
	}

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(w, "ERROR: Failed to start docker logs: %v\n", err)
		return
	}

	// Stream both stdout and stderr
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			fmt.Fprintln(w, scanner.Text())
		}
	}()

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Fprintln(w, "[stderr] "+scanner.Text())
		}
	}()

	wg.Wait()
	cmd.Wait()
}

// ImageExists checks if a Docker image exists locally.
// Returns (exists, size) where size is human-readable (e.g., "2.5GB").
func (r *DockerRuntime) ImageExists(image string) (bool, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try Docker API first if enabled
	if r.useAPI && r.api != nil {
		inspect, _, err := r.api.ImageInspectWithRaw(ctx, image)
		if err != nil {
			return false, ""
		}
		return true, formatSize(inspect.Size)
	}

	// Fallback to CLI
	return r.imageExistsCLI(image)
}

func (r *DockerRuntime) imageExistsCLI(image string) (bool, string) {
	cmd := exec.Command(r.dockerBin, "image", "inspect", image, "--format", "{{.Size}}")
	output, err := cmd.Output()
	if err != nil {
		return false, ""
	}

	// Parse size
	sizeStr := strings.TrimSpace(string(output))
	if sizeBytes, err := strconv.ParseInt(sizeStr, 10, 64); err == nil {
		return true, formatSize(sizeBytes)
	}
	return true, ""
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1fGB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1fMB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1fKB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

// PullImage pulls a Docker image. This is a blocking operation.
func (r *DockerRuntime) PullImage(image string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if r.useAPI && r.api != nil {
		return r.pullImageAPI(ctx, image)
	}
	return r.pullImageCLI(ctx, image)
}

// DiscoveredContainer holds info about a discovered running container.
type DiscoveredContainer struct {
	ID         string
	Name       string
	Image      string
	ModelAlias string
	Provider   ProviderKind
	Endpoint   string
	Status     string
	CreatedAt  time.Time
}

// DiscoverRunningContainers finds already running inference containers with "aigw-" prefix.
// This allows the server to recover state after restart.
func (r *DockerRuntime) DiscoverRunningContainers(ctx context.Context) ([]DiscoveredContainer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.useAPI && r.api != nil {
		return r.discoverContainersAPI(ctx)
	}
	return r.discoverContainersCLI(ctx)
}

func (r *DockerRuntime) discoverContainersAPI(ctx context.Context) ([]DiscoveredContainer, error) {
	containers, err := r.api.ContainerList(ctx, container.ListOptions{
		All: false, // Only running containers
	})
	if err != nil {
		return nil, fmt.Errorf("docker api list: %w", err)
	}

	var discovered []DiscoveredContainer
	for _, c := range containers {
		// Check for our container prefix
		var containerName string
		for _, name := range c.Names {
			name = strings.TrimPrefix(name, "/")
			if strings.HasPrefix(name, "aigw-") {
				containerName = name
				break
			}
		}
		if containerName == "" {
			continue
		}

		// Extract model alias from container name: aigw-{alias}-{timestamp}
		parts := strings.Split(containerName, "-")
		if len(parts) < 2 {
			continue
		}
		// Reconstruct alias (everything between "aigw-" and the last part which is timestamp)
		alias := strings.Join(parts[1:len(parts)-1], "-")
		if alias == "" {
			alias = parts[1] // Fallback for simple names
		}

		// Detect provider from image
		provider := detectProviderFromImage(c.Image)

		// Get port mapping to construct endpoint
		endpoint := ""
		for _, port := range c.Ports {
			if port.PublicPort > 0 {
				endpoint = fmt.Sprintf("http://127.0.0.1:%d", port.PublicPort)
				break
			}
		}

		dc := DiscoveredContainer{
			ID:         c.ID,
			Name:       containerName,
			Image:      c.Image,
			ModelAlias: alias,
			Provider:   provider,
			Endpoint:   endpoint,
			Status:     c.State,
			CreatedAt:  time.Unix(c.Created, 0),
		}
		discovered = append(discovered, dc)

		// Also register in handles map
		r.handles[containerName] = &ContainerHandle{
			ID:         c.ID,
			Provider:   provider,
			ModelAlias: alias,
			Endpoint:   endpoint,
		}

		// Start log streaming for discovered container
		r.startLogStreaming(alias, c.ID)

		r.logger.WithFields(logrus.Fields{
			"container": containerName,
			"alias":     alias,
			"provider":  provider,
			"endpoint":  endpoint,
			"image":     c.Image,
		}).Info("Discovered running inference container")
	}

	return discovered, nil
}

func (r *DockerRuntime) discoverContainersCLI(ctx context.Context) ([]DiscoveredContainer, error) {
	// docker ps --filter "name=aigw-" --format "{{.ID}}|{{.Names}}|{{.Image}}|{{.Ports}}|{{.State}}|{{.CreatedAt}}"
	cmd := exec.CommandContext(ctx, r.dockerBin, "ps", "--filter", "name=aigw-", "--format", "{{.ID}}|{{.Names}}|{{.Image}}|{{.Ports}}|{{.State}}|{{.CreatedAt}}")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker ps failed: %w", err)
	}

	var discovered []DiscoveredContainer
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 5 {
			continue
		}

		containerID := parts[0]
		containerName := parts[1]
		image := parts[2]
		portsStr := parts[3]
		state := parts[4]

		// Extract model alias from container name
		nameParts := strings.Split(containerName, "-")
		if len(nameParts) < 2 {
			continue
		}
		alias := strings.Join(nameParts[1:len(nameParts)-1], "-")
		if alias == "" {
			alias = nameParts[1]
		}

		// Detect provider from image
		provider := detectProviderFromImage(image)

		// Parse port mapping: "0.0.0.0:12345->8080/tcp" or "127.0.0.1:12345->8080/tcp"
		endpoint := ""
		if portsStr != "" {
			for _, portMap := range strings.Split(portsStr, ", ") {
				if idx := strings.Index(portMap, "->"); idx > 0 {
					hostPart := portMap[:idx]
					if colonIdx := strings.LastIndex(hostPart, ":"); colonIdx >= 0 {
						port := hostPart[colonIdx+1:]
						endpoint = fmt.Sprintf("http://127.0.0.1:%s", port)
						break
					}
				}
			}
		}

		dc := DiscoveredContainer{
			ID:         containerID,
			Name:       containerName,
			Image:      image,
			ModelAlias: alias,
			Provider:   provider,
			Endpoint:   endpoint,
			Status:     state,
		}
		discovered = append(discovered, dc)

		// Register in handles map
		r.handles[containerName] = &ContainerHandle{
			ID:         containerID,
			Provider:   provider,
			ModelAlias: alias,
			Endpoint:   endpoint,
		}

		// Start log streaming
		r.startLogStreaming(alias, containerID)

		r.logger.WithFields(logrus.Fields{
			"container": containerName,
			"alias":     alias,
			"provider":  provider,
			"endpoint":  endpoint,
			"image":     image,
		}).Info("Discovered running inference container")
	}

	return discovered, nil
}

// detectProviderFromImage determines provider type from Docker image name.
func detectProviderFromImage(image string) ProviderKind {
	imageLower := strings.ToLower(image)
	switch {
	case strings.Contains(imageLower, "vllm"):
		return ProviderVLLM
	case strings.Contains(imageLower, "sglang"):
		return ProviderSGLang
	case strings.Contains(imageLower, "text-generation-inference") || strings.Contains(imageLower, "tgi"):
		return ProviderTGI
	case strings.Contains(imageLower, "text-embeddings-inference") || strings.Contains(imageLower, "tei"):
		return ProviderTEI
	case strings.Contains(imageLower, "tensorrt") || strings.Contains(imageLower, "trt"):
		return ProviderTRTLLM
	case strings.Contains(imageLower, "llama.cpp") || strings.Contains(imageLower, "llama-cpp"):
		return ProviderLlamaCPP
	default:
		return ProviderVLLM // Default fallback
	}
}