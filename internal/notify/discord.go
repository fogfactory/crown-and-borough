package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

// discordWebhookPattern is the only URL shape the server posts to. It is
// strict on purpose: it is the SSRF guard, since the host of a game chooses
// the URL. No user info, port, query, fragment or extra path segment fits.
var discordWebhookPattern = regexp.MustCompile(`^https://(?:discord|discordapp)\.com/api/webhooks/[0-9]{1,25}/[A-Za-z0-9_-]{1,200}$`)

// ValidateDiscordWebhookURL accepts https://discord.com/api/webhooks/{id}/{token}
// (and discordapp.com) only.
func ValidateDiscordWebhookURL(raw string) error {
	if !discordWebhookPattern.MatchString(raw) {
		return ErrInvalidWebhookURL
	}
	return nil
}

var (
	// ErrInvalidWebhookURL is returned for any URL that is not a Discord
	// webhook URL.
	ErrInvalidWebhookURL = errors.New("notify: not a Discord webhook URL")
	// ErrWebhookGone means Discord answered 404 or 401: the webhook no longer
	// exists or its token is invalid. Retrying is pointless.
	ErrWebhookGone = errors.New("notify: discord webhook is gone")
	// ErrDiscordRateLimited means Discord asked for a wait longer than the
	// bound, or kept answering 429.
	ErrDiscordRateLimited = errors.New("notify: discord rate limited")
	// ErrDiscordFailed is any other failure of the delivery.
	ErrDiscordFailed = errors.New("notify: discord delivery failed")
)

// DiscordMention is the allowed_mentions object of a webhook request. Parse is
// always serialized (an absent allowed_mentions would let Discord parse every
// mention of the content); there is no users field, so a user is never
// mentioned.
type DiscordMention struct {
	Parse []string `json:"parse"`
	Roles []string `json:"roles,omitempty"`
}

// DiscordMessage is one announcement. URL is the webhook secret.
type DiscordMessage struct {
	URL     string
	Content string
	Mention DiscordMention
}

// String and GoString keep the secret out of any formatted log.
func (DiscordMessage) String() string   { return "discord message" }
func (DiscordMessage) GoString() string { return "notify.DiscordMessage{}" }

type discordPayload struct {
	Content         string         `json:"content"`
	AllowedMentions DiscordMention `json:"allowed_mentions"`
}

const (
	// discordTimeout bounds one request to Discord.
	discordTimeout = 5 * time.Second
	// maxRetryAfter is the longest 429 wait honoured; a longer one gives up.
	maxRetryAfter = 5 * time.Second
	// maxDiscordAttempts bounds the requests made for one message.
	maxDiscordAttempts = 3
)

// DiscordClient posts to Discord webhooks over HTTPS, without following
// redirects.
type DiscordClient struct {
	http  *http.Client
	sleep func(ctx context.Context, wait time.Duration) error
}

var _ Sender[DiscordMessage] = (*DiscordClient)(nil)

func NewDiscordClient() *DiscordClient { return newDiscordClient(nil) }

// NewDiscordClientWithTransport is NewDiscordClient over a custom transport,
// for the demo build, which redirects the requests to a local receiver. The
// client still validates the URL and builds the payload itself.
func NewDiscordClientWithTransport(transport http.RoundTripper) *DiscordClient {
	return newDiscordClient(transport)
}

func newDiscordClient(transport http.RoundTripper) *DiscordClient {
	return &DiscordClient{
		http: &http.Client{
			Timeout:   discordTimeout,
			Transport: transport,
			// A redirect answer is a failure, never followed.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
		sleep: func(ctx context.Context, wait time.Duration) error {
			timer := time.NewTimer(wait)
			defer timer.Stop()
			select {
			case <-timer.C:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}
}

func (c *DiscordClient) Send(ctx context.Context, message DiscordMessage) error {
	if err := ValidateDiscordWebhookURL(message.URL); err != nil {
		return err
	}
	if message.Mention.Parse == nil {
		message.Mention.Parse = []string{}
	}
	body, err := json.Marshal(discordPayload{Content: message.Content, AllowedMentions: message.Mention})
	if err != nil {
		return fmt.Errorf("%w: encode", ErrDiscordFailed)
	}
	for attempt := 1; ; attempt++ {
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, message.URL, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("%w: build request", ErrDiscordFailed)
		}
		request.Header.Set("Content-Type", "application/json")
		response, err := c.http.Do(request)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			// net/http errors quote the URL: keep only the cause.
			var urlErr *url.Error
			if errors.As(err, &urlErr) {
				err = urlErr.Err
			}
			return fmt.Errorf("%w: %v", ErrDiscordFailed, err)
		}
		retryAfter := response.Header.Get("Retry-After")
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
		_ = response.Body.Close()
		switch {
		case response.StatusCode >= 200 && response.StatusCode < 300:
			return nil
		case response.StatusCode == http.StatusNotFound, response.StatusCode == http.StatusUnauthorized:
			return ErrWebhookGone
		case response.StatusCode == http.StatusTooManyRequests:
			wait, ok := parseRetryAfter(retryAfter)
			if !ok || wait > maxRetryAfter || attempt >= maxDiscordAttempts {
				return ErrDiscordRateLimited
			}
			if err := c.sleep(ctx, wait); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%w: status %d", ErrDiscordFailed, response.StatusCode)
		}
	}
}

// parseRetryAfter reads the delay of a 429, in seconds (possibly fractional).
func parseRetryAfter(value string) (time.Duration, bool) {
	seconds, err := strconv.ParseFloat(value, 64)
	if err != nil || seconds < 0 || seconds > 3600 {
		return 0, false
	}
	return time.Duration(seconds * float64(time.Second)), true
}
