package purge

import (
	"fmt"
	"math/rand"
	"purge/internal/discord"
	"strings"
	"time"
)

type Purger struct {
	client *discord.Client
	userID string

	Filters     []string
	searchDelay time.Duration
	deleteDelay time.Duration
	maxAttempts int
}

func NewPurger(client *discord.Client) (*Purger, error) {

	if client.UserInfo == nil {
		if err := client.FetchCurrentUser(); err != nil {
			return nil, err
		}
	}

	return &Purger{
		client:      client,
		userID:      client.UserInfo.ID,
		searchDelay: 3000 * time.Millisecond,
		deleteDelay: 2000 * time.Millisecond,
		maxAttempts: 3,
	}, nil
}

func (p *Purger) SetFilters(filters []string) {
	p.Filters = filters
}

func (p *Purger) SetSearchDelay(d time.Duration) {
	if d > 0 {
		p.searchDelay = d
	}
}

func (p *Purger) SetDeleteDelay(d time.Duration) {
	if d > 0 {
		p.deleteDelay = d
	}
}

func (p *Purger) Purge(channelID string, push func(Update)) error {
	var before string
	var deleted, failed, throttled int
	const max429 = 10 // Safeguard, if you get 10 consecutive 429, discord has probably detected you using some tool.

	for {
		msgs, rl, err := p.client.FetchMessages(channelID, before)

		if rl.Hit {
			throttled++
			if err := p.handleRateLimit(rl.RetryAfter); err != nil {
				push(UpdateFailed{Message: err.Error()})
				return err
			}
			push(UpdateRateLimited{Timeout: rl.RetryAfter})
			continue
		}

		if err != nil {
			push(UpdateFailed{Message: err.Error()})
			return err
		}

		if len(msgs) == 0 {
			break
		}

		for _, m := range msgs {
			if m.Author.ID != p.userID || !p.matchesFilters(m.Content) {
				continue
			}

			if err := p.deleteMessage(channelID, m, push, &deleted, &failed, &throttled, max429); err != nil {
				return err
			}
		}
		before = msgs[len(msgs)-1].ID
		time.Sleep(p.searchDelay + RandDuration(50*time.Millisecond, 200*time.Millisecond))
	}

	push(UpdateDone{Deleted: deleted, Failed: failed, Throttled: throttled})

	return nil
}

func (p *Purger) matchesFilters(content string) bool {
	if len(p.Filters) == 0 {
		return true
	}
	lc := strings.ToLower(content)
	for _, f := range p.Filters {
		if strings.Contains(lc, strings.ToLower(f)) {
			return true
		}
	}
	return false
}

func (p *Purger) handleRateLimit(retryAfter time.Duration) error {
	if retryAfter > p.searchDelay {
		p.searchDelay = retryAfter
	}
	time.Sleep(retryAfter + RandDuration(100*time.Millisecond, 400*time.Millisecond))
	return nil
}

func (p *Purger) deleteMessage(channelID string, m discord.Message, push func(Update), deleted, failed, throttled *int, max429 int) error {
	attempts := 0
	consec429 := 0

	for attempts < p.maxAttempts {
		rl, err := p.client.DeleteMessage(channelID, m)

		if rl.Hit {
			*throttled++
			consec429++
			push(UpdateRateLimited{Timeout: rl.RetryAfter})

			if rl.RetryAfter > p.deleteDelay {
				p.deleteDelay = rl.RetryAfter
			}

			if consec429 >= max429 {
				push(UpdateFailed{Message: "too many 429s, exiting purge"})
				return fmt.Errorf("too many consecutive 429s")
			}
			time.Sleep(rl.RetryAfter + RandDuration(100*time.Millisecond, 400*time.Millisecond))
			continue
		}

		if err != nil {
			if isNotFound(err) {
				*deleted++
				push(UpdateDeleted{Content: m.Content})
				time.Sleep(p.deleteDelay + RandDuration(50*time.Millisecond, 300*time.Millisecond))
				return nil
			}
			attempts++
			if attempts >= p.maxAttempts {
				*failed++
				push(UpdateFailed{Message: err.Error()})
				return nil
			}
			time.Sleep(p.deleteDelay + RandDuration(50*time.Millisecond, 300*time.Millisecond))
			continue
		}
		*deleted++
		push(UpdateDeleted{Content: m.Content})
		time.Sleep(p.deleteDelay + RandDuration(50*time.Millisecond, 200*time.Millisecond))
		return nil
	}
	return nil
}

// If something is already deleted, this acts as a safeguard. Not found/404 means it's alteady been deleted.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "404") || strings.Contains(s, "not found")
}

// To make the purge seem less robotic, adding random ms delays.
func RandDuration(min, max time.Duration) time.Duration {
	if max <= min {
		return min
	}
	return min + time.Duration(rand.Int63n(int64(max-min)))
}
