// Package comment provides tools for building and formatting GitLab MR comments
package comment

import (
	"fmt"
	"strings"
	"time"

	"aigateway/internal/models"
)

// Builder builds formatted review comments
type Builder struct {
	showEmoji       bool
	showTimestamp   bool
	showStats       bool
	collapseSections bool
}

// NewBuilder creates a new comment builder with default settings
func NewBuilder() *Builder {
	return &Builder{
		showEmoji:       true,
		showTimestamp:   true,
		showStats:       true,
		collapseSections: true,
	}
}

// WithEmoji enables/disables emoji in comments
func (b *Builder) WithEmoji(enabled bool) *Builder {
	b.showEmoji = enabled
	return b
}

// WithTimestamp enables/disables timestamp in comments
func (b *Builder) WithTimestamp(enabled bool) *Builder {
	b.showTimestamp = enabled
	return b
}

// WithStats enables/disables stats section
func (b *Builder) WithStats(enabled bool) *Builder {
	b.showStats = enabled
	return b
}

// WithCollapsible enables/disables collapsible sections
func (b *Builder) WithCollapsible(enabled bool) *Builder {
	b.collapseSections = enabled
	return b
}

// BuildReviewComment builds the main review comment from analysis results
func (b *Builder) BuildReviewComment(result *models.GitLabReviewResult, stats ReviewStats) string {
	var sb strings.Builder

	// Header
	b.writeHeader(&sb, result)

	// Summary
	b.writeSummary(&sb, result)

	// Categories
	if len(result.Categories) > 0 {
		b.writeCategories(&sb, result.Categories)
	}

	// Suggestions
	if len(result.Suggestions) > 0 {
		b.writeSuggestions(&sb, result.Suggestions)
	}

	// File Reviews (collapsible if enabled)
	if len(result.FileReviews) > 0 {
		b.writeFileReviews(&sb, result.FileReviews)
	}

	// Stats footer
	if b.showStats {
		b.writeStats(&sb, stats)
	}

	// Footer
	b.writeFooter(&sb)

	return sb.String()
}

// ReviewStats contains statistics about the review
type ReviewStats struct {
	FilesAnalyzed    int
	LinesChanged     int
	IssuesFound      int
	ProcessingTimeMs int64
	TokensUsed       int
	Model            string
}

func (b *Builder) writeHeader(sb *strings.Builder, result *models.GitLabReviewResult) {
	emoji := ""
	if b.showEmoji {
		emoji = b.scoreEmoji(result.OverallScore) + " "
	}

	sb.WriteString(fmt.Sprintf("## %sAI Code Review\n\n", emoji))
}

func (b *Builder) writeSummary(sb *strings.Builder, result *models.GitLabReviewResult) {
	// Score badge
	scoreColor := b.scoreColor(result.OverallScore)
	sb.WriteString(fmt.Sprintf("**Overall Score:** ![%d/100](https://img.shields.io/badge/score-%d%%25--%s)\n\n", 
		result.OverallScore, result.OverallScore, scoreColor))

	// Summary text
	if result.Summary != "" {
		sb.WriteString(result.Summary)
		sb.WriteString("\n\n")
	}
}

func (b *Builder) writeCategories(sb *strings.Builder, categories []models.GitLabReviewCategory) {
	sb.WriteString("### Review Categories\n\n")
	sb.WriteString("| Category | Score | Issues |\n")
	sb.WriteString("|----------|-------|--------|\n")

	for _, cat := range categories {
		emoji := ""
		if b.showEmoji {
			emoji = b.categoryEmoji(cat.Name) + " "
		}
		sb.WriteString(fmt.Sprintf("| %s%s | %d/100 | %d |\n", 
			emoji, cat.Name, cat.Score, cat.IssueCount))
	}
	sb.WriteString("\n")
}

func (b *Builder) writeSuggestions(sb *strings.Builder, suggestions []models.GitLabSuggestion) {
	if b.collapseSections {
		sb.WriteString("<details>\n<summary>💡 Suggestions</summary>\n\n")
	} else {
		sb.WriteString("### 💡 Suggestions\n\n")
	}

	for _, s := range suggestions {
		priority := b.priorityBadge(s.Priority)
		sb.WriteString(fmt.Sprintf("#### %s %s\n\n%s\n\n", priority, s.Title, s.Description))
	}

	if b.collapseSections {
		sb.WriteString("</details>\n\n")
	}
}

