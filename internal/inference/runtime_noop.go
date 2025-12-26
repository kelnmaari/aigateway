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

