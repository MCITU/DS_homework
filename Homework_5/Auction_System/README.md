# Distributed Auction System - Server

A replicated auction server system that tolerates one node failure.

## Overview
- Auction duration: 100 seconds
- Automatic restart after auction ends
- Supports replication between two nodes
- Commands: BID, RESULT, REPLICATE

## Prerequisites
- Go 1.21+
- Protocol Buffers compiler (optional, for regenerating code)
- gRPC Go plugins (optional, for regenerating code)

## Starting the Servers

**1. Start the server:**
- Usage: go run main.go <port> [peer_address]
- Example : 
```bash
  go run main.go 5000 localhost:5001
```
- This creates a server on port 5000 with the [peer_address] marked as localhost:5001.

**2. Start the backup-server:**
- Usage: go run main.go <port> [peer_address]
- Example :
```bash
  go run main.go 5001 localhost:5000
```
- This creates a server on port 5001 with the [peer_address] marked as localhost:5000.

## Server Operations
### BID COMMAND
- Format: BID <[bidder]> <[amount]>
- Places a bid for the specific bidder 
- Returns: success, fail, or exception

### RESULT COMMAND
- Format: RESULT
- If ongoing: shows current leader
- If ended shows winner:

### REPLICATE COMMAND
- Format: REPLICATE <[bidder]> <[amount]>
- Internal command for node replication
- Automatically called when primary server recieves a bid

### RESPONSES
- Success: Bid Accepted - Bid placed successfully
- Fail: Bid must be higher.... - Bid rejected (it was too low)
- Exception: Auction has ended - Auction closed.

## Logs
- Each server creates its own log file: server_[port].log

## Starting a Client
**2. Start clients (in separate terminals):**
```bash
    
```


## Example

**log:**
```
    
```


