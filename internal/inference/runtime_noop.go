package inference

import (
	"context"
	"fmt"
)

// NoopRuntime is a placeholder runtime that returns not implemented errors.
type NoopRuntime struct{}

func (NoopRuntime) Start(ctx context.Context, req ContainerStartRequest) (*ContainerHandle, error) {
	return nil, fmt.Errorf("container runtime not configured for provider %s", req.Provider)
}

func (NoopRuntime) Stop(ctx context.Context, handleID string) error {
	return fmt.Errorf("container runtime not configured")
}

func (NoopRuntime) StopByAlias(ctx context.Context, alias string) error {
	return nil // no-op: no real containers to clean up
}

func (NoopRuntime) IsRunning(ctx context.Context, handleID string) (bool, error) {
	return false, nil
}

func (NoopRuntime) Logs(ctx context.Context, handleID string, tailLines int) (string, error) {
	return "", fmt.Errorf("container runtime not configured")
}

