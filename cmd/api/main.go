package main

import (
	"log"
	"net"
	"os"

	apiv1 "weave-project/gen/proto/weave/api/v1"
	"weave-project/internal/grpc/githubsearch"
	"weave-project/internal/middleware"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("failed to listen on :8080: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.GRPCRequestLogger()),
	)
	healthServer := health.NewServer()

	git_token := os.Getenv("GITHUB_TOKEN")
	if git_token == "" {
		log.Fatalf("GITHUB_TOKEN env not set")
	}
	githubSearchServer := githubsearch.NewServer(git_token)

	apiv1.RegisterGithubSearchServiceServer(grpcServer, githubSearchServer)
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	reflection.Register(grpcServer)

	// Empty service name sets overall server health status.
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	log.Println("gRPC server listening on :8080")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("failed to serve gRPC server: %v", err)
	}
}
