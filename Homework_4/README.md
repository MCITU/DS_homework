# Chit Chat - Distributed Chat Service

A distributed chat service using Go and gRPC with Lamport logical timestamps.

## Architecture
- **Client-server architecture**: One server, multiple clients
- **Server-side streaming**: Server pushes broadcast messages to clients
- **Lamport timestamps**: Logical clock for event ordering

## Prerequisites

- Go 1.21+
- Protocol Buffers compiler (optional, for regenerating code)
- gRPC Go plugins (optional, for regenerating code)

## Usage

**1. Start the server:**
```bash
go run server/server.go
```

**2. Start clients (in separate terminals):**
```bash
go run client/client.go Alice
go run client/client.go Bob
go run client/client.go Charlie
```

**3. Send messages:**
Type a message and press Enter. All clients will receive it with a Lamport timestamp.

**4. Leave:**
Type `/leave` to disconnect.

## Example

**Server log:**
```
[Server] Starting up
[Server] Listening on port 5000
[Server] Client Alice joined
[Server] Broadcasting: Participant Alice joined... (Lamport: 1)
```

**Client output:**
```
Connected as Alice (type '/leave' to exit)
[Lamport: 1] Participant Alice joined Chit Chat at Lamport time 1
[Lamport: 2] Participant Bob joined Chit Chat at Lamport time 2
Hello!
[Lamport: 3] Alice: Hello!
```

## Logs

- `server.log` - Server events
- `client_<name>.log` - Client events
