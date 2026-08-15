package firestorestore

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	cloudfirestore "cloud.google.com/go/firestore"

	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/models"
	"github.com/fogfactory/crown-and-borough/internal/store"
)

const schemaVersion = 1

type playerDocument struct {
	ID      models.PlayerID `firestore:"id"`
	Name    string          `firestore:"name"`
	Color   string          `firestore:"color"`
	ActorID string          `firestore:"actorId,omitempty"`
}

type gameDocument struct {
	SchemaVersion int              `firestore:"schemaVersion"`
	ID            store.GameID     `firestore:"id"`
	Name          string           `firestore:"name"`
	Seed          string           `firestore:"seed"`
	OwnerUID      string           `firestore:"ownerUid"`
	MemberUIDs    []string         `firestore:"memberUids"`
	Players       []playerDocument `firestore:"players"`
	Status        store.Status     `firestore:"status"`
	Turn          int              `firestore:"turn"`
	Season        models.Season    `firestore:"season"`
	WinnerUID     string           `firestore:"winnerUid,omitempty"`
	SubmittedUIDs []string         `firestore:"submittedUids"`
	Revision      int64            `firestore:"revision"`
	CreatedAt     time.Time        `firestore:"createdAt"`
	UpdatedAt     time.Time        `firestore:"updatedAt"`
}

type canonicalDocument struct {
	SchemaVersion int                    `firestore:"schemaVersion"`
	ID            store.GameID           `firestore:"id"`
	Seed          string                 `firestore:"seed"`
	Turn          int                    `firestore:"turn"`
	Revision      int64                  `firestore:"revision"`
	State         map[string]interface{} `firestore:"state"`
	MapJSON       string                 `firestore:"mapJson"`
	SubmittedUIDs []string               `firestore:"submittedUids"`
	Resolution    *resolutionClaim       `firestore:"resolution,omitempty"`
	UpdatedAt     time.Time              `firestore:"updatedAt"`
}

type resolutionClaim struct {
	OperationID  string    `firestore:"operationId"`
	ClaimedAt    time.Time `firestore:"claimedAt"`
	LeaseUntil   time.Time `firestore:"leaseUntil"`
	BaseRevision int64     `firestore:"baseRevision"`
	Turn         int       `firestore:"turn"`
}

type submissionDocument struct {
	SchemaVersion int       `firestore:"schemaVersion"`
	UID           string    `firestore:"uid"`
	PlayerID      string    `firestore:"playerId"`
	Turn          int       `firestore:"turn"`
	OrdersJSON    string    `firestore:"ordersJson"`
	SubmittedAt   time.Time `firestore:"submittedAt"`
}

type reportDocument struct {
	SchemaVersion int                    `firestore:"schemaVersion"`
	GameID        store.GameID           `firestore:"gameId"`
	Turn          int                    `firestore:"turn"`
	Report        map[string]interface{} `firestore:"report"`
	Privacy       map[string]interface{} `firestore:"privacy,omitempty"`
	CreatedAt     time.Time              `firestore:"createdAt"`
}

type viewDocument struct {
	SchemaVersion int                    `firestore:"schemaVersion"`
	GameID        store.GameID           `firestore:"gameId"`
	UID           string                 `firestore:"uid"`
	Revision      int64                  `firestore:"revision"`
	Turn          int                    `firestore:"turn"`
	Season        models.Season          `firestore:"season"`
	State         map[string]interface{} `firestore:"state"`
	UpdatedAt     time.Time              `firestore:"updatedAt"`
}

type filteredReportDocument struct {
	SchemaVersion int                    `firestore:"schemaVersion"`
	GameID        store.GameID           `firestore:"gameId"`
	UID           string                 `firestore:"uid"`
	Revision      int64                  `firestore:"revision"`
	Turn          int                    `firestore:"turn"`
	Season        models.Season          `firestore:"season"`
	Report        map[string]interface{} `firestore:"report"`
	UpdatedAt     time.Time              `firestore:"updatedAt"`
}

type profileDocument struct {
	SchemaVersion int       `firestore:"schemaVersion"`
	UID           string    `firestore:"uid"`
	Email         string    `firestore:"email"`
	DisplayName   string    `firestore:"displayName"`
	CreatedAt     time.Time `firestore:"createdAt"`
	UpdatedAt     time.Time `firestore:"updatedAt"`
}

type invitationDocument struct {
	SchemaVersion int          `firestore:"schemaVersion"`
	GameID        store.GameID `firestore:"gameId"`
	CreatedBy     string       `firestore:"createdBy"`
	CodeHash      string       `firestore:"codeHash"`
	CreatedAt     time.Time    `firestore:"createdAt"`
	Active        bool         `firestore:"active"`
}

func gameRef(client *cloudfirestore.Client, id store.GameID) *cloudfirestore.DocumentRef {
	return client.Collection("games").Doc(string(id))
}

func canonicalRef(client *cloudfirestore.Client, id store.GameID) *cloudfirestore.DocumentRef {
	return gameRef(client, id).Collection("canonical").Doc("current")
}

func submissionCollection(client *cloudfirestore.Client, id store.GameID, turn int) *cloudfirestore.CollectionRef {
	return gameRef(client, id).Collection("turns").Doc(strconv.Itoa(turn)).Collection("submissions")
}

