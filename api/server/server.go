package server

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"screw/auth"
	"screw/hr"
	mw "screw/middleware"
	"screw/session"
	"screw/store"
	"screw/ws"

	_ "github.com/joho/godotenv/autoload"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type server struct {
	host            string
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
	Host         string
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
		Host:         cfg.Host,
		CallbackURL:  cfg.Host + "/api/login/google/callback",
		SessionMgr:   sessionManager,
		Store:        store,
	}
	google := auth.NewGoogle(googleCfg)
	CORSAllowed := map[string]bool{
		cfg.Host + ":3001": true,
		cfg.Host:           true,
	}
	protectedRoutes := map[string]bool{
		"/api/login/session": true,
		"/api/logout":        true,
	}
	return &server{
		host:            cfg.Host,
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

const (
	port = 3000
)

var portStr = fmt.Sprintf(":%d", port)

func (s *server) Start() error {
	mux := http.NewServeMux()
	mux.Handle("/api/ws", hr.W(s.ws.Handle))
	mux.Handle("GET /api/login/google", hr.W(s.google.HandleLogin))
	mux.Handle("GET /api/login/google/callback", hr.W(s.google.HandleCallBack))
	mux.Handle("GET /api/login/session", hr.W(s.sessionManager.HandleCurrentSession))
	mux.Handle("POST /api/logout", hr.W(s.sessionManager.HandleLogout))
	mux.Handle("GET /metrics", promhttp.Handler())
	server := mw.Chain(
		mux,
		mw.RateLimit(15, 50), // add 15 requests per second to bucket, 50 in burst for chunk request
		mw.Logger(),
		mw.CORS(s.CORSAllowed),
		mw.Protect(s.protectedRoutes, s.sessionManager),
		mw.Metrics(),
	)

	slog.Info("Server is listening", "port", port)
	return http.ListenAndServe(portStr, server)
}
