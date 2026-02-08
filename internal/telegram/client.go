package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client wraps HTTP calls to the Telegram Bot API.
type Client struct {
	token      string
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new Telegram Bot API client.
func NewClient(token string) *Client {
	return &Client{
		token: token,
		httpClient: &http.Client{
			Timeout: 35 * time.Second, // slightly longer than long-poll timeout
		},
		baseURL: fmt.Sprintf("https://api.telegram.org/bot%s", token),
	}
}

// GetMe validates the bot token by calling getMe.
func (c *Client) GetMe(ctx context.Context) (*User, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/getMe", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getMe request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read getMe response: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse getMe response: %w", err)
	}

	if !apiResp.OK {
		return nil, fmt.Errorf("getMe failed: %s", apiResp.Description)
	}

	var user User
	if err := json.Unmarshal(apiResp.Result, &user); err != nil {
		return nil, fmt.Errorf("failed to parse user from getMe: %w", err)
	}

	return &user, nil
}

// GetUpdates calls getUpdates with long polling.
func (c *Client) GetUpdates(ctx context.Context, offset int, timeout int) ([]Update, error) {
	url := fmt.Sprintf("%s/getUpdates?offset=%d&timeout=%d", c.baseURL, offset, timeout)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getUpdates request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read getUpdates response: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse getUpdates response: %w", err)
	}

	if !apiResp.OK {
		return nil, fmt.Errorf("getUpdates failed: %s", apiResp.Description)
	}

	var updates []Update
	if err := json.Unmarshal(apiResp.Result, &updates); err != nil {
		return nil, fmt.Errorf("failed to parse updates: %w", err)
	}

	return updates, nil
}

// SendMessage sends a text message to a chat.
func (c *Client) SendMessage(ctx context.Context, chatID int64, text string) error {
	reqBody := SendMessageRequest{
		ChatID: chatID,
		Text:   text,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal sendMessage request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/sendMessage", bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sendMessage request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read sendMessage response: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("failed to parse sendMessage response: %w", err)
	}

	if !apiResp.OK {
		return fmt.Errorf("sendMessage failed: %s", apiResp.Description)
	}

	return nil
}

// SendMessageChunked splits text into 4096-char chunks and sends multiple messages.
func (c *Client) SendMessageChunked(ctx context.Context, chatID int64, text string) error {
	const maxLen = 4096

	for len(text) > 0 {
		chunk := text
		if len(chunk) > maxLen {
			// Try to break at last newline within limit
			idx := strings.LastIndex(chunk[:maxLen], "\n")
			if idx > maxLen/2 {
				chunk = chunk[:idx+1]
			} else {
				chunk = chunk[:maxLen]
			}
		}

		if err := c.SendMessage(ctx, chatID, chunk); err != nil {
			return err
		}

		text = text[len(chunk):]
	}

	return nil
}
