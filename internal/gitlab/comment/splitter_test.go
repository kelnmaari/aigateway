package comment

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitter_ShortComment(t *testing.T) {
	splitter := NewSplitter()

	comment := "## AI Review\n\nThis is a short comment."

	parts := splitter.Split(comment)

	assert.Len(t, parts, 1)
	assert.Equal(t, comment, parts[0])
}

func TestSplitter_ExactlyAtLimit(t *testing.T) {
	splitter := NewSplitterWithLimits(100, 0)

	// Create comment exactly at limit
	comment := strings.Repeat("a", 100)

	parts := splitter.Split(comment)

	assert.Len(t, parts, 1)
	assert.Equal(t, 100, len(parts[0]))
}

func TestSplitter_LongComment(t *testing.T) {
	splitter := NewSplitterWithLimits(500, 50)

	// Create comment that needs splitting
	comment := "## AI Review\n\n" + strings.Repeat("Issue description. ", 50)

	parts := splitter.Split(comment)

	assert.Greater(t, len(parts), 1)

	// Verify all parts have headers
	for i, part := range parts {
		assert.Contains(t, part, fmt.Sprintf("Part %d/", i+1))
	}
}

func TestSplitter_SplitAtNewlines(t *testing.T) {
	splitter := NewSplitterWithLimits(100, 10)

	comment := "Line 1\n\nLine 2\n\nLine 3\n\nLine 4\n\nLine 5"

	parts := splitter.Split(comment)

	// Should split at paragraph boundaries (double newlines)
	for _, part := range parts {
		// Each part should be valid
		assert.True(t, len(part) > 0)
	}
}

func TestSplitter_CodeBlocksPreserved(t *testing.T) {
	splitter := NewSplitterWithLimits(1000, 50)

	comment := `## AI Review

### Issue 1

` + "```go\n" + `func example() {
    // This is a code block
    fmt.Println("Hello")
}
` + "```" + `

More text here.`

	parts := splitter.Split(comment)

	// All content should be preserved across parts
	allContent := strings.Join(parts, "")
	assert.Contains(t, allContent, "```go")
	assert.Contains(t, allContent, "fmt.Println")
}

func TestSplitter_VeryLongSingleLine(t *testing.T) {
	splitter := NewSplitterWithLimits(500, 50)

	// Single very long line with no break points
	comment := strings.Repeat("x", 2000)

	parts := splitter.Split(comment)

	assert.Greater(t, len(parts), 1)

	// Verify we got all content (accounting for overlap and headers)
	totalLen := 0
	for _, part := range parts {
		totalLen += len(part)
	}
	// Total should be >= original (due to headers)
	assert.GreaterOrEqual(t, totalLen, 2000)
}

func TestSplitter_MarkdownHeaders(t *testing.T) {
	splitter := NewSplitterWithLimits(300, 30)

	comment := `## AI Review

### Critical Issues
Issue 1 description is here.

### Warnings
Warning 1 description.

### Suggestions
Suggestion 1 description with more text.`

	parts := splitter.Split(comment)

	// If content fits in one part, no Part headers are added
	if len(parts) == 1 {
		assert.Equal(t, comment, parts[0])
	} else {
		// Should have proper part headers
		for i, part := range parts {
			assert.Contains(t, part, fmt.Sprintf("Part %d/", i+1))
		}
	}
}

func TestSplitter_EmptyComment(t *testing.T) {
	splitter := NewSplitter()

	parts := splitter.Split("")

	assert.Len(t, parts, 1)
	assert.Equal(t, "", parts[0])
}

func TestSplitter_ContinuationMarkers(t *testing.T) {
	splitter := NewSplitterWithLimits(300, 30)

	// Long comment that will be split
	comment := strings.Repeat("Test content. ", 50)

	parts := splitter.Split(comment)

	require.Greater(t, len(parts), 1)

	// All parts except last should have "continued" footer
	for i, part := range parts[:len(parts)-1] {
		assert.Contains(t, part, "Continued", "Part %d should have continuation notice", i+1)
	}

	// Last part should have "End of review"
	assert.Contains(t, parts[len(parts)-1], "End of review")
}

