package info

import (
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/l-golofastov/Merch-Shop-Api/internal/handlers/authorize"
	"github.com/l-golofastov/Merch-Shop-Api/internal/lib/api/errresp"
	"github.com/l-golofastov/Merch-Shop-Api/internal/lib/datatype/pair"
	"github.com/l-golofastov/Merch-Shop-Api/internal/lib/logger/sl"
	"log/slog"
	"net/http"
)

type InfoResponse struct {
	Coins       int             `json:"coins"`
	Inventory   []InventoryCell `json:"inventory"`
	CoinHistory `json:"coinHistory"`
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

//go:generate go run github.com/vektra/mockery/v2@v2.52.2 --name=InfoHandler
type InfoHandler interface {
	CheckPassword(username, passwordHash string) (int, error)
	GetUserCoins(id int) (int, error)
	GetUserPurchases(id int) (map[string]int, error)
	GetUserReceivedCoins(id int) ([]pair.Pair[string, int], error)
	GetUserSentCoins(id int) ([]pair.Pair[string, int], error)
}

func NewInfoHandler(log *slog.Logger, ih InfoHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.info.NewInfoHandler"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		id, _, _ := authorize.Authorize(w, r, ih)
		if id != 0 {
			log.Info("user authorized")
		} else {
			return
		}

		coins, err := ih.GetUserCoins(id)
		if err != nil {
			log.Error("failed to get user coins", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, errresp.Error("internal server error"))

			return
		}

		purchases, err := ih.GetUserPurchases(id)
		if err != nil {
			log.Error("failed to get user purchases", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, errresp.Error("internal server error"))

			return
		}

		receives, err := ih.GetUserReceivedCoins(id)
		if err != nil {
			log.Error("failed to get user received coins", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, errresp.Error("internal server error"))

			return
		}

		sends, err := ih.GetUserSentCoins(id)
		if err != nil {
			log.Error("failed to get user sent coins", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, errresp.Error("internal server error"))

			return
		}

		itemsList := make([]InventoryCell, 0)
		for item, quantity := range purchases {
			itemsList = append(itemsList, InventoryCell{Type: item, Quantity: quantity})
		}

		receivedCoins := make([]ReceivedCoins, 0)
		for _, elem := range receives {
			fromUser := elem.Fst
			amount := elem.Snd
			receivedCoins = append(receivedCoins, ReceivedCoins{FromUser: fromUser, Amount: amount})
		}

		sentCoins := make([]SentCoins, 0)
		for _, elem := range sends {
			toUser := elem.Fst
			amount := elem.Snd
			sentCoins = append(sentCoins, SentCoins{ToUser: toUser, Amount: amount})
		}

		coinHistory := CoinHistory{Received: receivedCoins, Sent: sentCoins}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, InfoResponse{Coins: coins, Inventory: itemsList, CoinHistory: coinHistory})
	}
}
