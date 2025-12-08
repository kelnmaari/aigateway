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
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
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

	if r.useAPI && r.api != nil {
		if err := r.pullImageAPI(ctx, req.Image); err != nil {
			return nil, err
		}
		return r.startAPI(ctx, containerName, req, hostPorts)
	}
	if err := r.pullImageCLI(ctx, req.Image); err != nil {
		return nil, err
	}
	return r.startCLI(ctx, containerName, req, hostPorts)
}

func (r *DockerRuntime) startAPI(ctx context.Context, containerName string, req ContainerStartRequest, hostPorts map[string]int) (*ContainerHandle, error) {
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

	resp, err := r.api.ContainerCreate(ctx,
		&container.Config{
			Image:        req.Image,
			Cmd:          req.Command,
			Env:          env,
			ExposedPorts: exposed,
		},
		&container.HostConfig{
			PortBindings: portBindings,
			Mounts:       mounts,
			AutoRemove:   true,
			Resources: container.Resources{
				DeviceRequests: []container.DeviceRequest{gpuRequest},
			},
		},
		nil, nil, containerName,
	)
	if err != nil {
		return nil, fmt.Errorf("docker api create: %w", err)
	}

	if err := r.api.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
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
	args := []string{"run", "-d", "--rm", "--name", containerName}

	// GPU access - specific device(s) or all
	if req.GPUDevice != "" {
		// Use specific GPU(s), e.g., "0" or "0,1"
		args = append(args, "--gpus", fmt.Sprintf(`"device=%s"`, req.GPUDevice))
	} else {
		// Use all GPUs
		args = append(args, "--gpus", "all")
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

	cmd := exec.CommandContext(ctx, r.dockerBin, args...)
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
		buf := make([]byte, 64*1024)
		n, _ := rc.Read(buf)
		return string(buf[:n]), nil
	}
	// CLI fallback
	cmd := exec.CommandContext(ctx, r.dockerBin, "logs", "--tail", fmt.Sprintf("%d", tailLines), handleID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker logs failed: %w: %s", err, string(out))
	}
	return string(out), nil
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
