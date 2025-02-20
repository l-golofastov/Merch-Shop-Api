package main

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/l-golofastov/Merch-Shop-Api/internal/config"
	"github.com/l-golofastov/Merch-Shop-Api/internal/handlers/auth"
	"github.com/l-golofastov/Merch-Shop-Api/internal/handlers/buy"
	"github.com/l-golofastov/Merch-Shop-Api/internal/handlers/info"
	"github.com/l-golofastov/Merch-Shop-Api/internal/handlers/sendCoins"
	"github.com/l-golofastov/Merch-Shop-Api/internal/lib/api/jwtlib"
	"github.com/l-golofastov/Merch-Shop-Api/internal/lib/logger/sl"
	"github.com/l-golofastov/Merch-Shop-Api/internal/storage/postgres"
	"log/slog"
	"net/http"
	"os"
)

func main() {
	cfg := config.MustLoad()

	log := setupLogger()

	log.Info("starting app")

	storage, err := postgres.New(cfg.Postgres)
	if err != nil {
		log.Error("failed to init storage", sl.Err(err))
		os.Exit(1)
	}

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Post("/api/auth", auth.NewAuthHandler(log, storage))
	router.Get("/api/info", jwtlib.JWTMiddleware(info.NewInfoHandler(log, storage)))
	router.Post("/api/sendCoin", jwtlib.JWTMiddleware(sendCoins.NewCoinSenderHandler(log, storage)))
	router.Get("/api/buy/{item}", jwtlib.JWTMiddleware(buy.NewBuyerHandler(log, storage)))
	router.Get("/api/buy/", jwtlib.JWTMiddleware(buy.NewBuyerHandler(log, storage)))

	srv := &http.Server{
		Addr:         cfg.ServerAddress,
		Handler:      router,
		ReadTimeout:  cfg.Timeout,
		WriteTimeout: cfg.Timeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("failed to start server", sl.Err(err))
	}

	log.Error("shutting down")
}

func setupLogger() *slog.Logger {
	var log *slog.Logger

	log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	return log
}
