package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/docbiner/docbiner/internal/config"
	"github.com/docbiner/docbiner/internal/database"
	"github.com/docbiner/docbiner/internal/delivery"
	"github.com/docbiner/docbiner/internal/queue"
	"github.com/docbiner/docbiner/internal/renderer"
	"github.com/docbiner/docbiner/internal/storage"
)

func main() {
	log.Println("Docbiner Worker starting...")

	// 1. Load config.
	cfg := config.Load()

	// 2. Connect to PostgreSQL.
	ctx := context.Background()
	db, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Connected to PostgreSQL")

	// 3. Connect to NATS and create queue subscriber.
	q, err := queue.New(cfg.NatsURL)
	if err != nil {
		log.Fatalf("failed to connect to NATS: %v", err)
	}
	defer q.Close()
	log.Println("Connected to NATS")

	// 4. Initialize Chromium renderer.
	r, err := renderer.New()
	if err != nil {
		log.Fatalf("failed to initialize renderer: %v", err)
	}
	defer r.Close()
	log.Println("Renderer initialized")

	// 5. Initialize Minio storage.
	store, err := storage.New(cfg.MinioEndpoint, cfg.MinioAccessKey, cfg.MinioSecretKey, cfg.MinioBucket, false)
	if err != nil {
		log.Fatalf("failed to initialize storage: %v", err)
	}
	if err := store.EnsureBucket(ctx); err != nil {
		log.Fatalf("failed to ensure storage bucket: %v", err)
	}
	log.Println("Storage initialized (bucket: " + cfg.MinioBucket + ")")

	// 6. Initialize delivery dispatcher.
	webhookSender := delivery.NewWebhookSender(nil)
	s3Deliverer := &delivery.S3Deliverer{}
	dispatcher := newDeliveryDispatcher(webhookSender, s3Deliverer)
	log.Println("Delivery dispatcher initialized")

	// 7. Wire up handler with all dependencies.
	handler := NewJobHandler(
		db.Jobs,                // JobStore
		&rendererAdapter{r: r}, // Renderer
		store,                  // StorageUploader
		dispatcher,             // DeliveryDispatcher
	)

	// 8. Start processing loop in a goroutine.
	subCtx, subCancel := context.WithCancel(ctx)
	defer subCancel()

	errCh := make(chan error, 1)
	go func() {
		log.Println("Worker ready, waiting for jobs...")
		errCh <- q.Subscribe(subCtx, handler.Handle)
	}()

	// 9. Graceful shutdown on SIGINT/SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Printf("Received signal %v, shutting down...", sig)
		subCancel()
	case err := <-errCh:
		if err != nil {
			log.Printf("Subscribe error: %v", err)
		}
	}

	log.Println("Worker shut down")
}