func (b *Builder) writeFileReviews(sb *strings.Builder, fileReviews []models.GitLabFileReview) {
	if b.collapseSections {
		sb.WriteString("<details>\n<summary>📁 File Reviews</summary>\n\n")
	} else {
		sb.WriteString("### 📁 File Reviews\n\n")
	}

	for _, fr := range fileReviews {
		status := "✅"
		if !fr.Approved {
			status = "⚠️"
		}
		
		sb.WriteString(fmt.Sprintf("#### %s `%s`\n", status, fr.FilePath))
		sb.WriteString(fmt.Sprintf("*+%d/-%d lines*\n\n", fr.LinesAdded, fr.LinesRemoved))

		if len(fr.Issues) > 0 {
			for _, issue := range fr.Issues {
				severity := b.severityIcon(issue.Severity)
				lineInfo := fmt.Sprintf("Line %d", issue.Line)
				if issue.EndLine != nil {
					lineInfo = fmt.Sprintf("Lines %d-%d", issue.Line, *issue.EndLine)
				}

				sb.WriteString(fmt.Sprintf("- %s **%s** (%s): %s\n", 
					severity, issue.Category, lineInfo, issue.Message))
				
				if issue.Suggestion != "" {
					sb.WriteString(fmt.Sprintf("  > 💡 %s\n", issue.Suggestion))
				}
			}
			sb.WriteString("\n")
		}
	}

	if b.collapseSections {
		sb.WriteString("</details>\n\n")
	}
}

func (b *Builder) writeStats(sb *strings.Builder, stats ReviewStats) {
	sb.WriteString("---\n\n")
	sb.WriteString("<sub>\n")
	sb.WriteString(fmt.Sprintf("📊 **Stats:** %d files • %d lines changed • %d issues found\n", 
		stats.FilesAnalyzed, stats.LinesChanged, stats.IssuesFound))
	sb.WriteString(fmt.Sprintf("⏱️ **Processing:** %.2fs • %d tokens • Model: %s\n", 
		float64(stats.ProcessingTimeMs)/1000, stats.TokensUsed, stats.Model))
	sb.WriteString("</sub>\n")
}

func (b *Builder) writeFooter(sb *strings.Builder) {
	sb.WriteString("\n---\n")
	if b.showTimestamp {
		sb.WriteString(fmt.Sprintf("*Generated by AIGateway at %s*\n", 
			time.Now().UTC().Format("2006-01-02 15:04:05 UTC")))
	} else {
		sb.WriteString("*Generated by AIGateway*\n")
	}
}

func (b *Builder) scoreEmoji(score int) string {
	switch {
	case score >= 90:
		return "🌟"
	case score >= 70:
		return "✅"
	case score >= 50:
		return "⚠️"
	default:
		return "🔴"
	}
}

func (b *Builder) scoreColor(score int) string {
	switch {
	case score >= 90:
		return "brightgreen"
	case score >= 70:
		return "green"
	case score >= 50:
		return "yellow"
	case score >= 30:
		return "orange"
	default:
		return "red"
	}
}

func (b *Builder) categoryEmoji(category string) string {
	switch strings.ToLower(category) {
	case "security":
		return "🔒"
	case "performance":
		return "⚡"
	case "style":
		return "🎨"
	case "bugs", "bug":
		return "🐛"
	case "documentation":
		return "📚"
	case "testing":
		return "🧪"
	case "maintainability":
		return "🔧"
	default:
		return "📋"
	}
}

func (b *Builder) severityIcon(severity models.GitLabIssueSeverity) string {
	switch severity {
	case models.GitLabIssueSeverityCritical:
		return "🔴"
	case models.GitLabIssueSeverityHigh:
		return "🟠"
	case models.GitLabIssueSeverityMedium:
		return "🟡"
	case models.GitLabIssueSeverityLow:
		return "🟢"
	default:
		return "ℹ️"
	}
}

func (b *Builder) priorityBadge(priority string) string {
	switch strings.ToLower(priority) {
	case "high":
		return "🔴"
	case "medium":
		return "🟡"
	case "low":
		return "🟢"
	default:
		return "ℹ️"
	}
}

// BuildInlineComment builds a comment for a specific line
func (b *Builder) BuildInlineComment(issue models.GitLabCodeIssue) string {
	var sb strings.Builder

	severity := b.severityIcon(issue.Severity)
	sb.WriteString(fmt.Sprintf("%s **%s** [%s]\n\n", severity, 
		strings.Title(string(issue.Severity)), issue.Category))
	sb.WriteString(issue.Message)

	if issue.Suggestion != "" {
		sb.WriteString("\n\n💡 **Suggestion:**\n")
		sb.WriteString(issue.Suggestion)
	}

	if issue.CodeSnippet != "" {
		sb.WriteString("\n\n```suggestion\n")
		sb.WriteString(issue.CodeSnippet)
		sb.WriteString("\n```")
	}

	return sb.String()
}

