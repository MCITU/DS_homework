# TCP Handshake Simulation

This small project shows a TCP's 3-ways handshake in Go by using goroutines

- (SYN -> SYN-ACK -> ACK)  
## This is how to run the code
```bash
go run main.go
```
## a) What are packages in your implementation? What data structure do you use to transmit data and meta-data?
Packages used: Standard library imports fmt (printing) and time (simple timing).

Data structure: A simplified TCP packet as a Go struct:
```go
type Packet struct {
seq     int
ack     int
is_syn  bool
is_ack  bool
message string
}
```
Packets are transmitted over Go Channels: ```client_to_server chan Packet``` and ```server_to_client chan Packet```

## b) Does your implementation use threads or processes? Why is it not realistic to use threads?
- **Concurrency model:** It uses goroutines (threads) and channels.

- **Why this isn’t realistic:** Real TCP runs across a distributed, unreliable network where packets can be delayed, reordered, or dropped, and endpoints do not share memory. Threads in one process don’t produce those network properties

## c) In case the network changes the order in which messages are delivered, how would you handle message re-ordering?
Use sequence numbers.
- I already use sequence and acknowledgement numbers in my ```Packet``` struct (```seq```, ```ack``` fields) and i already print them.

However i do not unlike real TCP have a ```nextSeq``` check, buffering and reordering of out of order messages.

If i needed to handle re-ordering i could do so with a buffer ```map[int]Packet```
And whenever a ```seq == nextSeq``` arrives, i could deliver them in the correct order.
## d) In case messages can be delayed or lost, how does your implementation handle message loss?
My current code assumes everything is perfect and there is not delayed or lost messages.
However if i were to implement such "features" and need to handle loss, i could do so with timeouts and retransmissions.
If no correct ACK/response arrives before timeout, i could retransmit the last package.
## e) Why is the 3-way handshake important?
- It synchronizes sequence numbers in both directions
- It confirms that both endpoints are live before data flows
- It prevents old and, in case of delayed connections, duplicated connections from being mistaken for new ones (Sequence numbers)
- It creates ESTABLISHED context needed.
 

