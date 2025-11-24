# Distributed Auction System

A fault-tolerant distributed auction system using gRPC with state replication that tolerates one node failure.

## Overview

- **Auction Duration:** 100 seconds (automatically resets after completion)
- **Fault Tolerance:** Resilient to 1 server crash (requires 2 servers for full replication)
- **Communication:** gRPC with server-side streaming for real-time bid updates
- **Replication:** Automatic state synchronization between server nodes
- **Client Features:** Automatic failover between available servers

## Prerequisites

- **Go 1.21+** installed

## Project Structure

```
Auction_System/
server/
   - server.go          # Main server implementation
Client/
   - client.go          # Client implementation
grpc/
   - proto.proto        # Protocol buffer definition
   - proto.pb.go        # Generated protobuf code
   - proto_grpc.pb.go   # Generated gRPC code
- go.mod                 # Go module definition
- README.md              # This file
```

---

## Quick Start

### Option 1: Single Server (No Fault Tolerance)

**Terminal 1 - Start Server:**
```powershell
cd server
go run server.go 5000
```

**Terminal 2 - Start Client:**
```powershell
cd Client
go run client.go Alice
```

### Option 2: Two Replicated Servers (Fault Tolerant)

**Terminal 1 - Start Server 1:**
```powershell
cd server
go run server.go 5000 localhost:5001
```

**Terminal 2 - Start Server 2 (Backup):**
```powershell
cd server
go run server.go 5001 localhost:5000
```

**Terminal 3 - Start Client:**
```powershell
cd Client
go run client.go Alice
```

**Terminal 4 - Start Another Client:**
```powershell
cd Client
go run client.go Bob
```

---

## Server Usage

### Command Syntax
```bash
go run server.go <port> [peer_address]
```

### Parameters
- **`<port>`** (required): Port number for the server to listen on
- **`[peer_address]`** (optional): Address of the peer server for replication (format: `host:port`)

### Examples

**Standalone server:**
```bash
go run server.go 5000
```

**Server with replication peer:**
```bash
go run server.go 5000 localhost:5001
```

### Server Logs
- Each server creates a log file: `server_<port>.log`
- Logs include: client connections, bids received, replication events, auction resets

---

## Client Usage

### Command Syntax
```bash
go run client.go <client_name>
```

### Parameters
- **`<client_name>`** (required): Unique identifier for the client (e.g., Alice, Bob, Charlie)

### Client Commands

Once connected, the client accepts the following inputs:

| Command | Description | Example |
|---------|-------------|----------|
| `<number>` | Place a bid with the specified amount | `100` |
| `bid` | Query the current highest bid | `bid` |
| `/leave` | Exit the auction and disconnect | `/leave` |

### Client Behavior

1. **Connection:** Client automatically tries to connect to servers on ports 5000 and 5001
2. **Failover:** If one server is down, client connects to the available server
3. **Real-time Updates:** Client receives broadcast messages when other clients place bids
4. **Reconnection:** Client attempts to reconnect if connection is lost

### Client Logs
- Each client creates a log file: `client_<name>.log`

---

## Auction Rules

### Bidding Rules
1. **First Bid:** Automatically registers the bidder
2. **Subsequent Bids:** Must be higher than the bidder's previous bid
3. **Winning Bid:** Must be higher than the current highest bid from any bidder
4. **After Auction Ends:** Bids are rejected with an exception

### Bid Responses

- **Success:** `"Success: Bid Accepted"` - Bid was valid and accepted
- **Fail:** `"Fail: Bid must be higher than..."` - Bid was too low
- **Exception:** `"Exception: Auction has ended"` - Auction is closed

### Auction Lifecycle

1. Auction starts automatically when server launches
2. Runs for 100 seconds
3. After 100 seconds, auction ends and winner is determined
4. After 5 more seconds, auction automatically resets and starts again

---

## Complete Example Session

### Setup (3 Terminals)

