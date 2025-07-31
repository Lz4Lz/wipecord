package discord

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	Token    string
	HTTP     *http.Client
	UserInfo *Profile
	DMS      []Channel
}

func (e RateLimitError) Error() string {
	return fmt.Sprintf("rate limited: retry after %s", e.RetryAfter)
}

func NewClient(token string) *Client {
	return &Client{
		Token: token,
		HTTP:  &http.Client{},
	}
}

func (c *Client) Request(method, endpoint string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, "https://discord.com/api/v9"+endpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.Token)
	req.Header.Set("Content-Type", "application/json")
	return c.HTTP.Do(req)
}

func (c *Client) TokenCheck() error {
	resp, err := c.Request("GET", "/users/@me", nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Invalid Token!")
	}
	return nil
}

func (c *Client) FetchDMS() error {
	resp, err := c.Request("GET", "/users/@me/channels", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get user info: status %s", resp.Status)
	}

	var dm []Channel

	err = json.NewDecoder(resp.Body).Decode(&dm)
	if err != nil {
		return err
	}
	c.DMS = dm

	return nil
}

func (c *Client) FetchCurrentUser() error {
	resp, err := c.Request("GET", "/users/@me", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get user info: status %s", resp.Status)
	}

	var user Profile
	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		return err
	}
	c.UserInfo = &user
	return nil
}

/* Pagnation support if beforeID is set */
func (c *Client) FetchMessages(channelID, beforeID string) ([]Message, error) {

	endpoint := fmt.Sprintf("/channels/%s/messages?limit=100", channelID)

	if beforeID != "" {
		endpoint += fmt.Sprintf("&before=%s", beforeID)
	}

	resp, err := c.Request("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode == 429 {
		var body struct {
			RetryAfter float64 `json:"retry_after"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			return nil, fmt.Errorf("rate limited but failed to parse retry_after: %w", err)
		}

		return nil, RateLimitError{RetryAfter: time.Duration(body.RetryAfter * float64(time.Second))}
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get messages: status %s", resp.Status)
	}

	var messages []Message
	if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
		return nil, err
	}

	return messages, nil
}

func (c *Client) DeleteMessage(channelID string, msg Message) error {

	resp, err := c.Request("DELETE", fmt.Sprintf("/channels/%s/messages/%s", channelID, msg.ID), nil)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode == 429 {

		var body struct {
			RetryAfter float64 `json:"retry_after"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			return fmt.Errorf("rate limited but failed to parse retry_after: %w", err)
		}

		return RateLimitError{RetryAfter: time.Duration(body.RetryAfter * float64(time.Second))}
	}

	if resp.StatusCode != 204 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error %s: %s", resp.Status, string(body))
	}

	return nil
}
