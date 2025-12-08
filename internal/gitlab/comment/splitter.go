// Package comment provides tools for splitting large comments to fit GitLab limits
package comment

import (
	"fmt"
	"strings"
)

// GitLab API limits
const (
	// MaxNoteLength is the maximum length for a GitLab note (1MB)
	MaxNoteLength = 1_000_000
	
	// SafeNoteLength is the practical limit for readability (~50KB)
	SafeNoteLength = 50_000
	
	// MaxInlineCommentLength is the practical limit for inline comments
	MaxInlineCommentLength = 10_000
	
	// SplitThreshold is when to start splitting (slightly below safe limit)
	SplitThreshold = 45_000
	
	// PartOverlap is the context overlap between parts
	PartOverlap = 500
)

// Splitter splits long comments into multiple parts
type Splitter struct {
	maxLength int
	overlap   int
}

// NewSplitter creates a new splitter with default settings
func NewSplitter() *Splitter {
	return &Splitter{
		maxLength: SafeNoteLength,
		overlap:   PartOverlap,
	}
}

// NewSplitterWithLimits creates a splitter with custom limits
func NewSplitterWithLimits(maxLength, overlap int) *Splitter {
	if maxLength <= 0 {
		maxLength = SafeNoteLength
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= maxLength/2 {
		overlap = maxLength / 10
	}
	return &Splitter{
		maxLength: maxLength,
		overlap:   overlap,
	}
}

// NeedsSplit returns true if the content exceeds the threshold
func (s *Splitter) NeedsSplit(content string) bool {
	return len(content) > SplitThreshold
}

// Split splits content into multiple parts if it exceeds the limit
func (s *Splitter) Split(content string) []string {
	if len(content) <= s.maxLength {
		return []string{content}
	}

	var parts []string
	remaining := content
	partNum := 1
	totalParts := s.estimateTotalParts(len(content))

	for len(remaining) > 0 {
		// Calculate max content length for this part (accounting for header/footer)
		headerLen := len(s.partHeader(partNum, totalParts))
		footerLen := len(s.partFooter(partNum, totalParts))
		maxContent := s.maxLength - headerLen - footerLen - 10 // 10 chars safety margin

		if maxContent <= 0 {
			maxContent = s.maxLength / 2
		}

		// Find the best split point
		splitAt := s.findSplitPoint(remaining, maxContent)
		if splitAt <= 0 {
			splitAt = min(len(remaining), maxContent)
		}

		// Extract this part's content
		partContent := remaining[:splitAt]

		// Build the complete part with header and footer
		header := s.partHeader(partNum, totalParts)
		footer := s.partFooter(partNum, totalParts)
		completePart := header + partContent + footer

		parts = append(parts, completePart)

		// Move to next part
		if splitAt >= len(remaining) {
			break
		}

		// Apply overlap for context continuity
		overlapStart := splitAt
		if s.overlap > 0 && splitAt > s.overlap {
			// Find a good overlap point (prefer line break)
			overlapStart = splitAt - s.overlap
			if idx := strings.LastIndex(remaining[overlapStart:splitAt], "\n"); idx != -1 {
				overlapStart = overlapStart + idx + 1
			}
		}
		remaining = remaining[overlapStart:]
		partNum++

		// Recalculate total parts if we have more content than expected
		if partNum > totalParts {
			totalParts = partNum + s.estimateTotalParts(len(remaining))
		}
	}

	// Update total parts count in all headers and footers (if different from estimate)
	actualTotal := len(parts)
	if actualTotal != totalParts {
		for i := range parts {
			// Update header
			oldHeader := s.partHeader(i+1, totalParts)
			newHeader := s.partHeader(i+1, actualTotal)
			parts[i] = strings.Replace(parts[i], oldHeader, newHeader, 1)
			
			// Update footer (especially important for last part)
			oldFooter := s.partFooter(i+1, totalParts)
			newFooter := s.partFooter(i+1, actualTotal)
			parts[i] = strings.Replace(parts[i], oldFooter, newFooter, 1)
		}
	}

	return parts
}

// findSplitPoint finds the best place to split content
// Prefers: ## headers > --- > \n\n > \n > maxLen
func (s *Splitter) findSplitPoint(content string, maxLen int) int {
	if len(content) <= maxLen {
		return len(content)
	}

	// Search window: last 2000 chars before maxLen
	searchStart := maxLen - 2000
	if searchStart < 0 {
		searchStart = 0
	}
	searchWindow := content[searchStart:maxLen]

	// Try to find section header (## )
	if idx := strings.LastIndex(searchWindow, "\n## "); idx != -1 {
		return searchStart + idx
	}

	// Try to find horizontal rule (---)
	if idx := strings.LastIndex(searchWindow, "\n---"); idx != -1 {
		return searchStart + idx
	}

	// Try to find paragraph break (\n\n)
	if idx := strings.LastIndex(searchWindow, "\n\n"); idx != -1 {
		return searchStart + idx + 1 // Include one newline
	}

	// Try to find any line break
	if idx := strings.LastIndex(searchWindow, "\n"); idx != -1 {
		return searchStart + idx + 1
	}

	// Fallback: split at maxLen
	return maxLen
}

// estimateTotalParts estimates how many parts the content will be split into
func (s *Splitter) estimateTotalParts(contentLen int) int {
	if contentLen <= s.maxLength {
		return 1
	}
	// Account for headers/footers in estimation
	effectiveMax := s.maxLength - 200 // ~200 chars for header+footer
	if effectiveMax <= 0 {
		effectiveMax = s.maxLength / 2
		if effectiveMax <= 0 {
			effectiveMax = 1
		}
	}
	parts := (contentLen + effectiveMax - 1) / effectiveMax
	return parts
}

func (s *Splitter) partHeader(partNum, totalParts int) string {
	return fmt.Sprintf("## 📄 AI Review (Part %d/%d)\n\n", partNum, totalParts)
}

func (s *Splitter) partFooter(partNum, totalParts int) string {
	if partNum == totalParts {
		return "\n\n---\n*End of review*"
	}
	return "\n\n---\n*Continued in next comment...*"
}

// SplitResult contains the result of splitting a comment
type SplitResult struct {
	Parts      []string
	TotalParts int
	WasSplit   bool
}

// SplitWithMetadata splits content and returns metadata about the split
func (s *Splitter) SplitWithMetadata(content string) SplitResult {
	parts := s.Split(content)
	return SplitResult{
		Parts:      parts,
		TotalParts: len(parts),
		WasSplit:   len(parts) > 1,
	}
}

// TruncateInlineComment truncates an inline comment to fit the limit
func (s *Splitter) TruncateInlineComment(content string) string {
	if len(content) <= MaxInlineCommentLength {
		return content
	}

	// Find a good truncation point
	truncateAt := MaxInlineCommentLength - 50 // Leave room for truncation notice
	
	// Prefer truncating at a line break
	if idx := strings.LastIndex(content[:truncateAt], "\n"); idx > truncateAt/2 {
		truncateAt = idx
	}

	return content[:truncateAt] + "\n\n*... (comment truncated due to length)*"
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

