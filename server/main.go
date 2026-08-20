package main

import (
	"context"
	"log"
	"net/http"

	"github.com/namnhatpham1995/Hootden/server/internal/config"
	"github.com/namnhatpham1995/Hootden/server/internal/db"
	"github.com/namnhatpham1995/Hootden/server/internal/httpapi"
	"github.com/namnhatpham1995/Hootden/server/internal/migrate"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if err := migrate.Up(cfg.DatabaseURL); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", httpapi.Healthz(pool))

	log.Printf("listening on :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, mux))
}
