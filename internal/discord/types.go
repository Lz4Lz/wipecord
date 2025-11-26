package discord

import "time"

type UpdateType string

const (
	UpdateAdjust      UpdateType = "adjust"
	UpdateDeleted     UpdateType = "deleted"
	UpdateFailed      UpdateType = "failed"
	UpdateRateLimited UpdateType = "rate_limited"
	UpdateDelay       UpdateType = "delay"
	UpdateStatus      UpdateType = "status"
	UpdateDone        UpdateType = "done"
)

type Update struct {
	Type      UpdateType
	Message   string
	Content   string
	Delay     time.Duration
	Timeout   time.Duration
	Deleted   int
	Failed    int
	Throttled int
}

type RateLimitError struct {
	RetryAfter time.Duration
}

type RateLimit struct {
	RetryAfter time.Duration
	Remaining  int
	ResetAfter time.Duration
	Hit        bool
}

type Profile struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
}

type Channel struct {
	ID             string `json:"id"`
	Type           int    `json:"type"`
	LastMessageID  string `json:"last_message_id"`
	Flags          int    `json:"flags"`
	Recipients     []User `json:"recipients"`
	Name           string `json:"name,omitempty"`
	Icon           string `json:"icon,omitempty"`
	OwnerID        string `json:"owner_id,omitempty"`
	BlockedWarning bool   `json:"blocked_user_warning_dismissed,omitempty"`
}

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type Message struct {
	ID          string       `json:"id"`
	Content     string       `json:"content"`
	Author      Author       `json:"author"`
	Timestamp   string       `json:"timestamp"`
	ChannelID   string       `json:"channel_id"`
	Attachments []Attachment `json:"attachments"`
}

type Author struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type Attachment struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}
