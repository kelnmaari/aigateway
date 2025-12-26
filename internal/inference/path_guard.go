package inference

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ensurePathWithin checks that the target path is inside one of the allowed roots.
func ensurePathWithin(target string, allowed []string) error {
	if target == "" {
		return fmt.Errorf("empty path")
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("abs path: %w", err)
	}
	for _, root := range allowed {
		if root == "" {
			continue
		}
		absRoot, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		if strings.HasPrefix(absTarget, absRoot+string(os.PathSeparator)) || absTarget == absRoot {
			return nil
		}
	}
	return fmt.Errorf("path %s is outside allowed roots", absTarget)
}