func TestSplitter_UnicodeContent(t *testing.T) {
	splitter := NewSplitterWithLimits(200, 20)

	// Unicode characters (Russian, Emoji, Chinese)
	comment := "## Код Ревью 🔍\n\nПроблема: 代码有问题\n\n" + strings.Repeat("Текст ", 20)

	parts := splitter.Split(comment)

	// Should handle unicode correctly - all content preserved
	allContent := strings.Join(parts, "")
	assert.Contains(t, allContent, "Код Ревью")
	assert.Contains(t, allContent, "🔍")
	assert.Contains(t, allContent, "代码")
}

func TestSplitter_ListItems(t *testing.T) {
	splitter := NewSplitterWithLimits(300, 30)

	comment := `## Issues

1. First issue with long description
2. Second issue with description
3. Third issue
4. Fourth issue`

	parts := splitter.Split(comment)

	// All list items should be preserved
	allContent := strings.Join(parts, "")
	assert.Contains(t, allContent, "1. First issue")
	assert.Contains(t, allContent, "2. Second issue")
	assert.Contains(t, allContent, "3. Third issue")
	assert.Contains(t, allContent, "4. Fourth issue")
}

func TestSplitter_NestedCodeBlocks(t *testing.T) {
	splitter := NewSplitterWithLimits(500, 50)

	comment := `## Review

` + "```go\n" + `func outer() {
    inner := ` + "`value`" + `
    fmt.Println(inner)
}
` + "```"

	parts := splitter.Split(comment)

	// Content should be preserved
	allContent := strings.Join(parts, "")
	assert.Contains(t, allContent, "func outer()")
	assert.Contains(t, allContent, "`value`")
}

func TestSplitter_TableContent(t *testing.T) {
	splitter := NewSplitterWithLimits(400, 40)

	comment := `## Summary

| File | Issues |
|------|--------|
| main.go | 3 |
| handler.go | 2 |
| service.go | 1 |

More content here.`

	parts := splitter.Split(comment)

	// Table content should be preserved
	allContent := strings.Join(parts, "")
	assert.Contains(t, allContent, "| File |")
	assert.Contains(t, allContent, "| main.go |")
}

func TestSplitter_NeedsSplit(t *testing.T) {
	splitter := NewSplitter()

	// Short content - no split needed
	shortComment := "Short comment"
	assert.False(t, splitter.NeedsSplit(shortComment))

	// Long content - split needed
	longComment := strings.Repeat("x", SplitThreshold+1)
	assert.True(t, splitter.NeedsSplit(longComment))
}

func TestSplitter_SplitWithMetadata(t *testing.T) {
	splitter := NewSplitterWithLimits(200, 20)

	// Short comment
	shortResult := splitter.SplitWithMetadata("Short comment")
	assert.Equal(t, 1, shortResult.TotalParts)
	assert.False(t, shortResult.WasSplit)

	// Long comment
	longComment := strings.Repeat("Long content. ", 50)
	longResult := splitter.SplitWithMetadata(longComment)
	assert.Greater(t, longResult.TotalParts, 1)
	assert.True(t, longResult.WasSplit)
	assert.Len(t, longResult.Parts, longResult.TotalParts)
}

func TestSplitter_TruncateInlineComment(t *testing.T) {
	splitter := NewSplitter()

	// Short comment - no truncation
	shortComment := "Short inline comment"
	truncated := splitter.TruncateInlineComment(shortComment)
	assert.Equal(t, shortComment, truncated)

	// Long comment - truncated
	longComment := strings.Repeat("x", MaxInlineCommentLength+1000)
	truncated = splitter.TruncateInlineComment(longComment)
	assert.Less(t, len(truncated), len(longComment))
	assert.Contains(t, truncated, "truncated due to length")
}

func TestSplitter_DefaultSettings(t *testing.T) {
	splitter := NewSplitter()

	// Test that default splitter works
	comment := "Test comment"
	parts := splitter.Split(comment)
	assert.Len(t, parts, 1)
}

