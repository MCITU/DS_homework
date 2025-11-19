package main

import (
	proto "ITUserver/grpc"
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type userInfo struct {
	name         string
	conn         *grpc.ClientConn
	client       proto.ITUDatabaseClient
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.Mutex
	lamportClock int64
}

func createUser(name string) (*userInfo, error) {
	logFile, err := os.OpenFile(fmt.Sprintf("client_%s.log", name), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return nil, err
	}
	log.SetOutput(logFile)

	conn, err := grpc.Dial("localhost:5000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &userInfo{
		name:   name,
		conn:   conn,
		client: proto.NewITUDatabaseClient(conn),
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

func (c *userInfo) join() error {
	log.Printf("[Client %s] Connecting to server %s", c.name, c.conn.Target())

	stream, err := c.client.JoinChat(c.ctx, &proto.JoinRequest{
		ParticipantName: c.name,
	})
	if err != nil {
		return err
	}

	go c.receiveMessages(stream)
	return nil
}

func (c *userInfo) receiveMessages(stream proto.ITUDatabase_JoinChatClient) {
	for {
		msg, err := stream.Recv()
		if err != nil {
			log.Printf("[Client %s] Disconnected", c.name)
			return
		}

		newTime := c.updateClock(msg.LamportTimestamp)

		fmt.Printf("[Lamport: %d] %s \n", newTime, msg.Content)
		log.Printf("[Client: %s] Recieved: %s (Lamport: %d)", c.name, msg.Content, newTime)
	}
}

func (c *userInfo) sendMessage(msg string) error {
	if len(msg) > 128 {
		fmt.Errorf("Message too long - Max is 128 characters")
	}

	if len(msg) == 0 {
		fmt.Errorf("Message is empty")
	}

	lamportTime := c.incrementClock()

	_, err := c.client.PublishMessage(c.ctx, &proto.ChatMessage{
		ParticipantName: c.name,
		Content:         msg,
		Lamport:         lamportTime,
	})
	if err != nil {
		return err
	}

	log.Printf("[Client %s] Published message: %s", c.name, lamportTime)
	return nil
}

func (c *userInfo) leave() {
	log.Printf("[Client %s] Leaving", c.name)
	c.client.LeaveChat(c.ctx, &proto.LeaveRequest{
		ParticipantName: c.name,
	})

	c.cancel()
	c.conn.Close()
}

func (c *userInfo) incrementClock() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lamportClock++
	return c.lamportClock
}

func (c *userInfo) updateClock(received int64) int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	if received > c.lamportClock {
		c.lamportClock = received
	}
	c.lamportClock++
	return c.lamportClock
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Please enter your clientname")
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

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input == "/leave" {
			break
		}
		if err := c.sendMessage(input); err != nil {
			fmt.Printf("Error: %e\n", err)
		}
	}

	c.leave()
	fmt.Println("Bye!")
}
