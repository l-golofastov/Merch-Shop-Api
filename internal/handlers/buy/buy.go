package buy

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/l-golofastov/Merch-Shop-Api/internal/handlers/authorize"
	"github.com/l-golofastov/Merch-Shop-Api/internal/lib/api/errresp"
	"github.com/l-golofastov/Merch-Shop-Api/internal/lib/logger/sl"
	"github.com/l-golofastov/Merch-Shop-Api/internal/storage"
	"log/slog"
	"net/http"
)

//go:generate go run github.com/vektra/mockery/v2@v2.52.2 --name=Buyer
type Buyer interface {
	CheckPassword(username, passwordHash string) (int, error)
	BuyItem(itemName string, userId int) error
}

func NewBuyerHandler(log *slog.Logger, b Buyer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.sendCoins.NewBuyerHandler"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		id, _, _ := authorize.Authorize(w, r, b)
		if id != 0 {
			log.Info("user authorized")
		} else {
			return
		}

		itemName := chi.URLParam(r, "item")
		if itemName == "" {
			log.Info("no item name provided")

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, errresp.Error("no item name provided"))

			return
		}

		err := b.BuyItem(itemName, id)
		if err != nil {
			if errors.Is(err, storage.ErrItemNotFound) {
				log.Info("item not found")

				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, errresp.Error("item not found"))

				return
			} else if errors.Is(err, storage.ErrNotEnoughCoins) {
				log.Info("not enough coins")

				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, errresp.Error("not enough coins to buy"))

				return
			}
			log.Error("failed to buy item", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, errresp.Error("failed to buy item"))

			return
		}

		render.Status(r, http.StatusOK)
	}
}
