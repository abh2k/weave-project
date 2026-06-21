package main

import (
	"log"
	"net"
	"os"

	apiv1 "weave-project/gen/proto/weave/api/v1"
	"weave-project/internal/grpc/githubsearch"
	"weave-project/internal/middleware"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("failed to listen on :8080: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.LoggingInterceptor),
	)

	gitToken := os.Getenv("GITHUB_TOKEN")
	if gitToken == "" {
		log.Fatalf("GITHUB_TOKEN env not set")
	}
	githubSearchServer := githubsearch.NewServer(gitToken)

	apiv1.RegisterGithubSearchServiceServer(grpcServer, githubSearchServer)

	reflection.Register(grpcServer)
	log.Println("gRPC server listening on :8080")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("failed to serve gRPC server: %v", err)
	}
}
