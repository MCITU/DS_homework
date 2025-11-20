package main

import (
	pb "ITUserver/grpc"
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
)

const AuctionDuration = 100 * time.Second //1 min 40 sec

type server struct {
	pb.UnimplementedITUAuctionServerServer
	port          string                      //port vi lytter på
	peerAddress   string                      //addressen på den anden server (til replikering)
	bids          map[string]int64            // alle bud: Bidder -> Bids
	highestBid    int64                       //Det aller højeste
	highestBidder string                      //Hvem der vinder
	startTime     time.Time                   //Hvornår startede auktionen
	mu            sync.Mutex                  // Lock så vi kan beskytte bids
	clients       map[string]chan *pb.BroadcastMessage // Active client streams
	clientsMu     sync.Mutex                  // Lock for clients map
}

// starter serveren
func newServer(port, peerAddress string) *server {
	return &server{
		port:        port,
		peerAddress: peerAddress,
		bids:        make(map[string]int64),
		startTime:   time.Now(),
		clients:     make(map[string]chan *pb.BroadcastMessage),
	}
}

// er auctionen slut?
func (s *server) isAuctionEnded() bool {
	return time.Since(s.startTime) > AuctionDuration
}

// getstate - return current highest bid
func (s *server) getState() int64 {
	return s.highestBid
}

func (s *server) handleBid(bidder string, amount int64) string {
	s.mu.Lock()         //låser
	defer s.mu.Unlock() // Lås op - Defer, vent til return for at låse op.
	if s.isAuctionEnded() {
		return "Exception : Auction has ended, couldn't place bid"
	}

	if prev, exists := s.bids[bidder]; exists && amount <= prev {
		return "Fail: Bid must be higher than the previous"
	}
	if amount <= s.highestBid {
		return "Fail: Bid must be higher than the highest bid"
	}

	s.bids[bidder] = amount
	s.highestBid = amount
	s.highestBidder = bidder

	// Broadcast to all connected clients
	go s.broadcast(&pb.BroadcastMessage{
		Amount: amount,
		Type:   pb.MessageType_BID,
	})

	// ref til backup
	go s.replicate(bidder, amount) //Sender til backup

	return "Success: Bid Accepted"
}

func (s *server) endAuction() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.isAuctionEnded() {
		if s.highestBidder == "" {
			return "\n No bids placed" //Ingen bød Auctionen er slut.
		} else {
			fmt.Printf("\n Winner was %s, with the bid: %d", s.highestBidder, s.highestBid)
		}
	}
	if s.highestBidder == "" {
		return "Auction is ongoing: No bids placed"
	}
	return (fmt.Sprintf("Auction is ongoing: %s is leading with %d", s.highestBidder, s.highestBid))
}
func (s *server) resetAuction() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.bids = make(map[string]int64)
	s.highestBidder = ""
	s.highestBid = 0
	s.startTime = time.Now()
	log.Printf("Auction reset - New Auction starting in & ending at: %s", s.startTime.Add(AuctionDuration).Format("13:48:05"))
}

func (s *server) replicate(bidder string, amount int64) {
	if s.peerAddress == "" {
		return // der er ikke configureret en backup server/node
	}
	conn, err := net.DialTimeout("tcp", s.peerAddress, 2*time.Second)
	if err != nil {
		return // der kunne ikke connectes til backup server/node
	}
	defer conn.Close()
	fmt.Fprintf(conn, "REPLICATE %s %d\n", bidder, amount)
	return
}

func (s *server) handleReplicate(bidder string, amount int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if amount > s.highestBid {
		s.highestBid = amount
		s.bids[bidder] = amount
		s.highestBidder = bidder
	}
}

