package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"ITUserver/node"
)

func main() {
	
	name := os.Args[1]
	port := os.Args[2]
	peer1Name := os.Args[3]
	peer1Port := os.Args[4]
	peer2Name := os.Args[5]
	peer2Port := os.Args[6]

	n, err := node.CreateNode(name, port)
	if err != nil {
		log.Fatalf("Failed to create node: %v", err)
	}

	n.AddPeer(peer1Name, peer1Port)

	n.AddPeer(peer2Name, peer2Port)

	if err := n.Start(); err != nil {
		log.Fatalf("Failed to start node: %v", err)
	}

	fmt.Printf("Node %s running on port %s (Lamport T=%d)", n.Name, n.Port, n.LamportClock)

	go func() {
		time.Sleep(2 * time.Second)
		n.RequestAccess()

		for !n.InCriticalSec {
			time.Sleep(500 * time.Millisecond)
		}

		log.Printf("[%s] Doing work in cs...", n.Name)
		time.Sleep(5 * time.Second)

		n.ReleaseAccess()
	}()

	select {}
}
