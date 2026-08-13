package i18n

import (
	"net/http/httptest"
	"testing"
)

func TestTranslateFormatsPlayerMessagesInBothLanguages(t *testing.T) {
	message := Message{Key: ErrorUnknownPlayer, Args: []any{"P9"}}
	tests := []struct {
		language Language
		want     string
	}{
		{language: English, want: `player "P9" does not exist`},
		{language: French, want: `le joueur "P9" n'existe pas`},
	}
	for _, test := range tests {
		t.Run(string(test.language), func(t *testing.T) {
			if got := Translate(test.language, message); got != test.want {
				t.Fatalf("Translate(%q) = %q, want %q", test.language, got, test.want)
			}
		})
	}
}

func TestFromRequestPrefersQueryThenAcceptLanguageAndDefaultsToEnglish(t *testing.T) {
	for _, test := range []struct {
		name   string
		url    string
		accept string
		want   Language
	}{
		{name: "query", url: "/api/orders?lang=fr-FR", accept: "en", want: French},
		{name: "header", url: "/api/orders", accept: "fr-FR, en;q=0.8", want: French},
		{name: "default", url: "/api/orders", accept: "de", want: English},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", test.url, nil)
			request.Header.Set("Accept-Language", test.accept)
			if got := FromRequest(request); got != test.want {
				t.Fatalf("FromRequest() = %q, want %q", got, test.want)
			}
		})
	}
}
