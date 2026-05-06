package youtube

import (
	"errors"
	"log"
	"time"

	"google.golang.org/api/googleapi"
)

// IsQuotaExceeded reports whether err is a YouTube Data API daily-quota
// exhaustion error (HTTP 403 with reason "quotaExceeded"). It does not
// match the short-window rateLimitExceeded / userRateLimitExceeded
// reasons — those clear within seconds and warrant retry, not abort.
func IsQuotaExceeded(err error) bool {
	var ge *googleapi.Error
	if !errors.As(err, &ge) {
		return false
	}
	if ge.Code != 403 {
		return false
	}
	for _, e := range ge.Errors {
		if e.Reason == "quotaExceeded" {
			return true
		}
	}
	return false
}

// nextQuotaReset returns the next time the YouTube Data API daily quota
// will refill. The API documents quota as resetting at midnight Pacific
// Time. The error response itself does not advertise a Retry-After or
// reset timestamp, so we compute it from the documented schedule.
func nextQuotaReset() time.Time {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)
	return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, loc)
}

// logQuotaExceeded emits a single warning describing which API hit the
// daily quota wall and when the quota is expected to refill. The
// YouTube error payload itself does not carry a reset timestamp, so the
// "lifts at" value is derived from the documented midnight-Pacific
// reset schedule.
func logQuotaExceeded(api string, err error) {
	reset := nextQuotaReset()
	log.Printf("WARNING: youtube: %s daily quota exceeded; aborting further YouTube API calls. Quota resets at %s (in ~%s). Original error: %v",
		api, reset.Format(time.RFC1123), time.Until(reset).Round(time.Minute), err)
}
