package sendCoins

import (
	"errors"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/l-golofastov/Merch-Shop-Api/internal/handlers/authorize"
	"github.com/l-golofastov/Merch-Shop-Api/internal/lib/api/errresp"
	"github.com/l-golofastov/Merch-Shop-Api/internal/lib/logger/sl"
	"github.com/l-golofastov/Merch-Shop-Api/internal/storage"
	"log/slog"
	"net/http"
)

type SendCoinRequest struct {
	ToUser string `json:"toUser" validate:"required"`
	Amount int    `json:"amount" validate:"required,gt=0"`
}

//go:generate go run github.com/vektra/mockery/v2@v2.52.2 --name=CoinSender
type CoinSender interface {
	CheckPassword(username, passwordHash string) (int, error)
	SendCoins(username string, id, amount int) error
}

func NewCoinSenderHandler(log *slog.Logger, cs CoinSender) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.sendCoins.NewCoinSenderHandler"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		id, _, _ := authorize.Authorize(w, r, cs)
		if id != 0 {
			log.Info("user authorized")
		} else {
			return
		}

		var req SendCoinRequest

		err := render.DecodeJSON(r.Body, &req)
		if err != nil {
			log.Error("failed to decode request body", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, errresp.Error("failed to decode request body"))

			return
		}

		log.Info("request body decoded", slog.Any("request", req))

		if err = validator.New().Struct(req); err != nil {
			log.Error("failed to validate request body: not all required fields are provided", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, errresp.Error("failed to validate request body: not all required fields are provided or amount of coins <= 0"))

			return
		}

		err = cs.SendCoins(req.ToUser, id, req.Amount)
		if err != nil {
			if errors.Is(err, storage.ErrNotEnoughCoins) {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, errresp.Error("not enough coins to send"))

				return
			} else if errors.Is(err, storage.ErrUserNotFound) {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, errresp.Error("user does not exist"))

				return
			}
			log.Error("failed to send coins", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, errresp.Error("internal server error"))

			return
		}

		render.Status(r, http.StatusOK)
	}
}
