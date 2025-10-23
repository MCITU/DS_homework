package main

import (
	proto "ITUserver/grpc"
	"bufio"
	"context"
	"fmt"
	"log"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "grpc"
)

var (
	userIDNR int
)

func main() {
	conn, err := grpc.NewClient("localhost:8000", grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Fatalf("Count not connect")
	}

	client := proto.NewITUDatabaseClient(conn)

	if err != nil {
		log.Fatalf("WE DID NOT RECIEVE OR FAILED TO SEND")
	}

	messages, err := client.GetMessages(context.Background(), &proto.Empty{})

	for _, message := range messages.Message {
		log.Println(" - " + message)
	}
}

type userInfo struct {
	name   string
	Stream pb.ChatServce_ChatStreamClient
}

func createUser() {
	fmt.Print("enter username: ")
	userInfo

}

func sendMessage() {

	// retrv custom msg append userInfo append lamport time
	// then send to server for storing
}

func scanner() {
	var input []string
	var s string
	fmt.Fscan(bufio.NewReader(os.Stdin), &s)
	input = make([]string, len(s))
	for i, r := range s {
		input[i] = string(r)
	}
}
