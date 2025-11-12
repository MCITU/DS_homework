# Ricart-Agrawala

using Go, gRPC and the Ricart-Agrawala algorithm

## Prerequisites

- Go 1.21+
- Protocol Buffers compiler (optional, for regenerating code)
- gRPC Go plugins (optional, for regenerating code)

## Usage

**1. setup nodes:**
```bash
go run main.go Node1 5001 Node2 localhost:5002 Node3 localhost:5003
go run main.go Node2 5002 Node1 localhost:5001 Node3 localhost:5003
go run main.go Node3 5003 Node1 localhost:5001 Node2 localhost:5002
```
this creates three nodes that will start communicating with eachother

**2. The nodes will run the commands by them self:**
```bash
    request
    reply
    leave
```

## Example

**client_Node log:**
```
2025/11/12 23:05:15 [Node3] Connected to peer localhost:5001
2025/11/12 23:05:15 [Node3] Connected to peer localhost:5002
2025/11/12 23:05:25 [Node3] Event handler started
2025/11/12 23:05:25 [Node3] Received REQUEST from Node1 (T=1)
2025/11/12 23:05:25 [Node3] Sent reply to Node1
2025/11/12 23:05:25 [Node3] Received REQUEST from Node2 (T=4)
2025/11/12 23:05:25 [Node3] Sent reply to Node2
2025/11/12 23:05:27 [Node3] Sent REQUEST to all peers (T=7)
2025/11/12 23:05:30 [Node3] Received LEAVE from Node2 (T=10)
2025/11/12 23:05:30 [Node3] Received REPLY from Node2 (T=11)
2025/11/12 23:05:30 [Node3] Received LEAVE from Node1 (T=12)
2025/11/12 23:05:30 [Node3] Received REPLY from Node1 (T=13)
2025/11/12 23:05:30 [Node3] Is in Critical Section
2025/11/12 23:05:30 [Node3] Doing work in cs...
2025/11/12 23:05:35 [Node3] Released CS and sent deferred replies
```

