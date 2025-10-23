package server

import (
	proto "ITUserver/grpc"
	"context"
	"log"
	"net"
	"os"
	"sync"

	"google.golang.org/grpc"
)

type ITU_databaseServer struct {
	proto.UnimplementedITUDatabaseServer
	mu           sync.Mutex
	lamportClock int
	clients      map[string]chan proto.BroadcastMessage
	logger       *log.Logger
}

func newServer() *ITU_databaseServer {
	//Log file creation
	logFile, err := os.OpenFile("server.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	return &ITU_databaseServer{
		lamportClock: 0,
		clients:      make(map[string]chan proto.BroadcastMessage),
		logger:       log.New(logFile, "", log.LstdFlags),
	}
}

// increment Lamport clock and return new value
func (s *ITU_databaseServer) incrementClock() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lamportClock++
	s.logger.Printf("(SERVER) Lamport clock incremeneted to %d", s.lamportClock)
	return s.lamportClock
}

func (s *ITU_databaseServer) updateClock(recievedTime int) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if recievedTime > s.lamportClock {
		s.lamportClock = recievedTime
	}
	s.lamportClock++
	s.logger.Printf("(SERVER) Lamport clock updated to %d", s.lamportClock)
	return s.lamportClock
}

func (s *ITU_databaseServer) broadcast(message proto.BroadcastMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Printf("(SERVER) Broadcasting message: %s (Lamport: %d, Type: %v)\n", message.Content, message.LamportTimestamp, message.Type)

	for clientName, ch := range s.clients {
		select {
		case ch <- message:
			s.logger.Printf("(SERVER) Message delivered to client: %s\n", clientName)
		default:
			s.logger.Printf("(SERVER) Message failed to deliver to client: %s\n", clientName)
		}
	}
}
