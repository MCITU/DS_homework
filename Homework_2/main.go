package main

import (
	"fmt"
	"time"
)

// TCP packet struct
type Packet struct {
	seq     int
	ack     int
	is_syn  bool
	is_ack  bool
	message string
}

// function to print packets
func (p Packet) print() {
	flags := ""
	if p.is_syn {
		flags += "SYN "
	}
	if p.is_ack {
		flags += "ACK"
	}
	fmt.Printf("seq=%d ack=%d [%s] '%s'\n", p.seq, p.ack, flags, p.message)
}

// Client starts the handshake
func client(client_to_server chan Packet, server_to_client chan Packet) {
	fmt.Println("\nCLIENT STARTING")

	// sequence number
	my_seq := 1

	fmt.Printf("Client starting with seq=%d\n", my_seq)

	// Step 1: Send SYN packet
	fmt.Println("\nStep 1: Sending SYN...")
	syn_packet := Packet{
		seq:     my_seq,
		ack:     0, // no ack yet
		is_syn:  true,
		is_ack:  false,
		message: "Hello server!",
	}
	syn_packet.print()
	client_to_server <- syn_packet

	// Step 3: Wait for SYN-ACK and send ACK back
	fmt.Println("\nWaiting for server response...")
	response := <-server_to_client
	fmt.Println("Got response:")
	response.print()

	// Check if it's the right kind of packet
	if response.is_syn && response.is_ack {
		fmt.Println("\nStep 3: Server sent SYN-ACK! Sending final ACK...")

		// Final ACK packet
		ack_packet := Packet{
			seq:     my_seq + 1,       // my seq + 1
			ack:     response.seq + 1, // server's seq + 1
			is_syn:  false,
			is_ack:  true,
			message: "Connection OK!",
		}
		ack_packet.print()
		client_to_server <- ack_packet

		fmt.Println("\nHandshake complete!")
	}
}

// Server - responds to handshake
func server(client_to_server chan Packet, server_to_client chan Packet) {
	fmt.Println("\nSERVER STARTING")

	//Sequence number
	my_seq := 100

	fmt.Printf("Server listening: seq=%d\n", my_seq)
	fmt.Println("Waiting for client...")

	// Step 1: Wait for SYN
	request := <-client_to_server
	fmt.Println("\nStep 1: Got packet from client:")
	request.print()

	// Check if it's a SYN packet
	if request.is_syn && !request.is_ack {
		fmt.Println("\nStep 2: Client sent SYN! Sending SYN-ACK back...")

		// Send SYN-ACK response
		synack_packet := Packet{
			seq:     my_seq,          // my sequence number
			ack:     request.seq + 1, // client's seq + 1
			is_syn:  true,
			is_ack:  true,
			message: "Hello client!",
		}
		synack_packet.print()
		server_to_client <- synack_packet

		// Step 3: Wait for final ACK
		fmt.Println("\nWaiting for final ACK...")
		final_ack := <-client_to_server
		fmt.Println("Got final packet:")
		final_ack.print()

		// Check if it's the final ACK
		if final_ack.is_ack && !final_ack.is_syn {
			fmt.Println("\nConnection established!")
		}
	} else {
		fmt.Println("ERROR: Expected SYN packet")
	}
}

func main() {
	fmt.Println("TCP Handshake Simulation")
	fmt.Println("This shows the 3-way handshake:")
	fmt.Println("1. Client -> Server: SYN")
	fmt.Println("2. Server -> Client: SYN-ACK")
	fmt.Println("3. Client -> Server: ACK")

	// Make channels for communication
	client_to_server := make(chan Packet)
	server_to_client := make(chan Packet)

	// Start server in background (goroutine)
	go server(client_to_server, server_to_client)

	// Wait a bit so server starts first
	time.Sleep(100 * time.Millisecond)

	// Start client in background
	go client(client_to_server, server_to_client)

	// Wait for everything to finish
	time.Sleep(2 * time.Second)

	fmt.Println("\nDone! The handshake shows:")
	fmt.Println("- How TCP establishes connections")
	fmt.Println("- Sequence numbers track packets")
	fmt.Println("- Both sides confirm the connection")
	fmt.Println("- Uses goroutines for concurrent processing")
}
