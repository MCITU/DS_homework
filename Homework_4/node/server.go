package node

import (
	"context"
	"fmt"
	"log"
	"net"

	pb "ITUserver/grpc"
	"google.golang.org/grpc"
)

type ituServer struct {
	pb.UnimplementedITUDatabaseServer
	n *node
}

func newServer(port string, n *node) (*grpc.Server, error) {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, err
	}

	grpcServer := grpc.NewServer()
	serverImpl := &ituServer{n: n}

	pb.RegisterITUDatabaseServer(grpcServer, serverImpl)

	go func() {
		fmt.Printf("Node %s listening on port %s\n", n.Name, port)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()
	return grpcServer, nil
}

func (s *ituServer) SendRequest(ctx context.Context, r *pb.AccessRequest) (*pb.Confirm, error) {
	s.n.Events <- event{kind: "request", data: r}
	return &pb.Confirm{}, nil
}

func (s *ituServer) SendReply(ctx context.Context, r *pb.AccessReply) (*pb.Confirm, error) {
	s.n.Events <- event{kind: "reply", data: r}
	return &pb.Confirm{}, nil
}

func (s *ituServer) SendLeave(ctx context.Context, r *pb.LeaveNotice) (*pb.Confirm, error) {
	s.n.Events <- event{kind: "leave", data: r}
	return &pb.Confirm{}, nil
}
