package server

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"screw/auth"
	"screw/herr"
	mw "screw/middleware"
	"screw/session"
	"screw/store"
	"screw/ws"
	"sync"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type server struct {
	addr            string
	clientId        string
	clientSecret    string
	env             string
	store           store.Store
	sessionManager  *session.Manager
	ws              *ws.WS
	google          *auth.Google
	CORSAllowed     map[string]bool
	protectedRoutes map[string]bool
}

type ServerCfg struct {
	Addr         string
	ClientId     string
	ClientSecret string
	Env          string
	DBPath       string
}

func New(cfg ServerCfg) *server {
	store, err := store.New(cfg.DBPath)
	if err != nil {
		log.Panicln("something went wrong creating the store:", err)
	}
	sessionManager := session.NewManager(store, 30, 15)
	ws := ws.New(store)
	googleCfg := auth.GoogleCgf{
		ClientID:     cfg.ClientId,
		ClientSecret: cfg.ClientSecret,
		Host:         cfg.Addr,
		CallbackURL:  cfg.Addr + "/api/login/google/callback",
		SessionMgr:   sessionManager,
		Store:        store,
	}
	google := auth.NewGoogle(googleCfg)
	CORSAllowed := map[string]bool{
		cfg.Addr + ":3001": true,
		cfg.Addr:           true,
	}
	protectedRoutes := map[string]bool{
		"/api/login/session": true,
		"/api/logout":        true,
	}
	return &server{
		addr:            cfg.Addr,
		clientId:        cfg.ClientId,
		clientSecret:    cfg.ClientSecret,
		env:             cfg.Env,
		store:           store,
		sessionManager:  sessionManager,
		ws:              ws,
		google:          google,
		CORSAllowed:     CORSAllowed,
		protectedRoutes: protectedRoutes,
	}
}

func (s *server) Start(ctx context.Context) {
	mux := http.NewServeMux()
	mux.Handle("/api/ws", herr.W(s.ws.Handle))
	mux.Handle("GET /api/login/google", herr.W(s.google.HandleLogin))
	mux.Handle("GET /api/login/google/callback", herr.W(s.google.HandleCallBack))
	mux.Handle("GET /api/login/session", herr.W(s.sessionManager.HandleCurrentSession))
	mux.Handle("POST /api/logout", herr.W(s.sessionManager.HandleLogout))
	mux.Handle("GET /metrics", promhttp.Handler())
	server := mw.Chain(
		mux,
		mw.RateLimit(15, 50), // add 15 requests per second to bucket, 50 in burst for chunk request
		mw.Logger(),
		mw.CORS(s.CORSAllowed),
		mw.Protect(s.protectedRoutes, s.sessionManager),
		mw.Metrics(),
	)

	httpServer := &http.Server{
		Addr:    s.addr,
		Handler: server,
	}

	go func() {
		slog.Info("Server is listening", "addr", s.addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "error listening and serving: %s\n", err)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		shutdownCtx := context.Background()
		shutdownCtx, cancel := context.WithTimeout(shutdownCtx, 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			fmt.Fprintf(os.Stderr, "error shutting down http server: %s\n", err)
		}
	}()
	wg.Wait()
}
