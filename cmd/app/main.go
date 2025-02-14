package main

import (
	"github.com/l-golofastov/Merch-Shop-Api/internal/config"
	"github.com/l-golofastov/Merch-Shop-Api/internal/storage/postgres"
	"log/slog"
	"os"
)

func main() {
	cfg := config.MustLoad()

	log := setupLogger()

	log.Info("starting app")

	_, err := postgres.New(cfg.StorageConnStr)
	if err != nil {
		log.Error("failed to init storage") // TODO: Add more verbose message
		os.Exit(1)
	}
}

func setupLogger() *slog.Logger {
	var log *slog.Logger

	log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	return log
}
