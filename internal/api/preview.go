package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"github.com/fogfactory/crown-and-borough/internal/engine"
	"github.com/fogfactory/crown-and-borough/internal/i18n"
	"github.com/fogfactory/crown-and-borough/internal/models"
	"github.com/fogfactory/crown-and-borough/internal/store"
	"github.com/fogfactory/crown-and-borough/internal/turn"
)

// Winter line statuses of an orders preview.
const (
	winterLineApplied  = "applied"
	winterLineRejected = "rejected"
	winterLineInvalid  = "invalid"
	winterLineDiscard  = "discard"
)

// OrdersPreviewView is the dry run of a draft returned to the client while
// the player types, so the client never re-implements the order rules.
type OrdersPreviewView struct {
	Errors     []engine.InputError     `json:"errors"`
	Chains     []ChainPreviewView      `json:"chains"`
	Winter     []WinterLinePreviewView `json:"winter"`
	WinterCost *WinterCostPreviewView  `json:"winterCost,omitempty"`
}

// ChainPreviewView lists the order lines of one drafted chain that parse.
type ChainPreviewView struct {
	Noble  models.NobleCode `json:"noble"`
	Orders []OrderView      `json:"orders"`
}

// WinterLinePreviewView is the simulated outcome of one winter line. Reason
// is an engine rejection reason; Message explains a line that does not parse.
type WinterLinePreviewView struct {
	Line           int                    `json:"line"`
	Status         string                 `json:"status"`
	Type           models.WinterOrderType `json:"type,omitempty"`
	Territory      models.TerritoryID     `json:"territory,omitempty"`
	Source         models.TerritoryID     `json:"source,omitempty"`
	Target         models.TerritoryID     `json:"target,omitempty"`
	Amount         int                    `json:"amount,omitempty"`
	Infrastructure models.InfraType       `json:"infrastructure,omitempty"`
	Level          int                    `json:"level,omitempty"`
	Noble          models.NobleCode       `json:"noble,omitempty"`
	Cost           int                    `json:"cost,omitempty"`
	Reason         string                 `json:"reason,omitempty"`
	Message        string                 `json:"message,omitempty"`
}

// WinterCostPreviewView compares what the winter sheet spends with the
// player's payment reserves.
type WinterCostPreviewView struct {
	Spent     int `json:"spent"`
	Available int `json:"available"`
}

func (h *GamesHandler) preview(w http.ResponseWriter, r *http.Request, actor store.Actor, id store.GameID) {
	var request gameOrdersRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_orders_request", err.Error())
		return
	}
	snapshot, err := h.store.State(r.Context(), actor, id)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	playerID, ok := snapshot.PlayerFor(actor)
	if !ok {
		writeAPIError(w, http.StatusForbidden, "not_member", "only a player of this game can preview orders")
		return
	}
	input, err := turn.NormalizeSubmission(playerID, engine.OrdersInput{
		Chains:  toChainSubmissions(request.Chains),
		Winter:  toWinterSubmissions(request.Winter),
		Special: toDeckSubmissions(request.Special),
	})
	var inputErrors *engine.InputErrors
	if errors.As(err, &inputErrors) {
		writeResolutionError(w, err, i18n.FromRequest(r))
		return
	}
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	view, err := previewView(snapshot.State, h.balance, playerID, input, i18n.FromRequest(r))
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func previewView(state *models.GameState, balance assetgen.Balance, playerID models.PlayerID, input engine.OrdersInput, language i18n.Language) (OrdersPreviewView, error) {
	preview, err := engine.PreviewOrders(state, balance, playerID, input)
	if err != nil {
		return OrdersPreviewView{}, err
	}
	view := OrdersPreviewView{
		Errors: make([]engine.InputError, len(preview.Errors)),
		Chains: make([]ChainPreviewView, 0, len(preview.Chains)),
		Winter: make([]WinterLinePreviewView, 0, len(preview.Winter)),
	}
	for index, inputError := range preview.Errors {
		view.Errors[index] = localizedInputError(inputError, language)
	}
	for _, chain := range preview.Chains {
		orders := make([]OrderView, 0, len(chain.Orders))
		for _, order := range chain.Orders {
			orders = append(orders, projectOrder(order))
		}
		view.Chains = append(view.Chains, ChainPreviewView{Noble: chain.Noble, Orders: orders})
	}
	for _, line := range preview.Winter {
		view.Winter = append(view.Winter, winterLineView(line, language))
	}
	if preview.WinterCost != nil {
		view.WinterCost = &WinterCostPreviewView{Spent: preview.WinterCost.Spent, Available: preview.WinterCost.Available}
	}
	return view, nil
}

func winterLineView(line engine.WinterLinePreview, language i18n.Language) WinterLinePreviewView {
	view := WinterLinePreviewView{Line: line.Line}
	switch {
	case line.ParseError != nil:
		view.Status = winterLineInvalid
		view.Message = localizedInputError(*line.ParseError, language).Message
	case line.Discard != nil:
		view.Status = winterLineDiscard
	case line.Order != nil:
		order := line.Order
		view.Status = winterLineRejected
		if line.Applied {
			view.Status = winterLineApplied
		}
		view.Type = order.Type
		view.Territory = line.Territory
		view.Source = order.SourceID
		view.Target = order.TargetID
		view.Amount = order.Amount
		view.Infrastructure = order.InfraType
		view.Level = line.Level
		view.Noble = order.NobleCode
		view.Cost = line.Cost
		view.Reason = line.Reason
	}
	return view
}

func localizedInputError(inputError engine.InputError, language i18n.Language) engine.InputError {
	if inputError.MessageKey != "" {
		inputError.Message = i18n.Translate(language, i18n.Message{Key: inputError.MessageKey, Args: inputError.MessageArgs})
	}
	return inputError
}
