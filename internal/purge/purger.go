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

func (p *Purger) Purge(channelID string, push func(Update)) error {
	var before string
	var deleted, failed, throttled int

	const max429 = 10 // Should never hit, but *if* the discord api fucks you hard it'll hit i guess

	for {
		msgs, rl, err := p.client.FetchMessages(channelID, before)

		if rl.Hit {
			throttled++
			push(UpdateRateLimited{Timeout: rl.RetryAfter})

			if rl.RetryAfter > p.searchDelay {
				p.searchDelay = rl.RetryAfter
			}

			time.Sleep(rl.RetryAfter)
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
			if m.Author.ID != p.userID {
				continue
			}

			attempts := 0
			consec429 := 0

			for {
				if attempts >= p.maxAttempts {
					failed++
					push(UpdateFailed{Message: fmt.Sprintf("max attempts reached for message %s", m.ID)})
					break
				}

				rl, err := p.client.DeleteMessage(channelID, m)

				if rl.Hit {
					throttled++
					consec429++
					push(UpdateRateLimited{Timeout: rl.RetryAfter})

					if rl.RetryAfter > p.deleteDelay {
						p.deleteDelay = rl.RetryAfter
					}

					if consec429 >= max429 {
						failed++
						push(UpdateFailed{Message: fmt.Sprintf("too many 429s deleting message %s", m.ID)})
						break
					}

					time.Sleep(rl.RetryAfter)
					continue
				}

				if err != nil {
					if isNotFound(err) {
						deleted++
						push(UpdateDeleted{Content: m.Content})
						time.Sleep(p.deleteDelay)
						break
					}

					attempts++

					if attempts >= p.maxAttempts {
						failed++
						push(UpdateFailed{Message: err.Error()})
						break
					}
					time.Sleep(p.deleteDelay)
					continue
				}

				deleted++
				push(UpdateDeleted{Content: m.Content})
				time.Sleep(p.deleteDelay)
				break
			}
		}

		before = msgs[len(msgs)-1].ID

		time.Sleep(p.searchDelay)
	}

	push(UpdateDone{
		Deleted:   deleted,
		Failed:    failed,
		Throttled: throttled,
	})

	return nil
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "404") || strings.Contains(s, "not found")
}

func RandDuration(min, max time.Duration) time.Duration {
	if max <= min {
		return min
	}
	return min + time.Duration(rand.Int63n(int64(max-min)))
}