**Terminal 1:**
```powershell
cd server
go run server.go 5000 localhost:5001
# Output: [node1] Auction server started on port 5000
#         [node1] Auction will end in 1m40s
```

**Terminal 2:**
```powershell
cd server
go run server.go 5001 localhost:5000
# Output: [node2] Auction server started on port 5001
#         [node2] Connected to peer at localhost:5000
```

**Terminal 3:**
```powershell
cd Client
go run client.go Alice
# Output: Connected to localhost:5000
#         Client Alice joined successfully (type /leave to leave)
#         
#         Type an amount to bid
#         Type 'bid' to get the current highest bid
```

### Testing Bids

**Alice's Terminal:**
```
> bid
Current highest bid: 0

> 100
# (Bid accepted)

> bid  
Current highest bid: 100

> 50
# (Silently rejected - too low)

> 200
# (Bid accepted)
```

**Bob's Terminal (in another window):**
```powershell
cd Client
go run client.go Bob
```
```
> bid
Current highest bid: 200

> 250
# (Bid accepted - now Bob is winning)

New bid: 250
# (Bob sees his own bid broadcasted)
```

**Alice sees Bob's bid:**
```
New bid: 250
# (Alice is notified in real-time)

> bid
Current highest bid: 250
```

### Testing Fault Tolerance

1. **Kill Server 1** (Ctrl+C in Terminal 1)
2. **Client continues working** with Server 2
3. **Place a bid** - still works!
4. **Restart Server 1** - it syncs state from Server 2

---

## Troubleshooting

### Port Already in Use
**Error:** `listen tcp :5000: bind: Only one usage of each socket address...`

**Solution:**
```powershell
# Find and kill process using the port
Get-Process | Where-Object {$_.ProcessName -eq "server"} | Stop-Process -Force
```

### Client Can't Connect
**Error:** `could not connect`

**Solutions:**
1. Verify server is running: Check if you see "gRPC server listening on port..."
2. Check firewall settings
3. Verify port numbers match

### Bids Not Updating
**Issue:** Client doesn't see other clients' bids

**Solutions:**
1. Verify multiple clients are connected to the same server
2. Check server logs for broadcast messages
3. Restart clients

---

## Technical Details

### Replication Strategy
- **Peer-to-peer replication:** Each server maintains a connection to its peer
- **State synchronization:** When a bid is accepted, it's replicated to the peer server
- **Consistency:** Both servers maintain identical auction state
- **Fault tolerance:** If one server fails, the other continues operating

### Communication Protocol
- **Client ↔ Server:** gRPC with Protocol Buffers
- **Server ↔ Server:** TCP sockets for state replication
- **Streaming:** Server-side streaming for real-time bid broadcasts to clients

### Key Operations

**Client Operations (gRPC):**
- `JoinBidding` - Join auction and receive bid stream
- `PublishBid` - Submit a bid
- `GetHighestBid` - Query current highest bid
- `LeaveBidding` - Leave the auction

**Server Replication (TCP):**
- `REPLICATE <bidder> <amount>` - Sync bid to peer server

---

## Regenerating Protocol Buffer Files (Optional)

Only needed if you modify `proto.proto`:

```powershell
cd grpc
protoc --go_out=. --go-grpc_out=. proto.proto
```

---

## Assignment Requirements Checklist

✅ **Distributed component** - Multiple server nodes run as separate processes  
✅ **Replication** - State replicated between nodes  
✅ **Client API** - `bid()` and `result()` operations implemented  
✅ **Fault tolerance** - System tolerates 1 node failure  
✅ **Bidding semantics** - First bid registers, subsequent must be higher  
✅ **Time-bound auction** - 100 second duration with auto-reset  
✅ **Result query** - Returns winner if ended, else highest bid  
✅ **Reliable network** - Uses TCP/gRPC with ordered message delivery  

---

## Authors

Distributed Systems - Homework 5  
ITU - 2025
