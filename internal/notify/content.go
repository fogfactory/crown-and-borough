package notify

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"github.com/fogfactory/crown-and-borough/internal/store"
)

// renderEmail writes the email of an event. It holds no sensitive content: no
// message, no author, no order; only the game name and links.
func renderEmail(kind Kind, language, gameName, gameURL, profileURL string) (subject, body string) {
	if language == store.NotificationLanguageEnglish {
		switch kind {
		case KindNewTurn:
			subject = fmt.Sprintf("A new turn has started in %s", gameName)
			body = fmt.Sprintf("A new turn has started in your game %q.\n\nOpen the game: %s\n", gameName, gameURL)
		case KindPrivateMessage:
			subject = fmt.Sprintf("New private message in %s", gameName)
			body = fmt.Sprintf("Your house has received a private message in the game %q.\n\nOpen the game: %s\n", gameName, gameURL)
		default:
			subject = fmt.Sprintf("New message in the public room of %s", gameName)
			body = fmt.Sprintf("A house has written in the public room of the game %q.\n\nOpen the game: %s\n", gameName, gameURL)
		}
		body += "\nYou receive this email because you enabled notifications. Change or disable them in your profile: " + profileURL + "\n"
		return subject, body
	}
	switch kind {
	case KindNewTurn:
		subject = fmt.Sprintf("Un nouveau tour a commencé dans %s", gameName)
		body = fmt.Sprintf("Un nouveau tour a commencé dans votre partie « %s ».\n\nOuvrir la partie : %s\n", gameName, gameURL)
	case KindPrivateMessage:
		subject = fmt.Sprintf("Nouveau message privé dans %s", gameName)
		body = fmt.Sprintf("Votre maison a reçu un message privé dans la partie « %s ».\n\nOuvrir la partie : %s\n", gameName, gameURL)
	default:
		subject = fmt.Sprintf("Nouveau message dans le salon public de %s", gameName)
		body = fmt.Sprintf("Une maison a écrit dans le salon public de la partie « %s ».\n\nOuvrir la partie : %s\n", gameName, gameURL)
	}
	body += "\nVous recevez cet email parce que vous avez activé les notifications. Pour les modifier ou les désactiver, rendez-vous sur votre profil : " + profileURL + "\n"
	return subject, body
}

// gameURL and profileURL follow the front routes /games/:gameId and /profile.
func gameURL(baseURL string, id store.GameID) string {
	return baseURL + "/games/" + url.PathEscape(string(id))
}

func profileURL(baseURL string) string {
	return baseURL + "/profile"
}

// maxGameNameRunes bounds the game name quoted in an email.
const maxGameNameRunes = 60

// dotBeforeLetter finds the dots that would let a mail client turn a word of
// the name into a link ("evil.example").
var dotBeforeLetter = regexp.MustCompile(`\.(\pL)`)

// safeGameName prepares a game name, chosen by the host, for an email sent to
// the other players: one line, no control character, nothing a mail client
// turns into a link or an address, and a bounded length.
func safeGameName(name string) string {
	name = strings.Map(func(r rune) rune {
		switch {
		case unicode.IsControl(r):
			return ' '
		case r == '@' || r == '/' || r == '\\' || r == '<' || r == '>':
			return ' '
		}
		return r
	}, name)
	name = dotBeforeLetter.ReplaceAllString(name, ". $1")
	name = strings.Join(strings.Fields(name), " ")
	if runes := []rune(name); len(runes) > maxGameNameRunes {
		name = strings.TrimSpace(string(runes[:maxGameNameRunes])) + "…"
	}
	return name
}
