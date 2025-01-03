package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	pb "grpc_proxy_yt_thumbnail/grpc-proxy/proto/echo"
)

type server struct {
	pb.UnimplementedEchoServer
}

func (s *server) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloResponse, error) {
	log.Printf("Received: %s", req.Message)
	return &pb.HelloResponse{Message: fmt.Sprintf("Hello from Backend: %s", req.Message)}, nil
}

func main() {
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterEchoServer(s, &server{})

	log.Println("Backend gRPC server is running on :50051")
	if err := s.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
