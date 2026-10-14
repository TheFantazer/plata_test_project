package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TheFantazer/plata_test_project/internal/provider/frankfurter"
	"github.com/TheFantazer/plata_test_project/internal/repository/postgres"
	"github.com/TheFantazer/plata_test_project/internal/service"
	httptransport "github.com/TheFantazer/plata_test_project/internal/transport/http"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()

	pingCtx, cancelPing := context.WithTimeout(context.Background(), 5*time.Second)
	err = pool.Ping(pingCtx)
	cancelPing()
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}

	repository := postgres.NewRepository(pool)

	requeueCtx, cancelRequeue := context.WithTimeout(context.Background(), 5*time.Second)
	requeued, err := repository.RequeueProcessing(requeueCtx)
	cancelRequeue()
	if err != nil {
		return fmt.Errorf("recover interrupted quote updates: %w", err)
	}
	if requeued > 0 {
		log.Printf("requeued %d interrupted quote updates", requeued)
	}

	quoteService := service.NewQuoteService(repository)
	handler := httptransport.NewHandler(quoteService)

	rateProvider, err := frankfurter.NewClient(
		&http.Client{Timeout: 10 * time.Second},
		"https://api.frankfurter.dev",
	)
	if err != nil {
		return fmt.Errorf("create rate provider: %w", err)
	}

	worker := service.NewWorker(repository, rateProvider)

	server := &http.Server{
		Addr:              addr,
		Handler:           httptransport.NewRouter(handler),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	serverError := make(chan error, 1)
	workerError := make(chan error, 1)

	go func() {
		log.Printf("HTTP server listening on %s", addr)
		serverError <- server.ListenAndServe()
	}()

	go func() {
		workerError <- worker.Run(ctx)
	}()

	var runError error
	workerFinished := false

	select {
	case <-ctx.Done():

	case err := <-serverError:
		if !errors.Is(err, http.ErrServerClosed) {
			runError = fmt.Errorf("HTTP server: %w", err)
		}

	case err := <-workerError:
		workerFinished = true
		if err != nil {
			runError = fmt.Errorf("quote worker: %w", err)
		} else if ctx.Err() == nil {
			runError = errors.New("quote worker stopped unexpectedly")
		}
	}

	stop()

	shutdownCtx, cancelShutdown := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	shutdownError := server.Shutdown(shutdownCtx)
	cancelShutdown()

	if !workerFinished {
		if err := <-workerError; err != nil && runError == nil {
			runError = fmt.Errorf("quote worker: %w", err)
		}
	}

	if runError != nil {
		return runError
	}
	if shutdownError != nil {
		return fmt.Errorf("shut down HTTP server: %w", shutdownError)
	}

	return nil
}