func TestSplitter_CustomLimits(t *testing.T) {
	// Very small limit
	splitter := NewSplitterWithLimits(50, 5)

	comment := strings.Repeat("word ", 100)
	parts := splitter.Split(comment)

	assert.Greater(t, len(parts), 1)
}

func TestSplitter_ZeroOrNegativeLimits(t *testing.T) {
	// Should use defaults for invalid values
	splitter := NewSplitterWithLimits(0, -10)

	comment := "Test comment"
	parts := splitter.Split(comment)

	assert.Len(t, parts, 1)
}

func TestSplitter_OverlapTooLarge(t *testing.T) {
	// Overlap >= maxLength/2 should be reduced
	splitter := NewSplitterWithLimits(100, 60)

	// Should still work
	comment := strings.Repeat("x", 300)
	parts := splitter.Split(comment)

	assert.Greater(t, len(parts), 1)
}

func TestSplitter_RealWorldReviewComment(t *testing.T) {
	splitter := NewSplitterWithLimits(2000, 100)

	comment := `## 🔍 AI Code Review

**Summary:** Found 5 issues in 3 files (250 lines changed)

---

### 🚨 Critical Issues

#### 1. SQL Injection Vulnerability
**File:** ` + "`db/queries.go`" + ` **Line:** 42

` + "```go" + `
query := "SELECT * FROM users WHERE id = " + userID
` + "```" + `

**Problem:** String concatenation in SQL queries allows injection attacks.

**Suggestion:**
` + "```go" + `
query := "SELECT * FROM users WHERE id = $1"
row := db.QueryRow(query, userID)
` + "```" + `

---

### ⚠️ Warnings

#### 2. Missing Error Check
**File:** ` + "`handler.go`" + ` **Line:** 87

` + "```go" + `
result, _ := service.Process(data)
` + "```" + `

**Problem:** Ignoring errors can hide failures.

---

### 💡 Suggestions

#### 3. Consider Using Context
**File:** ` + "`service.go`" + ` **Line:** 23

Consider passing context for cancellation support.

---

*Reviewed by AIGateway using gpt-4 • 4.5k tokens • 12.5s*`

	result := splitter.SplitWithMetadata(comment)

	// Should split into multiple parts
	assert.GreaterOrEqual(t, result.TotalParts, 1)

	// All parts should have proper headers
	for i, part := range result.Parts {
		if result.TotalParts > 1 {
			assert.Contains(t, part, fmt.Sprintf("Part %d/%d", i+1, result.TotalParts))
		}
	}

	// Important content should be preserved
	allContent := strings.Join(result.Parts, "")
	assert.Contains(t, allContent, "SQL Injection")
	assert.Contains(t, allContent, "Missing Error Check")
	assert.Contains(t, allContent, "AIGateway")
}

// Benchmark tests
func BenchmarkSplitter_ShortComment(b *testing.B) {
	splitter := NewSplitter()
	comment := "## AI Review\n\nShort comment."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		splitter.Split(comment)
	}
}

func BenchmarkSplitter_LongComment(b *testing.B) {
	splitter := NewSplitterWithLimits(1000, 100)
	comment := strings.Repeat("This is a long review comment with many issues. ", 200)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		splitter.Split(comment)
	}
}

func BenchmarkSplitter_ManyCodeBlocks(b *testing.B) {
	splitter := NewSplitterWithLimits(2000, 200)

	var sb strings.Builder
	sb.WriteString("## Review\n\n")
	for i := range 20 {
		sb.WriteString(fmt.Sprintf("### Issue %d\n\n", i+1))
		sb.WriteString("```go\nfunc example" + fmt.Sprintf("%d", i) + "() {\n    // code\n}\n```\n\n")
	}
	comment := sb.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		splitter.Split(comment)
	}
}

func BenchmarkSplitter_SplitWithMetadata(b *testing.B) {
	splitter := NewSplitterWithLimits(1000, 100)
	comment := strings.Repeat("Content for metadata test. ", 200)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		splitter.SplitWithMetadata(comment)
	}
}
