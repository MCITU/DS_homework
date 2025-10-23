package main

import (
	proto "ITUserver/grpc"
	"context"
	"log"
	"net"

	"google.golang.org/grpc"
)

type ITU_databaseServer struct {
	proto.UnimplementedITUDatabaseServer
	messages []string
}

func (s *ITU_databaseServer) GetMessages(ctx context.Context, in *proto.Empty) (*proto.Message, error) {
	return &proto.Message{Message: s.messages}, nil
}

func main() {
	server := &ITU_databaseServer{messages: []string{}}
	server.messages = append(server.messages, "Message 1")
	server.messages = append(server.messages, "Message 2")
	server.messages = append(server.messages, "Message 3")
	server.messages = append(server.messages, "Message 4")

	server.start_server()
}

func (s *ITU_databaseServer) start_server() {
	grpcserver := grpc.NewServer()
	listener, err := net.Listen("tcp", ":8000")

	if err != nil {
		log.Fatalf("SERVER WONT WORK")
	}

	log.Println("Server Started")

	proto.RegisterITUDatabaseServer(grpcserver, s)

	err = grpcserver.Serve(listener)
}
