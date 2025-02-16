package info

import (
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/l-golofastov/Merch-Shop-Api/internal/handlers"
	"github.com/l-golofastov/Merch-Shop-Api/internal/lib/api/errresp"
	"github.com/l-golofastov/Merch-Shop-Api/internal/lib/logger/sl"
	"log/slog"
	"net/http"
)

type InfoResponse struct {
	Coins       int `json:"coins"`
	Inventory   `json:"inventory"`
	CoinHistory `json:"coinHistory"`
}

type Inventory struct {
	Items []InventoryCell
}

type InventoryCell struct {
	Type     string `json:"type"`
	Quantity int    `json:"quantity"`
}

type CoinHistory struct {
	Received []ReceivedCoins `json:"received"`
	Sent     []SentCoins     `json:"sent"`
}

type ReceivedCoins struct {
	FromUser string `json:"fromUser"`
	Amount   int    `json:"amount"`
}

type SentCoins struct {
	ToUser string `json:"toUser"`
	Amount int    `json:"amount"`
}

type InfoHandler interface {
	CheckPassword(username, passwordHash string) (int, error)
	GetUserCoins(id int) (int, error)
	GetUserPurchases(id int) (map[string]int, error)
	GetUserReceivedCoins(id int) (map[string]int, error)
	GetUserSentCoins(id int) (map[string]int, error)
}

func NewInfoHandler(log *slog.Logger, ih InfoHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.info.NewInfoHandler"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		id, _, _ := handlers.Authorize(w, r, ih)

		log.Info("user authorized")

		coins, err := ih.GetUserCoins(id)
		if err != nil {
			log.Error("failed to get user coins", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, errresp.Error("failed to get user coins"))

			return
		}

		purchases, err := ih.GetUserPurchases(id)
		if err != nil {
			log.Error("failed to get user purchases", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, errresp.Error("failed to get user purchases"))

			return
		}

		recieves, err := ih.GetUserReceivedCoins(id)
		if err != nil {
			log.Error("failed to get user received coins", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, errresp.Error("failed to get user received coins"))

			return
		}

		sends, err := ih.GetUserSentCoins(id)
		if err != nil {
			log.Error("failed to get user sent coins", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, errresp.Error("failed to get user sent coins"))

			return
		}

		itemsList := make([]InventoryCell, 0)
		for item, quantity := range purchases {
			itemsList = append(itemsList, InventoryCell{Type: item, Quantity: quantity})
		}

		inventory := Inventory{Items: itemsList}

		receivedCoins := make([]ReceivedCoins, 0)
		for fromUser, amount := range recieves {
			receivedCoins = append(receivedCoins, ReceivedCoins{FromUser: fromUser, Amount: amount})
		}

		sentCoins := make([]SentCoins, 0)
		for toUser, amount := range sends {
			sentCoins = append(sentCoins, SentCoins{ToUser: toUser, Amount: amount})
		}

		coinHistory := CoinHistory{Received: receivedCoins, Sent: sentCoins}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, InfoResponse{Coins: coins, Inventory: inventory, CoinHistory: coinHistory})
	}
}
