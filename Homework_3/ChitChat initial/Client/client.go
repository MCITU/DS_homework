package main

import (
	proto "ITUserver/grpc"
	"context"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
