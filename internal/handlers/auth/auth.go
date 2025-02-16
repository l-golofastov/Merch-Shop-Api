package auth

import (
	"errors"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/l-golofastov/Merch-Shop-Api/internal/lib/api/errresp"
	"github.com/l-golofastov/Merch-Shop-Api/internal/lib/api/jwtlib"
	"github.com/l-golofastov/Merch-Shop-Api/internal/lib/logger/sl"
	"github.com/l-golofastov/Merch-Shop-Api/internal/storage"
	"golang.org/x/crypto/bcrypt"
	"log/slog"
	"net/http"
)

type AuthRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type AuthResponse struct {
	Token string `json:"token"`
}

type AuthHandler interface {
	CreateUser(username, passwordHash string) (int, error)
	FindUserByUsername(username string) (int, error)
	CheckPassword(username, passwordHash string) (int, error)
}

func NewAuthHandler(log *slog.Logger, ah AuthHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.auth.NewAuthHandler"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req AuthRequest

		err := render.DecodeJSON(r.Body, &req)
		if err != nil {
			log.Error("failed to decode request body", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, errresp.Error("failed to decode request body"))

			return
		}

		log.Info("request body decoded", slog.Any("request", req))

		if err = validator.New().Struct(req); err != nil {
			log.Error("failed to validate request: not all required fields are provided", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, errresp.Error("failed to validate request: not all required fields are provided"))

			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Error("failed to get password hash", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, errresp.Error("failed to get password hash"))

			return
		}

		passwordHash := string(hashedPassword)

		id, err := ah.FindUserByUsername(req.Username)
		if err != nil && !errors.Is(err, storage.ErrUserNotFound) {
			log.Error("failed to find user by username", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, errresp.Error("failed to find user by username"))

			return
		}

		if id != -1 {
			id, err = ah.CheckPassword(req.Username, passwordHash)
			if err != nil {
				if errors.Is(err, storage.ErrPasswordUnmatched) {
					render.Status(r, http.StatusUnauthorized)
					render.JSON(w, r, errresp.Error("invalid password"))

					return
				}
				log.Error("failed to check user password", sl.Err(err))

				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, errresp.Error("failed to check user password"))

				return
			}
		} else {
			id, err = ah.CreateUser(req.Username, passwordHash)
			if err != nil {
				log.Error("failed to create user", sl.Err(err))

				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, errresp.Error("failed to create user"))

				return
			}
		}

		token, err := jwtlib.GenerateJWT(id, req.Username, passwordHash)
		if err != nil {
			log.Error("failed to generate JWT token", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, errresp.Error("failed to generate JWT token"))

			return
		}

		log.Info("user authenticated")

		render.Status(r, http.StatusOK)
		render.JSON(w, r, AuthResponse{Token: token})
	}
}
