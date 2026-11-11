package notify

import (
	"testing"
	"time"

	"github.com/fogfactory/crown-and-borough/internal/store"
)

func TestShouldSend(t *testing.T) {
	event := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	allOn := store.NotificationPreferences{NewTurn: true, PrivateMessage: true, PublicMessage: true}
	reduce := func(p store.NotificationPreferences) store.NotificationPreferences {
		p.ReduceWhenActive = true
		return p
	}
	cases := []struct {
		name        string
		preferences store.NotificationPreferences
		email       string
		kind        Kind
		lastActive  time.Time
		want        bool
	}{
		{"zero value never sends a new turn", store.NotificationPreferences{}, "a@x.test", KindNewTurn, time.Time{}, false},
		{"zero value never sends a private message", store.NotificationPreferences{}, "a@x.test", KindPrivateMessage, time.Time{}, false},
		{"zero value never sends a public message", store.NotificationPreferences{}, "a@x.test", KindPublicMessage, time.Time{}, false},
		{"reduce alone never sends", store.NotificationPreferences{ReduceWhenActive: true}, "a@x.test", KindNewTurn, time.Time{}, false},
		{"new turn opt-in", store.NotificationPreferences{NewTurn: true}, "a@x.test", KindNewTurn, time.Time{}, true},
		{"new turn opt-in does not cover messages", store.NotificationPreferences{NewTurn: true}, "a@x.test", KindPrivateMessage, time.Time{}, false},
		{"private opt-in", store.NotificationPreferences{PrivateMessage: true}, "a@x.test", KindPrivateMessage, time.Time{}, true},
		{"private opt-in does not cover public", store.NotificationPreferences{PrivateMessage: true}, "a@x.test", KindPublicMessage, time.Time{}, false},
		{"public opt-in", store.NotificationPreferences{PublicMessage: true}, "a@x.test", KindPublicMessage, time.Time{}, true},
		{"public opt-in does not cover private", store.NotificationPreferences{PublicMessage: true}, "a@x.test", KindPrivateMessage, time.Time{}, false},
		{"no email address", allOn, "", KindNewTurn, time.Time{}, false},
		{"unknown kind", allOn, "a@x.test", Kind("other"), time.Time{}, false},
		{"active player without reduce", allOn, "a@x.test", KindNewTurn, event.Add(-time.Minute), true},
		{"reduce skips an active player", reduce(allOn), "a@x.test", KindNewTurn, event.Add(-time.Minute), false},
		{"reduce skips at the window edge", reduce(allOn), "a@x.test", KindNewTurn, event.Add(-ActiveWindow), false},
		{"reduce sends just past the window", reduce(allOn), "a@x.test", KindNewTurn, event.Add(-ActiveWindow - time.Second), true},
		{"reduce sends to a player never seen", reduce(allOn), "a@x.test", KindPublicMessage, time.Time{}, true},
		{"reduce skips activity recorded after the event", reduce(allOn), "a@x.test", KindPublicMessage, event.Add(time.Second), false},
	}
	for _, tc := range cases {
		if got := ShouldSend(tc.preferences, tc.email, tc.kind, tc.lastActive, event); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}
