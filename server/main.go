package main

import (
	"context"
	"log"
	"net/http"

	"github.com/namnhatpham1995/Hootden/server/internal/auth"
	"github.com/namnhatpham1995/Hootden/server/internal/config"
	"github.com/namnhatpham1995/Hootden/server/internal/db"
	"github.com/namnhatpham1995/Hootden/server/internal/httpapi"
	"github.com/namnhatpham1995/Hootden/server/internal/migrate"
	"github.com/namnhatpham1995/Hootden/server/internal/workspace"
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

	oauthConfig := auth.NewOAuthConfig(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURL)
	authHandlers := auth.Handlers{
		Pool:         pool,
		OAuthConfig:  oauthConfig,
		Exchanger:    auth.GoogleExchanger{OAuthConfig: oauthConfig},
		UserResolver: workspace.UserResolver{Pool: pool},
		AppOrigin:    cfg.AppOrigin,
		CookieDomain: cfg.CookieDomain,
	}
	requireAuth := auth.RequireAuth(pool, cfg.CookieDomain)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", httpapi.Healthz(pool))
	mux.HandleFunc("GET /auth/google/start", authHandlers.Start)
	mux.HandleFunc("GET /auth/google/callback", authHandlers.Callback)
	mux.HandleFunc("POST /auth/signout", authHandlers.SignOut)
	mux.Handle("GET /me", requireAuth(workspace.Me(pool)))
	// requireAuth wraps each further protected route as page endpoints are
	// added in later task groups.

	var handler http.Handler = mux
	handler = httpapi.MaxBytes(handler)
	handler = httpapi.CORS(cfg.AppOrigin, handler)

	log.Printf("listening on :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, handler))
}
