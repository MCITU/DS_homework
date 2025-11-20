package main

import (
	proto "ITUserver/grpc"
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type userInfo struct {
	name   string
	conn   *grpc.ClientConn
	client proto.ITUAuctionServerClient
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex
}

func dialWithFallback() (*grpc.ClientConn, error) {
	addrs := []string{"localhost:5000", "localhost:5001"}

	for _, addr := range addrs {
		conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())

		if err == nil {
			log.Printf("Connected to %s", addr)
			return conn, nil
		}
		log.Printf("Failed to connect to %s: %s", addr, err)
	}
	return nil, fmt.Errorf("could not connect")
}

func createUser(name string) (*userInfo, error) {
	logFile, err := os.OpenFile(fmt.Sprintf("client_%s.log", name), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return nil, err
	}
	log.SetOutput(logFile)

	conn, err := dialWithFallback()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &userInfo{
		name:   name,
		conn:   conn,
		client: proto.NewITUAuctionServerClient(conn),
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

func (c *userInfo) join() error {
	log.Printf("[Client %s] Connecting to server %s", c.name, c.conn.Target())

	stream, err := c.client.JoinBidding(c.ctx, &proto.JoinRequest{
		ParticipantName: c.name,
	})
	if err != nil {
		return err
	}

	go c.receiveBid(stream)
	return nil
}

func (c *userInfo) reconnect() error {
	if c.conn != nil {
		_ = c.conn.Close()
	}

	conn, err := dialWithFallback()
	if err != nil {
		return err
	}

	c.conn = conn
	c.client = proto.NewITUAuctionServerClient(conn)
	log.Printf("[Client %s] Connecting to server %s", c.name, c.conn.Target())
	return nil
}

func (c *userInfo) receiveBid(stream proto.ITUAuctionServer_JoinBiddingClient) {
	for {
		msg, err := stream.Recv()
		if err != nil {
			if c.ctx.Err() != nil {
				log.Printf("[Client %s] context canceled", c.name)
				return
			}

			log.Printf("[Client %s] Error receiving bid: %v. Trying to reconnect...", c.name, err)

			//Trying to reconnect
			if recErr := c.reconnect(); recErr != nil {
				log.Printf("[Client %s] Error reconnecting: %v", c.name, recErr)
				return
			}

			//Create a new stream by calling JoinBidding again
			newStream, err := c.client.JoinBidding(c.ctx, &proto.JoinRequest{
				ParticipantName: c.name,
			})
			if err != nil {
				log.Printf("[Client %s] Error reconnecting: %v", c.name, err)
				return
			}

			log.Printf("[Client %s] Rejoined bidding after reconnect", c.name)
			stream = newStream
		}

		fmt.Printf("New bid: %d\n", msg.Amount)
		log.Printf("[Client: %s] Recieved: %d", c.name, msg.Amount)
	}
}

func (c *userInfo) sendBid(amount int64) error {
	bid := &proto.BidMessage{
		ParticipantName: c.name,
		Amount:          amount,
	}

	_, err := c.client.PublishBid(c.ctx, bid)
	if err != nil {
		return err
	}

	log.Printf("[Client %s] Published bid: %d", c.name, amount)

	//Trying to reconnect
	if recErr := c.reconnect(); recErr != nil {
		return fmt.Errorf("error reconnecting: %v", recErr)
	}

	//Try to publish bid after reconnect
	_, err = c.client.PublishBid(c.ctx, bid)
	if err != nil {
		return fmt.Errorf("error publishing bid after reconnect: %v", err)
	}

	log.Printf("[Client %s] Send a bid: %d", c.name, amount)
	return nil
}

func (c *userInfo) getHighestBid() error {
	req := &proto.GetHighestBidRequest{}

	resp, err := c.client.GetHighestBid(c.ctx, req)
	if err != nil {
		log.Printf("[Client %s] Get Highest Bid failed: %v, trying to reconnect", c.name, err)
		return nil
	}

	//Trying to reconnect
	if recErr := c.reconnect(); recErr != nil {
		log.Printf("[Client %s] Error reconnecting: %v", c.name, recErr)
	}

	//ry to get the highest bid after reconnect
	resp, err = c.client.GetHighestBid(c.ctx, req)
	if err != nil {
		return fmt.Errorf("GetHighestBid failed after reconnect: %v", err)
	}

	fmt.Printf("Current highest bid: %d\n", resp.Amount)
	return nil
}

func (c *userInfo) leave() {
	log.Printf("[Client %s] Leaving", c.name)
	_, err := c.client.LeaveBidding(c.ctx, &proto.LeaveRequest{
		ParticipantName: c.name,
	})
	if err != nil {
		log.Printf("[Client %s] Leave failed: %v", c.name, err)
	}

	c.cancel()
	c.conn.Close()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Please enter your client name")
		os.Exit(1)
	}
	c, err := createUser(os.Args[1])
	if err != nil {
		log.Fatalf("Client not created: %e", err)
	}
	if err := c.join(); err != nil {
		log.Fatalf("Failed to join: %e", err)
	}

	fmt.Printf("Client %s joined succesfully (type /leave to leave)\n", c.name)
	fmt.Println()
	fmt.Println("Type an amount to bid")
	fmt.Println("Type 'bid' to get the current highest bid")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input == "/leave" {
			c.leave()
		}
		if input == "bid" {
			c.getHighestBid()
		}

		amount, err := strconv.ParseInt(input, 10, 64)
		if err != nil {
			fmt.Println("Please enter a whole number for your bid, or type /leave to exit.")
			continue
		}

		if err := c.sendBid(amount); err != nil {
			fmt.Printf("Error seding bid: %v\n", err)
		}
	}

	c.leave()
	fmt.Println("Bye!")
}
