// Package telegram provides GitLab review notification service via Telegram
package telegram

import (
	"context"
	"fmt"
	"strings"

	"aigateway/internal/models"
)

// ReviewNotifier sends review notifications via Telegram
type ReviewNotifier struct {
	client     *Client
	chatIDs    map[string][]string // projectID -> chatIDs
	enabled    bool
}

// NotifierConfig holds notifier configuration
type NotifierConfig struct {
	Enabled    bool              `json:"enabled" yaml:"enabled"`
	BotToken   string            `json:"bot_token" yaml:"bot_token"`
	DefaultChat string           `json:"default_chat" yaml:"default_chat"`
	ProjectChats map[string][]string `json:"project_chats" yaml:"project_chats"` // projectID -> chatIDs
}

// NewReviewNotifier creates a new review notifier
func NewReviewNotifier(config NotifierConfig) *ReviewNotifier {
	if !config.Enabled || config.BotToken == "" {
		return &ReviewNotifier{enabled: false}
	}

	client := NewClient(Config{
		BotToken: config.BotToken,
	})

	chatIDs := config.ProjectChats
	if chatIDs == nil {
		chatIDs = make(map[string][]string)
	}
	
	// Add default chat to all projects if specified
	if config.DefaultChat != "" {
		chatIDs["default"] = []string{config.DefaultChat}
	}

	return &ReviewNotifier{
		client:  client,
		chatIDs: chatIDs,
		enabled: true,
	}
}

// NotifyReviewStarted notifies that a review has started
func (n *ReviewNotifier) NotifyReviewStarted(ctx context.Context, review *models.GitLabMRReview, project *models.GitLabProject) error {
	if !n.enabled {
		return nil
	}

	chats := n.getChatsForProject(project.ID)
	if len(chats) == 0 {
		return nil
	}

	message := n.formatStartedMessage(review, project)
	
	for _, chatID := range chats {
		_, err := n.client.SendMessage(ctx, chatID, message, &MessageOptions{
			ParseMode:         "HTML",
			DisableWebPreview: true,
		})
		if err != nil {
			return fmt.Errorf("send to chat %s: %w", chatID, err)
		}
	}

	return nil
}

// NotifyReviewCompleted notifies that a review has completed
func (n *ReviewNotifier) NotifyReviewCompleted(ctx context.Context, review *models.GitLabMRReview, project *models.GitLabProject, result *models.GitLabReviewResult) error {
	if !n.enabled {
		return nil
	}

	chats := n.getChatsForProject(project.ID)
	if len(chats) == 0 {
		return nil
	}

	message := n.formatCompletedMessage(review, project, result)
	keyboard := n.createReviewKeyboard(review)
	
	for _, chatID := range chats {
		_, err := n.client.SendMessage(ctx, chatID, message, &MessageOptions{
			ParseMode:         "HTML",
			DisableWebPreview: false,
			ReplyMarkup:       keyboard,
		})
		if err != nil {
			return fmt.Errorf("send to chat %s: %w", chatID, err)
		}
	}

	return nil
}

// NotifyReviewFailed notifies that a review has failed
func (n *ReviewNotifier) NotifyReviewFailed(ctx context.Context, review *models.GitLabMRReview, project *models.GitLabProject, errMsg string) error {
	if !n.enabled {
		return nil
	}

	chats := n.getChatsForProject(project.ID)
	if len(chats) == 0 {
		return nil
	}

	message := n.formatFailedMessage(review, project, errMsg)
	
	for _, chatID := range chats {
		_, err := n.client.SendMessage(ctx, chatID, message, &MessageOptions{
			ParseMode:         "HTML",
			DisableWebPreview: true,
		})
		if err != nil {
			return fmt.Errorf("send to chat %s: %w", chatID, err)
		}
	}

	return nil
}

// AddProjectChat adds a chat ID for a project
func (n *ReviewNotifier) AddProjectChat(projectID, chatID string) {
	if n.chatIDs == nil {
		n.chatIDs = make(map[string][]string)
	}
	n.chatIDs[projectID] = append(n.chatIDs[projectID], chatID)
}

// RemoveProjectChat removes a chat ID from a project
func (n *ReviewNotifier) RemoveProjectChat(projectID, chatID string) {
	if chats, ok := n.chatIDs[projectID]; ok {
		var newChats []string
		for _, c := range chats {
			if c != chatID {
				newChats = append(newChats, c)
			}
		}
		n.chatIDs[projectID] = newChats
	}
}

