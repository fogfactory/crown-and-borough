package notify

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/fogfactory/crown-and-borough/internal/models"
	"github.com/fogfactory/crown-and-borough/internal/store"
)

// Limits of the announcement templates.
const (
	// DiscordMaxContentRunes is Discord's limit on the content of a message.
	DiscordMaxContentRunes = 2000
	// maxTemplateRunes bounds the template text a host can submit.
	maxTemplateRunes = 2000
	// Longest values the variables take, to check the length of a template
	// before any game exists: game name (safeGameName), turn, season name and
	// game link.
	worstGameRunes   = maxGameNameRunes + 1
	worstTurnRunes   = 6
	worstSeasonRunes = 12
	worstURLRunes    = 256
	maxMentionRoles  = 100
)

// ErrTemplateTooLong is returned for a template whose message could exceed
// DiscordMaxContentRunes once its variables are replaced.
var ErrTemplateTooLong = errors.New("notify: discord template too long")

// UnknownVariableError is returned for a {name} that is not a variable.
type UnknownVariableError struct{ Name string }

func (e *UnknownVariableError) Error() string {
	return fmt.Sprintf("notify: unknown template variable {%s}", e.Name)
}

var (
	templateVariable = regexp.MustCompile(`\{([^{}]*)\}`)
	roleMention      = regexp.MustCompile(`<@&([0-9]{1,25})>`)
	groupMention     = regexp.MustCompile(`@(?:everyone|here)`)
)

// DiscordVars are the values of the template variables. They hold no message,
// author or order.
type DiscordVars struct {
	Game   string
	Turn   int
	Season models.Season
	URL    string
}

// ValidateDiscordTemplate checks a template at save time: only the variables
// {game}, {turn}, {season} and {url}, and a message that fits in
// DiscordMaxContentRunes whatever their values. An empty template is valid (it
// stands for the default).
func ValidateDiscordTemplate(template string) error {
	if utf8.RuneCountInString(template) > maxTemplateRunes {
		return ErrTemplateTooLong
	}
	worst := utf8.RuneCountInString(template)
	for _, match := range templateVariable.FindAllStringSubmatch(template, -1) {
		var size int
		switch match[1] {
		case "game":
			size = worstGameRunes
		case "turn":
			size = worstTurnRunes
		case "season":
			size = worstSeasonRunes
		case "url":
			size = worstURLRunes
		default:
			return &UnknownVariableError{Name: match[1]}
		}
		worst += size - utf8.RuneCountInString(match[0])
	}
	if worst > DiscordMaxContentRunes {
		return ErrTemplateTooLong
	}
	return nil
}

// RenderDiscordTemplate replaces the variables in one pass, so a value is never
// read as a variable itself. Values are neutralized: an @ in the game name
// cannot make a mention. The result is cut at DiscordMaxContentRunes.
func RenderDiscordTemplate(template, language string, vars DiscordVars) string {
	values := map[string]string{
		"game":   neutralizeMentions(safeGameName(vars.Game)),
		"turn":   strconv.Itoa(vars.Turn),
		"season": seasonName(vars.Season, language),
		"url":    neutralizeMentions(vars.URL),
	}
	content := templateVariable.ReplaceAllStringFunc(template, func(token string) string {
		if value, ok := values[token[1:len(token)-1]]; ok {
			return value
		}
		return token
	})
	if runes := []rune(content); len(runes) > DiscordMaxContentRunes {
		content = string(runes[:DiscordMaxContentRunes])
	}
	return content
}

// neutralizeMentions puts a zero-width space after every @, which Discord no
// longer reads as a mention.
func neutralizeMentions(value string) string {
	return strings.ReplaceAll(value, "@", "@​")
}

// DiscordMentionsOf computes allowed_mentions from the template text alone:
// "everyone" is parsed only when the template writes @everyone or @here, and
// the roles are the <@&id> the template writes. Users are never allowed.
func DiscordMentionsOf(template string) DiscordMention {
	mention := DiscordMention{Parse: []string{}, Roles: []string{}}
	if groupMention.MatchString(template) {
		mention.Parse = append(mention.Parse, "everyone")
	}
	seen := map[string]bool{}
	for _, match := range roleMention.FindAllStringSubmatch(template, -1) {
		if seen[match[1]] || len(mention.Roles) >= maxMentionRoles {
			continue
		}
		seen[match[1]] = true
		mention.Roles = append(mention.Roles, match[1])
	}
	return mention
}

func seasonName(season models.Season, language string) string {
	english := language == store.NotificationLanguageEnglish
	switch season {
	case models.SeasonSpring:
		if english {
			return "spring"
		}
		return "printemps"
	case models.SeasonSummer:
		if english {
			return "summer"
		}
		return "été"
	case models.SeasonAutumn:
		if english {
			return "autumn"
		}
		return "automne"
	case models.SeasonWinter:
		if english {
			return "winter"
		}
		return "hiver"
	}
	return string(season)
}

// DefaultDiscordTemplate is the template used when the host wrote none. It
// mentions nobody.
func DefaultDiscordTemplate(kind Kind, language string) string {
	if language == store.NotificationLanguageEnglish {
		if kind == KindNewTurn {
			return "A new turn has started in « {game} » ({season}, turn {turn}). {url}"
		}
		return "A house has written in the public room of « {game} ». {url}"
	}
	if kind == KindNewTurn {
		return "Un nouveau tour a commencé dans « {game} » ({season}, tour {turn}). {url}"
	}
	return "Une maison a écrit dans le salon public de « {game} ». {url}"
}

// discordTestTemplate is the text of the test message; it mentions nobody.
func discordTestTemplate(language string) string {
	if language == store.NotificationLanguageEnglish {
		return "Test message: the announcements of « {game} » will appear in this channel."
	}
	return "Message de test : les annonces de « {game} » arriveront dans ce salon."
}
