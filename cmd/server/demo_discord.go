//go:build demo

package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
)

// discordTransport is the transport of the Discord client. In demo builds
// (scripts/discord-demo.sh) DISCORD_DEMO_RECEIVER redirects every request to a
// local receiver instead of discord.com. The real client still validates the
// webhook URL and builds the payload; only the destination changes, and the
// webhook URL (token included) is never forwarded.
func discordTransport() http.RoundTripper {
	raw := os.Getenv("DISCORD_DEMO_RECEIVER")
	if raw == "" {
		return nil
	}
	receiver, err := url.Parse(raw)
	if err != nil || receiver.Scheme == "" || receiver.Host == "" {
		log.Fatalf("invalid DISCORD_DEMO_RECEIVER %q", raw)
	}
	return demoRedirect{receiver: receiver, next: http.DefaultTransport}
}

type demoRedirect struct {
	receiver *url.URL
	next     http.RoundTripper
}

func (d demoRedirect) RoundTrip(request *http.Request) (*http.Response, error) {
	redirected := request.Clone(request.Context())
	redirected.URL = &url.URL{Scheme: d.receiver.Scheme, Host: d.receiver.Host, Path: d.receiver.Path}
	redirected.Host = d.receiver.Host
	return d.next.RoundTrip(redirected)
}
