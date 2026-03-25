// Package telegram provides Telegram notifications for code reviews
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client handles Telegram Bot API interactions
type Client struct {
	token      string
	httpClient *http.Client
	baseURL    string
}

// Config holds Telegram client configuration
type Config struct {
	BotToken string        `json:"bot_token" yaml:"bot_token"`
	Timeout  time.Duration `json:"timeout" yaml:"timeout"`
}

// NewClient creates a new Telegram client
func NewClient(config Config) *Client {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		token: config.BotToken,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL: "https://api.telegram.org/bot",
	}
}

// SendMessage sends a message to a chat
func (c *Client) SendMessage(ctx context.Context, chatID string, text string, opts *MessageOptions) (*Message, error) {
	payload := map[string]any{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML",
	}

	if opts != nil {
		if opts.ParseMode != "" {
			payload["parse_mode"] = opts.ParseMode
		}
		if opts.DisableWebPreview {
			payload["disable_web_page_preview"] = true
		}
		if opts.DisableNotification {
			payload["disable_notification"] = true
		}
		if opts.ReplyToMessageID != 0 {
			payload["reply_to_message_id"] = opts.ReplyToMessageID
		}
		if opts.ReplyMarkup != nil {
			payload["reply_markup"] = opts.ReplyMarkup
		}
	}

	resp, err := c.doRequest(ctx, "sendMessage", payload)
	if err != nil {
		return nil, err
	}

	var result struct {
		OK     bool    `json:"ok"`
		Result Message `json:"result"`
		Error  string  `json:"description"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if !result.OK {
		return nil, fmt.Errorf("telegram API error: %s", result.Error)
	}

	return &result.Result, nil
}

// EditMessage edits an existing message
func (c *Client) EditMessage(ctx context.Context, chatID string, messageID int64, text string) (*Message, error) {
	payload := map[string]any{
		"chat_id":    chatID,
		"message_id": messageID,
		"text":       text,
		"parse_mode": "HTML",
	}

	resp, err := c.doRequest(ctx, "editMessageText", payload)
	if err != nil {
		return nil, err
	}

	var result struct {
		OK     bool    `json:"ok"`
		Result Message `json:"result"`
		Error  string  `json:"description"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if !result.OK {
		return nil, fmt.Errorf("telegram API error: %s", result.Error)
	}

	return &result.Result, nil
}

// GetMe returns bot information
func (c *Client) GetMe(ctx context.Context) (*User, error) {
	resp, err := c.doRequest(ctx, "getMe", nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		OK     bool   `json:"ok"`
		Result User   `json:"result"`
		Error  string `json:"description"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if !result.OK {
		return nil, fmt.Errorf("telegram API error: %s", result.Error)
	}

	return &result.Result, nil
}

// SetWebhook sets the webhook URL for receiving updates
func (c *Client) SetWebhook(ctx context.Context, url string, opts *WebhookOptions) error {
	payload := map[string]any{
		"url": url,
	}

	if opts != nil {
		if opts.Certificate != "" {
			payload["certificate"] = opts.Certificate
		}
		if len(opts.AllowedUpdates) > 0 {
			payload["allowed_updates"] = opts.AllowedUpdates
		}
		if opts.MaxConnections > 0 {
			payload["max_connections"] = opts.MaxConnections
		}
		if opts.SecretToken != "" {
			payload["secret_token"] = opts.SecretToken
		}
	}

	resp, err := c.doRequest(ctx, "setWebhook", payload)
	if err != nil {
		return err
	}

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"description"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("telegram API error: %s", result.Error)
	}

	return nil
}

// DeleteWebhook removes the webhook
func (c *Client) DeleteWebhook(ctx context.Context) error {
	resp, err := c.doRequest(ctx, "deleteWebhook", nil)
	if err != nil {
		return err
	}

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"description"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("telegram API error: %s", result.Error)
	}

	return nil
}

func (c *Client) doRequest(ctx context.Context, method string, payload any) ([]byte, error) {
	url := c.baseURL + c.token + "/" + method

	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal payload: %w", err)
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	return respBody, nil
}

// Types

// Message represents a Telegram message
type Message struct {
	MessageID int64  `json:"message_id"`
	From      *User  `json:"from,omitempty"`
	Chat      *Chat  `json:"chat"`
	Date      int64  `json:"date"`
	Text      string `json:"text,omitempty"`
}

// User represents a Telegram user
type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

// Chat represents a Telegram chat
type Chat struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"` // private, group, supergroup, channel
	Title     string `json:"title,omitempty"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

// MessageOptions contains optional message parameters
type MessageOptions struct {
	ParseMode           string `json:"parse_mode,omitempty"`
	DisableWebPreview   bool   `json:"disable_web_page_preview,omitempty"`
	DisableNotification bool   `json:"disable_notification,omitempty"`
	ReplyToMessageID    int64  `json:"reply_to_message_id,omitempty"`
	ReplyMarkup         any    `json:"reply_markup,omitempty"`
}

// WebhookOptions contains webhook configuration
type WebhookOptions struct {
	Certificate    string   `json:"certificate,omitempty"`
	AllowedUpdates []string `json:"allowed_updates,omitempty"`
	MaxConnections int      `json:"max_connections,omitempty"`
	SecretToken    string   `json:"secret_token,omitempty"`
}

// InlineKeyboard represents an inline keyboard
type InlineKeyboard struct {
	Buttons [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// InlineKeyboardButton represents a button
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	URL          string `json:"url,omitempty"`
	CallbackData string `json:"callback_data,omitempty"`
}
