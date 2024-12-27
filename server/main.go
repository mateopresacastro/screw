package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"

	pb "tagger/proto/gen"

	"google.golang.org/grpc"
)

const port = 3000

var portStr = fmt.Sprintf(":%d", port)

func main() {
	if err := run(); err != nil {
		slog.Error("Application failed", "err", err)
		os.Exit(1)
	}
}

type server struct {
	pb.UnimplementedTaggerServer
}

func (s *server) SayHello(_ context.Context, in *pb.HelloRequest) (*pb.HelloResponse, error) {
	return &pb.HelloResponse{Message: "Hello " + in.GetName()}, nil
}

func run() error {
	env := getEnv("ENV", "dev")
	if env == "prod" {
		go startFileServer()
	}
	return startGrpcServer()
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func startFileServer() error {
	dir := "/client/out"
	http.Handle("GET /", http.FileServer(http.Dir(dir)))
	slog.Info("Starting file server", "env", "prod", "port", port)
	return http.ListenAndServe(portStr, nil)
}

func startGrpcServer() error {
	lis, err := net.Listen("tcp", portStr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterTaggerServer(s, &server{})
	slog.Info("TCP server listening", "port", port)
	return s.Serve(lis)
}
