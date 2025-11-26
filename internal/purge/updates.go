package purge

import "time"

/* "Update" structs for the TUI to get information */

type Update any

type UpdateDeleted struct {
	Content string
}

type UpdateFailed struct {
	Message string
}

type UpdateRateLimited struct {
	Timeout time.Duration
}

type UpdateInfo struct {
	Message string
}

type UpdateDone struct {
	Deleted   int
	Failed    int
	Throttled int
}