func (s *server) handleConn(conn net.Conn) {
	defer conn.Close() //luk forbindelse når færdig
	scanner := bufio.NewScanner(conn)

	if !scanner.Scan() {
		return //ingen data
	}
	parts := strings.Fields(scanner.Text()) //Splitter linjen e.g "BID <navn> <bud>" -> ["BID","<navn>", "<bud>"]
	if len(parts) == 0 {
		return
	}

	cmd := parts[0]
	var response string

	switch cmd {
	case "BID":
		if len(parts) < 2 {
			response = "Exception: BID requires bidder and a bid"
		} else {
			amount, _ := strconv.ParseInt(parts[2], 10, 64)
			response = s.handleBid(parts[1], amount)
		}
	case "RESULT":
		response = s.endAuction()

	case "REPLICATE":
		if len(parts) >= 3 {
			amount, _ := strconv.ParseInt(parts[2], 10, 64)
			s.handleReplicate(parts[1], amount)
			response = "Ok!"
		}
	}
	fmt.Fprintln(conn, response) //Send svar tilbage
}

// gRPC method implementations
func (s *server) JoinBidding(req *pb.JoinRequest, stream grpc.ServerStreamingServer[pb.BroadcastMessage]) error {
	log.Printf("Client %s joined bidding", req.ParticipantName)

	// Create channel for this client
	clientChan := make(chan *pb.BroadcastMessage, 10)
	s.clientsMu.Lock()
	s.clients[req.ParticipantName] = clientChan
	s.clientsMu.Unlock()

	// Send join broadcast
	go s.broadcast(&pb.BroadcastMessage{
		Amount: 0,
		Type:   pb.MessageType_JOIN,
	})

	// Stream messages to client
	for msg := range clientChan {
		if err := stream.Send(msg); err != nil {
			log.Printf("Error sending to client %s: %v", req.ParticipantName, err)
			break
		}
	}

	// Cleanup
	s.clientsMu.Lock()
	delete(s.clients, req.ParticipantName)
	s.clientsMu.Unlock()
	return nil
}

func (s *server) PublishBid(ctx context.Context, bid *pb.BidMessage) (*pb.PublishResponse, error) {
	result := s.handleBid(bid.ParticipantName, bid.Amount)
	success := strings.Contains(result, "Success")
	return &pb.PublishResponse{
		Success: success,
		Reason:  result,
	}, nil
}

func (s *server) LeaveBidding(ctx context.Context, req *pb.LeaveRequest) (*pb.LeaveResponse, error) {
	log.Printf("Client %s left bidding", req.ParticipantName)

	s.clientsMu.Lock()
	if ch, exists := s.clients[req.ParticipantName]; exists {
		close(ch)
		delete(s.clients, req.ParticipantName)
	}
	s.clientsMu.Unlock()

	go s.broadcast(&pb.BroadcastMessage{
		Amount: 0,
		Type:   pb.MessageType_LEAVE,
	})

	return &pb.LeaveResponse{Success: true}, nil
}

func (s *server) GetHighestBid(ctx context.Context, req *pb.GetHighestBidRequest) (*pb.GetHighestBidResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return &pb.GetHighestBidResponse{
		Amount: s.highestBid,
	}, nil
}

// Broadcast message to all connected clients
func (s *server) broadcast(msg *pb.BroadcastMessage) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()

	for name, ch := range s.clients {
		select {
		case ch <- msg:
		default:
			log.Printf("Client %s channel full, skipping message", name)
		}
	}
}

func (s *server) Start() {
	ln, err := net.Listen("tcp", ":"+s.port)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	log.Printf("Server on port %s (peer: %s)", s.port, s.peerAddress)
	log.Printf("Auction started, ending at: %s", s.startTime.Add(AuctionDuration).Format("15:04:05"))

	// Auto-restart auctionen efter slut
	go func() {
		for {
			time.Sleep(AuctionDuration + 5*time.Second)
			s.resetAuction()
		}
	}()

	// Start gRPC server
	grpcServer := grpc.NewServer()
	pb.RegisterITUAuctionServerServer(grpcServer, s)

	log.Printf("gRPC server listening on port %s", s.port)
	if err := grpcServer.Serve(ln); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <port> [peer_address]")
		fmt.Println("Example: go run main.go 5000 localhost:5001")
		os.Exit(1)
	}
	port := os.Args[1]
	peer := ""
	if len(os.Args) > 2 {
		peer = os.Args[2]
	}
	logFile, _ := os.OpenFile(fmt.Sprintf("server_%s.log", port), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if logFile != nil {
		log.SetOutput(logFile)
		defer logFile.Close()
	}
	server := newServer(port, peer) //opret server
	server.Start()
}
