package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	pb "tagger/proto/gen"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"google.golang.org/grpc"
)

const port = 3000

var portStr = fmt.Sprintf(":%d", port)

type server struct {
	pb.UnimplementedTaggerServer
}

func (s *server) SayHello(_ context.Context, in *pb.HelloRequest) (*pb.HelloResponse, error) {
	return &pb.HelloResponse{Message: "Hello " + in.GetName()}, nil
}

func run() error {
	env := getEnv("ENV", "dev")
	if env == "prod" {
		return startCombinedServer(true)
	}
	return startCombinedServer(false)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func startCombinedServer(serveStatic bool) error {
	// Create a new gRPC server
	grpcServer := grpc.NewServer()
	pb.RegisterTaggerServer(grpcServer, &server{})

	// Create a gRPC-Web wrapper around the gRPC server
	wrappedGrpc := grpcweb.WrapServer(grpcServer,
		grpcweb.WithOriginFunc(func(origin string) bool {
			return true
		}),
		grpcweb.WithAllowedRequestHeaders([]string{"*"}),
	)

	// Create a new mux for all HTTP handling
	mux := http.NewServeMux()

	// Add static file server if needed
	if serveStatic {
		dir := "/frontend/out"
		fileServer := http.FileServer(http.Dir(dir))
		mux.Handle("/", fileServer)
		slog.Info("Registered static file server", "dir", dir)
	}

	// Create the main handler that will handle both gRPC-Web and regular HTTP
	mainHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers",
			"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-User-Agent, X-Grpc-Web")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Log incoming request
		slog.Info("Received request",
			"method", r.Method,
			"url", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)

		// Check if it's a gRPC-Web request
		if wrappedGrpc.IsGrpcWebRequest(r) {
			wrappedGrpc.ServeHTTP(w, r)
			return
		}

		// For all other requests, use the mux
		mux.ServeHTTP(w, r)
	})

	// Create TCP listener
	lis, err := net.Listen("tcp", portStr)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	// Create the HTTP server
	httpServer := &http.Server{
		Handler: mainHandler,
	}

	slog.Info("Starting combined server",
		"port", port,
		"static_files", serveStatic,
	)

	return httpServer.Serve(lis)
}

func main() {
	if err := run(); err != nil {
		slog.Error("Application failed", "err", err)
		os.Exit(1)
	}
}
