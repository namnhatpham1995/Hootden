package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/namnhatpham1995/Hootden/server/internal/auth"
	"github.com/namnhatpham1995/Hootden/server/internal/config"
	"github.com/namnhatpham1995/Hootden/server/internal/db"
	"github.com/namnhatpham1995/Hootden/server/internal/httpapi"
	"github.com/namnhatpham1995/Hootden/server/internal/migrate"
	"github.com/namnhatpham1995/Hootden/server/internal/page"
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
	passwordHandlers := auth.PasswordHandlers{
		Pool:         pool,
		Store:        workspace.PasswordStore{Pool: pool},
		CookieDomain: cfg.CookieDomain,
	}
	requireAuth := auth.RequireAuth(pool, cfg.CookieDomain)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", httpapi.Healthz(pool))
	mux.HandleFunc("GET /auth/config", auth.Config(cfg.GoogleClientID != ""))
	mux.HandleFunc("GET /auth/google/start", authHandlers.Start)
	mux.HandleFunc("GET /auth/google/callback", authHandlers.Callback)
	mux.HandleFunc("POST /auth/register", passwordHandlers.Register)
	mux.HandleFunc("POST /auth/login", passwordHandlers.Login)
	mux.HandleFunc("POST /auth/signout", authHandlers.SignOut)
	mux.Handle("GET /me", requireAuth(workspace.Me(pool)))

	pageHandlers := page.Handlers{Pool: pool}
	mux.Handle("GET /pages", requireAuth(http.HandlerFunc(pageHandlers.List)))
	mux.Handle("POST /pages", requireAuth(http.HandlerFunc(pageHandlers.Create)))
	mux.Handle("GET /pages/{id}", requireAuth(http.HandlerFunc(pageHandlers.Get)))
	mux.Handle("PATCH /pages/{id}", requireAuth(http.HandlerFunc(pageHandlers.Update)))
	mux.Handle("DELETE /pages/{id}", requireAuth(http.HandlerFunc(pageHandlers.Delete)))

	var handler http.Handler = mux
	handler = httpapi.MaxBytes(handler)
	handler = httpapi.CORS(cfg.AppOrigin, handler)

	serverCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	reaperDone := auth.StartSessionReaper(serverCtx, pool, auth.SessionReapInterval, log.Default())

	server := &http.Server{Addr: ":" + cfg.Port, Handler: handler}
	go func() {
		<-serverCtx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("server shutdown: %v", err)
		}
	}()

	log.Printf("listening on :%s", cfg.Port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		stop()
		<-reaperDone
		log.Fatalf("listen: %v", err)
	}
	stop()
	<-reaperDone
}