func (n *ReviewNotifier) getChatsForProject(projectID string) []string {
	// Check project-specific chats first
	if chats, ok := n.chatIDs[projectID]; ok && len(chats) > 0 {
		return chats
	}
	
	// Fall back to default chat
	if chats, ok := n.chatIDs["default"]; ok {
		return chats
	}
	
	return nil
}

func (n *ReviewNotifier) formatStartedMessage(review *models.GitLabMRReview, project *models.GitLabProject) string {
	var sb strings.Builder
	
	sb.WriteString("🔄 <b>AI Review Started</b>\n\n")
	sb.WriteString(fmt.Sprintf("📁 <b>Project:</b> %s\n", escapeHTML(project.Name)))
	sb.WriteString(fmt.Sprintf("🔀 <b>MR:</b> #%d - %s\n", review.MRIID, escapeHTML(review.MRTitle)))
	sb.WriteString(fmt.Sprintf("👤 <b>Author:</b> %s\n", escapeHTML(review.MRAuthor)))
	sb.WriteString(fmt.Sprintf("🌿 <b>Branch:</b> %s → %s\n", 
		escapeHTML(review.SourceBranch), 
		escapeHTML(review.TargetBranch)))
	
	return sb.String()
}

func (n *ReviewNotifier) formatCompletedMessage(review *models.GitLabMRReview, project *models.GitLabProject, result *models.GitLabReviewResult) string {
	var sb strings.Builder
	
	// Determine emoji based on score
	scoreEmoji := "🟢"
	if result.OverallScore < 50 {
		scoreEmoji = "🔴"
	} else if result.OverallScore < 70 {
		scoreEmoji = "🟡"
	}
	
	sb.WriteString("✅ <b>AI Review Completed</b>\n\n")
	sb.WriteString(fmt.Sprintf("📁 <b>Project:</b> %s\n", escapeHTML(project.Name)))
	sb.WriteString(fmt.Sprintf("🔀 <b>MR:</b> #%d - %s\n", review.MRIID, escapeHTML(review.MRTitle)))
	sb.WriteString(fmt.Sprintf("👤 <b>Author:</b> %s\n\n", escapeHTML(review.MRAuthor)))
	
	sb.WriteString(fmt.Sprintf("%s <b>Score:</b> %d/100\n", scoreEmoji, result.OverallScore))
	sb.WriteString(fmt.Sprintf("📊 <b>Files:</b> %d | <b>Issues:</b> %d\n", 
		review.FilesAnalyzed, review.IssuesFound))
	sb.WriteString(fmt.Sprintf("⏱ <b>Time:</b> %.1fs | <b>Tokens:</b> %d\n", 
		float64(review.ProcessingTimeMs)/1000, review.TokensUsed))
	
	if result.Summary != "" {
		sb.WriteString(fmt.Sprintf("\n📝 <i>%s</i>\n", escapeHTML(truncate(result.Summary, 200))))
	}
	
	return sb.String()
}

func (n *ReviewNotifier) formatFailedMessage(review *models.GitLabMRReview, project *models.GitLabProject, errMsg string) string {
	var sb strings.Builder
	
	sb.WriteString("❌ <b>AI Review Failed</b>\n\n")
	sb.WriteString(fmt.Sprintf("📁 <b>Project:</b> %s\n", escapeHTML(project.Name)))
	sb.WriteString(fmt.Sprintf("🔀 <b>MR:</b> #%d - %s\n", review.MRIID, escapeHTML(review.MRTitle)))
	sb.WriteString(fmt.Sprintf("👤 <b>Author:</b> %s\n\n", escapeHTML(review.MRAuthor)))
	sb.WriteString(fmt.Sprintf("⚠️ <b>Error:</b> %s\n", escapeHTML(truncate(errMsg, 200))))
	sb.WriteString(fmt.Sprintf("\n<i>Retry count: %d/%d</i>", review.RetryCount, review.MaxRetries))
	
	return sb.String()
}

func (n *ReviewNotifier) createReviewKeyboard(review *models.GitLabMRReview) *InlineKeyboard {
	if review.MRURL == "" {
		return nil
	}
	
	return &InlineKeyboard{
		Buttons: [][]InlineKeyboardButton{
			{
				{
					Text: "📝 View MR",
					URL:  review.MRURL,
				},
			},
		},
	}
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

