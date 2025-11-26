package discord

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
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

func parseRateLimit(resp *http.Response) RateLimit {
	rl := RateLimit{}

	if s := resp.Header.Get("X-RateLimit-Remaining"); s != "" {
		rl.Remaining, _ = strconv.Atoi(s)
	}

	if s := resp.Header.Get("X-RateLimit-Reset-After"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			rl.ResetAfter = time.Duration(v * float64(time.Second))
		}
	}

	return rl
}

/* Pagnation support if beforeID is set */
func (c *Client) FetchMessages(channelID, beforeID string) ([]Message, RateLimit, error) {

	endpoint := fmt.Sprintf("/channels/%s/messages?limit=100", channelID)

	if beforeID != "" {
		endpoint += fmt.Sprintf("&before=%s", beforeID)
	}

	resp, err := c.Request("GET", endpoint, nil)
	if err != nil {
		return nil, RateLimit{}, fmt.Errorf("failed to send request: %w", err)
	}

	defer resp.Body.Close()

	rl := parseRateLimit(resp)

	if resp.StatusCode == 429 {
		var data struct {
			RetryAfter float64 `json:"retry_after"`
		}

		json.NewDecoder(resp.Body).Decode(&data)

		rl.Hit = true
		rl.RetryAfter = time.Duration(data.RetryAfter * float64(time.Second))

		return nil, rl, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, rl, fmt.Errorf("failed to get messages: %s (%s)", resp.Status, body)
	}

	var messages []Message
	if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
		return nil, rl, err
	}

	return messages, rl, nil
}

func (c *Client) DeleteMessage(channelID string, msg Message) (RateLimit, error) {

	resp, err := c.Request("DELETE", fmt.Sprintf("/channels/%s/messages/%s", channelID, msg.ID), nil)

	if err != nil {
		return RateLimit{}, err
	}

	defer resp.Body.Close()

	rl := parseRateLimit(resp)

	if resp.StatusCode == 429 {
		var data struct {
			RetryAfter float64 `json:"retry_after"`
		}
		json.NewDecoder(resp.Body).Decode(&data)

		rl.Hit = true
		rl.RetryAfter = time.Duration(data.RetryAfter * float64(time.Second))

		return rl, nil
	}

	if resp.StatusCode == http.StatusNoContent {
		return rl, nil
	}

	body, _ := io.ReadAll(resp.Body)
	return rl, fmt.Errorf("delete failed: %s (%s)", resp.Status, body)

}
