package inference

import (
	"context"
	"fmt"
	"net"
	"os/exec"
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

	mu      sync.Mutex
	handles map[string]*ContainerHandle // containerName -> handle

	useAPI bool
	api    *client.Client
}

// DockerRuntimeConfig holds runtime options.
type DockerRuntimeConfig struct {
	DockerBin string
	Logger    *logrus.Logger
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
	r := &DockerRuntime{
		dockerBin: bin,
		logger:    cfg.Logger,
		handles:   make(map[string]*ContainerHandle),
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
				DeviceRequests: []container.DeviceRequest{
					{
						Capabilities: [][]string{{"gpu"}},
						Count:        -1, // all GPUs
					},
				},
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
	return handle, nil
}

func (r *DockerRuntime) startCLI(ctx context.Context, containerName string, req ContainerStartRequest, hostPorts map[string]int) (*ContainerHandle, error) {
	args := []string{"run", "-d", "--rm", "--name", containerName}

	// GPU access (all GPUs) if NVIDIA runtime available
	args = append(args, "--gpus", "all")

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
	return handle, nil
}

// Stop removes the container by ID.
func (r *DockerRuntime) Stop(ctx context.Context, handleID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.useAPI && r.api != nil {
		if err := r.api.ContainerRemove(ctx, handleID, container.RemoveOptions{Force: true}); err != nil {
			return fmt.Errorf("docker api rm failed: %w", err)
		}
		return nil
	}

	cmd := exec.CommandContext(ctx, r.dockerBin, "rm", "-f", handleID)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker rm failed: %w: %s", err, string(out))
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
