package discord

import (
	"fmt"
	"time"
)

func (c *Client) PurgeOwnDM(channelID string, updates chan<- Update) error {
	defer close(updates)

	if err := c.FetchCurrentUser(); err != nil {
		return err
	}

	searchDelay := 500 * time.Millisecond
	deleteDelay := 1000 * time.Millisecond

	updates <- Update{Type: UpdateDelay, Delay: deleteDelay}

	var totalDeleted, totalFailed, throttledCount int
	var lastMsgID string

	for {
		messages, err := c.FetchMessages(channelID, lastMsgID)
		if rlErr, ok := err.(RateLimitError); ok {
			time.Sleep(rlErr.RetryAfter)
			continue
		} else if err != nil {
			return err
		}

		if len(messages) == 0 {
			break
		}

		for _, msg := range messages {
			if msg.Author.ID != string(c.UserInfo.ID) {
				continue
			}
			for {
				err := c.DeleteMessage(channelID, msg)
				if err == nil {
					totalDeleted++
					updates <- Update{Type: UpdateDeleted, Content: msg.Content}
					if deleteDelay > 1000*time.Millisecond {
						deleteDelay -= 100 * time.Millisecond
					}
					break
				} else if rlErr, ok := err.(RateLimitError); ok {
					throttledCount++
					wait := rlErr.RetryAfter
					newDelay := time.Duration(1.2 * float64(wait))
					if newDelay > deleteDelay {
						deleteDelay = newDelay
					}
					updates <- Update{Type: UpdateRateLimited, Timeout: wait}
					updates <- Update{Type: UpdateDelay, Delay: deleteDelay}
					time.Sleep(wait)
				} else {
					totalFailed++
					updates <- Update{
						Type:    UpdateFailed,
						Message: fmt.Sprintf("Error deleting %s: %v", msg.ID, err),
					}
					break
				}
			}
			time.Sleep(deleteDelay)
		}

		lastMsgID = messages[len(messages)-1].ID
		updates <- Update{Type: UpdateDelay, Delay: searchDelay}
		time.Sleep(searchDelay)
	}

	updates <- Update{
		Type:      UpdateDone,
		Deleted:   totalDeleted,
		Failed:    totalFailed,
		Throttled: throttledCount,
	}

	return nil
}
