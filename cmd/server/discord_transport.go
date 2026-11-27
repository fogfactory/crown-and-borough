//go:build !demo

package main

import "net/http"

// discordTransport is the transport of the Discord client: the default one.
func discordTransport() http.RoundTripper { return nil }