func reportCollection(client *cloudfirestore.Client, id store.GameID) *cloudfirestore.CollectionRef {
	return gameRef(client, id).Collection("reports")
}

func viewRef(client *cloudfirestore.Client, id store.GameID, uid string) *cloudfirestore.DocumentRef {
	return gameRef(client, id).Collection("views").Doc(uid)
}

func filteredReportRef(client *cloudfirestore.Client, id store.GameID, uid string, turn int) *cloudfirestore.DocumentRef {
	return gameRef(client, id).Collection("reports").Doc(uid).Collection("turns").Doc(strconv.Itoa(turn))
}

func invitationRef(client *cloudfirestore.Client, codeHash string) *cloudfirestore.DocumentRef {
	return client.Collection("invitations").Doc(strings.ToLower(strings.TrimSpace(codeHash)))
}

func jsonMap(value interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	if result == nil {
		result = make(map[string]interface{})
	}
	return result, nil
}

func decodeJSONMap(source map[string]interface{}, target interface{}) error {
	data, err := json.Marshal(source)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func jsonString(value interface{}) (string, error) {
	data, err := json.Marshal(value)
	return string(data), err
}

func decodeJSONString(source string, target interface{}) error {
	return json.Unmarshal([]byte(source), target)
}

func ordersJSON(input engine.OrdersInput) (string, error) {
	data, err := json.Marshal(input)
	return string(data), err
}

func decodeOrdersJSON(value string) (engine.OrdersInput, error) {
	var input engine.OrdersInput
	if err := json.Unmarshal([]byte(value), &input); err != nil {
		return engine.OrdersInput{}, err
	}
	return input, nil
}

func decodeGameDocument(snapshot *cloudfirestore.DocumentSnapshot) (gameDocument, error) {
	var document gameDocument
	if err := snapshot.DataTo(&document); err != nil {
		return gameDocument{}, fmt.Errorf("decode game %q: %w", snapshot.Ref.ID, err)
	}
	if document.SchemaVersion != schemaVersion {
		return gameDocument{}, fmt.Errorf("%w: games/%s has version %d, want %d", ErrSchemaVersion, snapshot.Ref.ID, document.SchemaVersion, schemaVersion)
	}
	return document, nil
}

func decodeCanonicalDocument(snapshot *cloudfirestore.DocumentSnapshot) (canonicalDocument, error) {
	var document canonicalDocument
	if err := snapshot.DataTo(&document); err != nil {
		return canonicalDocument{}, fmt.Errorf("decode canonical document: %w", err)
	}
	if document.SchemaVersion != schemaVersion {
		return canonicalDocument{}, fmt.Errorf("%w: canonical has version %d, want %d", ErrSchemaVersion, document.SchemaVersion, schemaVersion)
	}
	return document, nil
}

func decodeSubmissionDocument(snapshot *cloudfirestore.DocumentSnapshot) (submissionDocument, error) {
	var document submissionDocument
	if err := snapshot.DataTo(&document); err != nil {
		return submissionDocument{}, fmt.Errorf("decode submission %q: %w", snapshot.Ref.Path, err)
	}
	if document.SchemaVersion != schemaVersion {
		return submissionDocument{}, fmt.Errorf("%w: submission %s has version %d, want %d", ErrSchemaVersion, snapshot.Ref.Path, document.SchemaVersion, schemaVersion)
	}
	return document, nil
}

func decodeReportDocument(snapshot *cloudfirestore.DocumentSnapshot) (reportDocument, error) {
	var document reportDocument
	if err := snapshot.DataTo(&document); err != nil {
		return reportDocument{}, fmt.Errorf("decode report %q: %w", snapshot.Ref.Path, err)
	}
	if document.SchemaVersion != schemaVersion {
		return reportDocument{}, fmt.Errorf("%w: report %s has version %d, want %d", ErrSchemaVersion, snapshot.Ref.Path, document.SchemaVersion, schemaVersion)
	}
	return document, nil
}

func decodeProfileDocument(snapshot *cloudfirestore.DocumentSnapshot) (profileDocument, error) {
	var document profileDocument
	if err := snapshot.DataTo(&document); err != nil {
		return profileDocument{}, fmt.Errorf("decode profile %q: %w", snapshot.Ref.ID, err)
	}
	if document.SchemaVersion != schemaVersion {
		return profileDocument{}, fmt.Errorf("%w: players/%s has version %d, want %d", ErrSchemaVersion, snapshot.Ref.ID, document.SchemaVersion, schemaVersion)
	}
	return document, nil
}

func decodeInvitationDocument(snapshot *cloudfirestore.DocumentSnapshot) (invitationDocument, error) {
	var document invitationDocument
	if err := snapshot.DataTo(&document); err != nil {
		return invitationDocument{}, fmt.Errorf("decode invitation %q: %w", snapshot.Ref.ID, err)
	}
	if document.SchemaVersion != schemaVersion {
		return invitationDocument{}, fmt.Errorf("%w: invitations/%s has version %d, want %d", ErrSchemaVersion, snapshot.Ref.ID, document.SchemaVersion, schemaVersion)
	}
	return document, nil
}
