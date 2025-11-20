package main

import (
	pb "ITUserver/grpc"
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const AuctionDuration = 100 * time.Second //1 min 40 sec

type server struct {
	pb.ITUAuctionServer
	port          string           //port vi lytter på
	peerAddress   string           //addressen på den anden server (til replikering)
	bids          map[string]int64 // alle bud: Bidder -> Bids
	highestBid    int64            //Det aller højeste
	highestBidder string           //Hvem der vinder
	startTime     time.Time        //Hvornår startede auktionen
	mu            sync.Mutex       // Lock så vi kan beskytte bids
}

// starter serveren
func newServer(port, peerAddress string) *server {
	return &server{
		port:        port,
		peerAddress: peerAddress,
		bids:        make(map[string]int64),
		startTime:   time.Now(),
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

func (s *server) Start() {
	ln, err := net.Listen("tcp", ":"+s.port)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	log.Printf("Server on port %s (peer: %s", s.port, s.peerAddress)
	log.Printf("Auction started, ending at: %s", s.startTime.Add(AuctionDuration).Format("13:33:05"))

	// Auto-restart auctionen efter slut
	go func() {
		for {
			time.Sleep(AuctionDuration + 5*time.Second)
			s.resetAuction()
		}
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go s.handleConn(conn)
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
