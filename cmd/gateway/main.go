package main

import (
	"ecomGateway/internal/config"
	ordergrpc "ecomGateway/internal/grpc/order"
	productgrpc "ecomGateway/internal/grpc/product"
	usergrpc "ecomGateway/internal/grpc/user"
	httphandler "ecomGateway/internal/http_handler"
	"ecomGateway/internal/processor"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)

	log.Info("starting url-shortener")
	log.Debug("debug messages are enabled")

	userClient, err := usergrpc.New(log, cfg.UserTarget, cfg.UserTimeout, cfg.UserRetries)

	if err != nil {
		log.Error("failed to init user client", "err", err)
		os.Exit(1)
	}

	orderClient, err := ordergrpc.New(log, cfg.OrderTarget, cfg.OrderTimeout, cfg.UserRetries)

	if err != nil {
		log.Error("failed to init order client", "err", err)
		os.Exit(1)
	}

	productClient, err := productgrpc.New(log, cfg.ProductTarget, cfg.ProductTimeout, cfg.ProductRetries)

	if err != nil {
		log.Error("failed to init product client", "err", err)
		os.Exit(1)
	}

	processor := processor.NewProcessorService(*userClient, *orderClient, *productClient)

	httphandler := httphandler.NewHTTPHandler(processor, log)

	router := chi.NewRouter()

	httphandler.RegisterRoutes(router)

	log.Info("starting server", slog.String("address", cfg.HttpAddress))

	srv := &http.Server{
		Addr:         cfg.HttpAddress,
		Handler:      router,
		ReadTimeout:  cfg.HttpTimeout,
		WriteTimeout: cfg.HttpTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("failed to start server", slog.String("error", err.Error()))
		os.Exit(1)
	}
	log.Info("server stopped")

}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	default:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}
