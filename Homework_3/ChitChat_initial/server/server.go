package main

import (
	pb "ITUserver/grpc"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"google.golang.org/grpc"
)

type server struct {
	pb.ITUDatabaseServer
	mu           sync.Mutex
	lamportClock int64
	clients      map[string]chan *pb.BroadcastMessage
}

func newServer() *server {
	return &server{
		clients: make(map[string]chan *pb.BroadcastMessage),
	}
}

func (s *server) incrementClock() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lamportClock++
	return s.lamportClock
}

func (s *server) broadcast(msg *pb.BroadcastMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("[Server] Broadcasting: %s (Lamport: %d)", msg.Content, msg.LamportTimestamp)

	for _, ch := range s.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (s *server) JoinChat(req *pb.JoinRequest, stream pb.ITUDatabase_JoinChatServer) error {
	clientName := req.ParticipantName
	log.Printf("[Server] Client %s joined", clientName)

	msgChan := make(chan *pb.BroadcastMessage, 100)

	s.mu.Lock()
	s.clients[clientName] = msgChan
	s.mu.Unlock()

	lamportTime := s.incrementClock()
	joinMsg := &pb.BroadcastMessage{
		Content:          fmt.Sprintf("Participant %s joined Chit Chat at Lamport time %d", clientName, lamportTime),
		LamportTimestamp: lamportTime,
		Type:             pb.MessageType_JOIN,
	}
	s.broadcast(joinMsg)

	for {
		select {
		case msg, ok := <-msgChan:
			if !ok {
				return nil
			}
			if err := stream.Send(msg); err != nil {
				s.removeClient(clientName)
				return err
			}
		case <-stream.Context().Done():
			log.Printf("[Server] Client %s disconnected", clientName)
			s.removeClient(clientName)
			return nil
		}
	}
}

func (s *server) PublishMessage(ctx context.Context, msg *pb.ChatMessage) (*pb.PublishResponse, error) {
	if len(msg.Content) > 128 {
		return &pb.PublishResponse{Success: false}, fmt.Errorf("message exceeds 128 characters")
	}

	lamportTime := s.incrementClock()
	log.Printf("[Server] Message from %s (Lamport: %d)", msg.ParticipantName, lamportTime)

	broadcastMsg := &pb.BroadcastMessage{
		Content:          fmt.Sprintf("%s: %s", msg.ParticipantName, msg.Content),
		LamportTimestamp: lamportTime,
		Type:             pb.MessageType_CHAT,
	}
	s.broadcast(broadcastMsg)

	return &pb.PublishResponse{
		Success:          true,
		LamportTimestamp: lamportTime,
	}, nil
}

func (s *server) LeaveChat(ctx context.Context, req *pb.LeaveRequest) (*pb.LeaveResponse, error) {
	clientName := req.ParticipantName
	lamportTime := s.incrementClock()

	log.Printf("[Server] Client %s left (Lamport: %d)", clientName, lamportTime)

	leaveMsg := &pb.BroadcastMessage{
		Content:          fmt.Sprintf("Participant %s left Chit Chat at Lamport time %d", clientName, lamportTime),
		LamportTimestamp: lamportTime,
		Type:             pb.MessageType_LEAVE,
	}
	s.broadcast(leaveMsg)
	s.removeClient(clientName)

	return &pb.LeaveResponse{Success: true}, nil
}

func (s *server) removeClient(clientName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ch, exists := s.clients[clientName]; exists {
		close(ch)
		delete(s.clients, clientName)
	}
}

func main() {
	logFile, err := os.OpenFile("server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log: %v", err)
	}
	log.SetOutput(logFile)

	srv := newServer()
	log.Println("[Server] Starting up")

	grpcServer := grpc.NewServer()
	pb.RegisterITUDatabaseServer(grpcServer, srv)

	lis, err := net.Listen("tcp", ":5000")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Println("[Server] Listening on port 5000")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("[Server] Shutting down")
		grpcServer.GracefulStop()
	}()

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
