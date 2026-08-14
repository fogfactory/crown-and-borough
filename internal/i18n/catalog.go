package i18n

import (
	"net/http"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// Language is one of the player-facing languages supported by the game.
type Language string

const (
	English Language = "en"
	French  Language = "fr"
)

// Message keeps a catalog key and its formatting arguments together. Engine
// errors retain this value so the API can render the same diagnostic in the
// language selected by the client.
type Message struct {
	Key  string
	Args []any
}

const (
	ErrorChainsInWinter    = "error.chains_in_winter"
	ErrorWinterOutOfSeason = "error.winter_out_of_season"
	ErrorUnknownPlayer     = "error.unknown_player"
	ErrorPlayerRequired    = "error.player_required"
	ErrorForeignChain      = "error.foreign_chain"
	ErrorForeignWinter     = "error.foreign_winter"
	ErrorNobleUnknown      = "error.unknown_noble"
	ErrorNobleMismatch     = "error.noble_mismatch"
	ErrorNobleNotOwned     = "error.noble_not_owned"
	ErrorDuplicateEmission = "error.duplicate_emission"

	ParseNoHeader                    = "error.parse.no_header"
	ParseBadHeader                   = "error.parse.bad_header"
	ParseNobleNotFound               = "error.parse.noble_not_found"
	ParseOrderSymbolMissing          = "error.parse.order_symbol_missing"
	ParseUnclosedParenthesis         = "error.parse.unclosed_parenthesis"
	ParsePositionRequired            = "error.parse.position_required"
	ParsePositionOnlyOne             = "error.parse.position_only_one"
	ParseOrderPositionRequired       = "error.parse.order_position_required"
	ParseUnsupportedOrderSymbol      = "error.parse.unsupported_order_symbol"
	ParseDestinationRequired         = "error.parse.destination_required"
	ParseDestinationOnlyOne          = "error.parse.destination_only_one"
	ParseSupportPositionRequired     = "error.parse.support_position_required"
	ParseOffensiveSupportDestination = "error.parse.offensive_support_destination"
	ParseSupportShape                = "error.parse.support_shape"
	ParseOffensiveSupportDash        = "error.parse.offensive_support_dash"
	ParseDisperseDestinationRequired = "error.parse.disperse_destination_required"
	ParseAssignmentDestination       = "error.parse.assignment_destination"
	ParseAssignmentEmptyNoble        = "error.parse.assignment_empty_noble"
	ParseAssignmentInvalidNoble      = "error.parse.assignment_invalid_noble"
	ParseAssignmentUnknownNoble      = "error.parse.assignment_unknown_noble"
	ParseCodeFormat                  = "error.parse.code_format"
	ParseTerritoryUnknown            = "error.parse.territory_unknown"

	WinterOrderShape             = "error.winter.order_shape"
	WinterOrderTargetOnlyOne     = "error.winter.target_only_one"
	WinterUnknownSymbol          = "error.winter.unknown_symbol"
	WinterTerritoryTrigramFormat = "error.winter.territory_code_format"
	WinterTerritoryUnknown       = "error.winter.territory_unknown"
	WinterNobleCodeFormat        = "error.winter.noble_code_format"
	WinterNobleUnknown           = "error.winter.noble_unknown"
	WinterUnknownSubtype         = "error.winter.unknown_subtype"

	ValidationUnknownNoble                 = "error.validation.unknown_noble"
	ValidationEmptyChain                   = "error.validation.empty_chain"
	ValidationDuplicateOrder               = "error.validation.duplicate_order"
	ValidationInvalidOrderID               = "error.validation.invalid_order_id"
	ValidationInvalidOrderType             = "error.validation.invalid_order_type"
	ValidationInvalidLiaison               = "error.validation.invalid_liaison"
	ValidationUnexpectedNobleAssignments   = "error.validation.unexpected_noble_assignments"
	ValidationUnknownPosition              = "error.validation.unknown_position"
	ValidationUnknownTarget                = "error.validation.unknown_target"
	ValidationJoinNotLast                  = "error.validation.join_not_last"
	ValidationUnexpectedTarget             = "error.validation.unexpected_target"
	ValidationMissingTarget                = "error.validation.missing_target"
	ValidationTooManyTargets               = "error.validation.too_many_targets"
	ValidationNotAdjacent                  = "error.validation.not_adjacent"
	ValidationSupportSamePosition          = "error.validation.support_same_position"
	ValidationUnknownAssignmentDestination = "error.validation.unknown_assignment_destination"
	ValidationAssignmentDestinationMissing = "error.validation.assignment_destination_missing"
	ValidationUnknownAssignmentNoble       = "error.validation.unknown_assignment_noble"
	ValidationDuplicateAssignmentNoble     = "error.validation.duplicate_assignment_noble"
	ValidationMultipleWildcards            = "error.validation.multiple_wildcards"

	AssignmentGameNil          = "error.assignment.game_nil"
	AssignmentInvalidState     = "error.assignment.invalid_state"
	AssignmentChainValidation  = "error.assignment.chain_validation"
	AssignmentNobleUnknown     = "error.assignment.noble_unknown"
	AssignmentNobleDungeon     = "error.assignment.noble_dungeon"
	AssignmentEmissionCapacity = "error.assignment.emission_capacity"
	AssignmentNoArmy           = "error.assignment.no_army"
	AssignmentArmyNotOwned     = "error.assignment.army_not_owned"
	AssignmentPendingDisperse  = "error.assignment.pending_disperse"
	AssignmentChainIDInUse     = "error.assignment.chain_id_in_use"
	ReceptionConcurrent        = "error.reception.concurrent"
)

var supportedTags = []language.Tag{language.English, language.French}

func init() {
	register(ErrorChainsInWinter, "chains cannot be submitted during winter", "les chaînes ne peuvent pas être soumises pendant l'hiver")
	register(ErrorWinterOutOfSeason, "winter orders can only be submitted during winter", "les ordres d'hiver ne peuvent être soumis qu'en hiver")
	register(ErrorUnknownPlayer, "player %q does not exist", "le joueur %q n'existe pas")
	register(ErrorPlayerRequired, "one player's orders must be submitted at a time", "les ordres d'un seul joueur doivent être soumis à la fois")
	register(ErrorForeignChain, "chain %d belongs to player %q", "la chaîne %d appartient au joueur %q")
	register(ErrorForeignWinter, "winter submission %d belongs to player %q", "la soumission d'hiver %d appartient au joueur %q")
	register(ErrorNobleUnknown, "chain header noble does not exist", "le noble de l'en-tête de chaîne n'existe pas")
	register(ErrorNobleMismatch, "submission noble %q does not match chain header %q", "le noble soumis %q ne correspond pas à l'en-tête de chaîne %q")
	register(ErrorNobleNotOwned, "noble %q belongs to player %q", "le noble %q appartient au joueur %q")
	register(ErrorDuplicateEmission, "noble %q appears more than once", "le noble %q apparaît plusieurs fois")

	register(ParseNoHeader, "an order chain requires a noble header", "une chaîne d'ordres doit commencer par un en-tête de noble")
	register(ParseBadHeader, "the first content line must contain exactly one noble code", "la première ligne de contenu doit contenir exactement un code de noble")
	register(ParseNobleNotFound, "noble code %q does not exist", "le code de noble %q n'existe pas")
	register(ParseOrderSymbolMissing, "an order line must contain an order symbol", "une ligne d'ordre doit contenir un symbole d'ordre")
	register(ParseUnclosedParenthesis, "parentheses must enclose one complete order line", "les parenthèses doivent entourer une ligne d'ordre complète")
	register(ParsePositionRequired, "%s requires a position code", "%s exige un code de position")
	register(ParsePositionOnlyOne, "%s accepts only one position code", "%s n'accepte qu'un seul code de position")
	register(ParseOrderPositionRequired, "an order line requires a position code and order symbol", "une ligne d'ordre exige un code de position et un symbole d'ordre")
	register(ParseUnsupportedOrderSymbol, "%q is not a supported order symbol", "%q n'est pas un symbole d'ordre pris en charge")
	register(ParseDestinationRequired, "%s requires one destination code", "%s exige un code de destination")
	register(ParseDestinationOnlyOne, "%s accepts exactly one destination code", "%s accepte exactement un code de destination")
	register(ParseSupportPositionRequired, "S requires a supported army position", "S exige la position d'une armée soutenue")
	register(ParseOffensiveSupportDestination, "offensive S requires an attack destination after -", "un S offensif exige une destination d'attaque après -")
	register(ParseSupportShape, "S accepts either one supported position or one position and attack destination", "S accepte une position soutenue, ou une position et une destination d'attaque")
	register(ParseOffensiveSupportDash, "offensive S requires - before its attack destination", "un S offensif exige - avant sa destination d'attaque")
	register(ParseDisperseDestinationRequired, "D requires at least one destination code", "D exige au moins un code de destination")
	register(ParseAssignmentDestination, "a dispersion assignment requires a destination code before *", "une affectation de dispersion exige un code de destination avant *")
	register(ParseAssignmentEmptyNoble, "a dispersion assignment contains an empty noble code", "une affectation de dispersion contient un code de noble vide")
	register(ParseAssignmentInvalidNoble, "invalid noble code %q in dispersion assignment", "code de noble %q invalide dans l'affectation de dispersion")
	register(ParseAssignmentUnknownNoble, "noble code %q does not exist", "le code de noble %q n'existe pas")
	register(ParseCodeFormat, "%s code %q must contain exactly three uppercase letters", "le code de %s %q doit contenir exactement trois lettres majuscules")
	register(ParseTerritoryUnknown, "territory code %q does not exist", "le code de territoire %q n'existe pas")

	register(WinterOrderShape, "a winter order requires a symbol, a subtype, and one target code", "un ordre d'hiver exige un symbole, un sous-type et un code cible")
	register(WinterOrderTargetOnlyOne, "a winter order accepts exactly one target code", "un ordre d'hiver accepte exactement un code cible")
	register(WinterUnknownSymbol, "unknown winter order symbol %q", "symbole d'ordre d'hiver inconnu : %q")
	register(WinterTerritoryTrigramFormat, "territory code %q must contain exactly three uppercase letters", "le code de territoire %q doit contenir exactement trois lettres majuscules")
	register(WinterTerritoryUnknown, "territory code %q does not exist", "le code de territoire %q n'existe pas")
	register(WinterNobleCodeFormat, "noble code %q must contain exactly three uppercase letters", "le code de noble %q doit contenir exactement trois lettres majuscules")
	register(WinterNobleUnknown, "noble code %q does not exist", "le code de noble %q n'existe pas")
	register(WinterUnknownSubtype, "unknown winter order %s %s", "ordre d'hiver inconnu : %s %s")

	register(ValidationUnknownNoble, "noble %q does not exist", "le noble %q n'existe pas")
	register(ValidationEmptyChain, "a chain must contain at least one order", "une chaîne doit contenir au moins un ordre")
	register(ValidationDuplicateOrder, "order id %q is duplicated", "l'identifiant d'ordre %q est dupliqué")
	register(ValidationInvalidOrderID, "an order must have an id", "un ordre doit avoir un identifiant")
	register(ValidationInvalidOrderType, "order type %q is invalid", "le type d'ordre %q est invalide")
	register(ValidationInvalidLiaison, "liaison %q is invalid", "le mode de liaison %q est invalide")
	register(ValidationUnexpectedNobleAssignments, "%s does not accept noble assignments", "%s n'accepte pas d'affectations de nobles")
	register(ValidationUnknownPosition, "position %q does not exist", "la position %q n'existe pas")
	register(ValidationUnknownTarget, "territory target %q does not exist", "la cible de territoire %q n'existe pas")
	register(ValidationJoinNotLast, "J must be the last order in a chain", "J doit être le dernier ordre d'une chaîne")
	register(ValidationUnexpectedTarget, "%s does not accept territory targets", "%s n'accepte pas de cibles de territoire")
	register(ValidationMissingTarget, "%s requires one territory target", "%s exige une cible de territoire")
	register(ValidationTooManyTargets, "%s accepts one territory target", "%s accepte une cible de territoire")
	register(ValidationNotAdjacent, "%s target %q is not adjacent to %q", "la cible %s %q n'est pas adjacente à %q")
	register(ValidationSupportSamePosition, "defensive S cannot support its own position", "un S défensif ne peut pas soutenir sa propre position")
	register(ValidationUnknownAssignmentDestination, "assignment destination %q does not exist", "la destination d'affectation %q n'existe pas")
	register(ValidationAssignmentDestinationMissing, "assignment destination %q is not listed in D targets", "la destination d'affectation %q n'est pas listée dans les cibles de D")
	register(ValidationUnknownAssignmentNoble, "assigned noble %q does not exist", "le noble affecté %q n'existe pas")
	register(ValidationDuplicateAssignmentNoble, "noble %q is assigned more than once", "le noble %q est affecté plusieurs fois")
	register(ValidationMultipleWildcards, "D accepts at most one remaining-nobles wildcard", "D accepte au plus un joker pour les nobles restants")

	register(AssignmentGameNil, "game state is nil", "l'état de partie est absent")
	register(AssignmentInvalidState, "game state is invalid: %s", "l'état de partie est invalide : %s")
	register(AssignmentChainValidation, "chain validation failed: %s", "la validation de la chaîne a échoué : %s")
	register(AssignmentNobleUnknown, "emitting noble %q does not exist", "le noble émetteur %q n'existe pas")
	register(AssignmentNobleDungeon, "noble %q is in a dungeon", "le noble %q est au donjon")
	register(AssignmentEmissionCapacity, "noble %q has already emitted in turn %d", "le noble %q a déjà émis au tour %d")
	register(AssignmentNoArmy, "no army occupies receiving position %q", "aucune armée n'occupe la position de réception %q")
	register(AssignmentArmyNotOwned, "army at receiving position %q belongs to %q, not emitting noble owner %q", "l'armée à la position de réception %q appartient à %q, et non au propriétaire du noble émetteur %q")
	register(AssignmentPendingDisperse, "army at receiving position %q executes pending dispersion for chain %q", "l'armée à la position de réception %q exécute une dispersion en attente pour la chaîne %q")
	register(AssignmentChainIDInUse, "next chain id %q is already in use", "le prochain identifiant de chaîne %q est déjà utilisé")
	register(ReceptionConcurrent, "concurrent reception: army at %q was targeted by %d chains in turn %d", "réception concurrente : l'armée en %q a été ciblée par %d chaînes au tour %d")
}

func register(key, english, french string) {
	message.SetString(language.English, key, english)
	message.SetString(language.French, key, french)
}

// Normalize accepts language tags such as "en-US" and "fr-FR" while keeping
// the public language surface deliberately limited to English and French.
func Normalize(value string) Language {
	value = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(value, "_", "-")))
	if strings.HasPrefix(value, "fr") {
		return French
	}
	return English
}

// FromRequest chooses an explicit query language first, then the browser's
// Accept-Language preference, and finally English.
func FromRequest(r *http.Request) Language {
	if languageValue := r.URL.Query().Get("lang"); languageValue != "" {
		return Normalize(languageValue)
	}
	tags, _, err := language.ParseAcceptLanguage(r.Header.Get("Accept-Language"))
	if err == nil && len(tags) != 0 {
		_, index, _ := language.NewMatcher(supportedTags).Match(tags...)
		if index >= 0 && index < len(supportedTags) {
			if supportedTags[index] == language.French {
				return French
			}
		}
	}
	return English
}

// Translate renders a catalog message in the requested language.
func Translate(languageValue Language, messageValue Message) string {
	return message.NewPrinter(tag(languageValue)).Sprintf(messageValue.Key, messageValue.Args...)
}

// EnglishText renders the canonical internal English form used by engine
// errors and diagnostics.
func EnglishText(messageValue Message) string {
	return Translate(English, messageValue)
}

func tag(languageValue Language) language.Tag {
	if languageValue == French {
		return language.French
	}
	return language.English
}
