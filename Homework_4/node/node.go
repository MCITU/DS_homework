package node

import (
	pb "ITUserver/grpc"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type event struct {
	kind string // "request", "reply", "leave"
	data interface{}
}

type node struct {
	Name             string
	Port             string
	Conn             *grpc.ClientConn
	Client           pb.ITUDatabaseClient
	Ctx              context.Context
	Cancel           context.CancelFunc
	Srv              *grpc.Server
	Peers            map[string]pb.ITUDatabaseClient // key: node name
	Events           chan event
	ReplyCount       int
	DeferredReplies  []pb.AccessRequest
	LamportClock     int64
	RequestTimestamp int64
	InCriticalSec    bool
}

func CreateNode(name string, port string) (*node, error) {
	logFile, err := os.OpenFile(fmt.Sprintf("client_%s.log", name), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return nil, err
	}
	log.SetOutput(logFile)

	ctx, cancel := context.WithCancel(context.Background())
	n := &node{
		Name:            name,
		Port:            port,
		Ctx:             ctx,
		Cancel:          cancel,
		Peers:           make(map[string]pb.ITUDatabaseClient),
		Events:          make(chan event, 100),
		LamportClock:    0,
		InCriticalSec:   false,
		DeferredReplies: make([]pb.AccessRequest, 0),
	}
	return n, nil
}

func (n *node) incrementClock() int64 {
	n.LamportClock++
	return n.LamportClock
}

func (n *node) updateClock(received int64) {
	if received >= n.LamportClock {
		n.LamportClock = received + 1
	} else {
		n.incrementClock()
	}
}

func (n *node) AddPeer(id, port string) error {
	address := port
	if !strings.Contains(port, ":") {
		address = "localhost:" + port
	}

	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to peer %s: %v", address, err)
	}
	client := pb.NewITUDatabaseClient(conn)
	n.Peers[id] = client
	log.Printf("[%s] Connected to peer %s", n.Name, address)
	return nil
}

// functions as Ricart-agrawala algo
func (n *node) handleEvents() {
	if n.Events == nil {
		log.Fatalf("[%s] FATAL: Events channel is nil", n.Name)
	}
	if n.Peers == nil {
		log.Fatalf("[%s] FATAL: Peers map is nil", n.Name)
	}
	log.Printf("[%s] Event handler started", n.Name)

	for e := range n.Events {
		switch e.kind {
		case "request":
			req := e.data.(*pb.AccessRequest)
			n.updateClock(int64(req.LamportTimestamp))
			log.Printf("[%s] Received REQUEST from %s (T=%d)\n", n.Name, req.NodeId, req.LamportTimestamp)
			shouldDefer := n.InCriticalSec || (n.RequestTimestamp != 0 && (req.LamportTimestamp < uint64(n.RequestTimestamp) ||
				(req.LamportTimestamp == uint64(n.RequestTimestamp) && req.NodeId < n.Name)))
			if shouldDefer {
				n.DeferredReplies = append(n.DeferredReplies, *req)
				log.Printf("[%s] DeferredReplies to %s", n.Name, req.NodeId)
			} else {
				rep := &pb.AccessReply{
					NodeId:           n.Name,
					LamportTimestamp: uint64(n.incrementClock()),
				}
				_, err := n.Peers[req.NodeId].SendReply(n.Ctx, rep)
				if err != nil {
					log.Printf("[%s] Failed to send reply to %s: %v", n.Name, req.NodeId, err)
				} else {
					log.Printf("[%s] Sent reply to %s", n.Name, req.NodeId)
				}

			}

		case "reply":
			rep := e.data.(*pb.AccessReply)
			n.updateClock(int64(rep.LamportTimestamp))
			n.ReplyCount++
			log.Printf("[%s] Received REPLY from %s (T=%d)\n", n.Name, rep.NodeId, rep.LamportTimestamp)

			if n.ReplyCount == len(n.Peers) {
				n.InCriticalSec = true
				log.Printf("[%s] Is in Critical Section", n.Name)
			}

		case "leave":
			lv := e.data.(*pb.LeaveNotice)
			n.updateClock(int64(lv.LamportTimestamp))
			log.Printf("[%s] Received LEAVE from %s (T=%d)\n", n.Name, lv.NodeId, lv.LamportTimestamp)
		}
	}
}

func (n *node) RequestAccess() error {
	n.RequestTimestamp = n.incrementClock()
	n.ReplyCount = 0
	n.InCriticalSec = false

	req := &pb.AccessRequest{
		NodeId:           n.Name,
		LamportTimestamp: uint64(n.RequestTimestamp),
	}

	for _, peer := range n.Peers {
		_, err := peer.SendRequest(n.Ctx, req)
		if err != nil {
			log.Println("Failed to send request to peer:", err)
		}
	}
	log.Printf("[%s] Sent REQUEST to all peers (T=%d)", n.Name, n.RequestTimestamp)
	return nil

}

func (n *node) ReleaseAccess() error {
	n.incrementClock()
	leave := &pb.LeaveNotice{
		NodeId:           n.Name,
		LamportTimestamp: uint64(n.LamportClock),
	}

	for _, peer := range n.Peers {
		_, err := peer.SendLeave(n.Ctx, leave)
		if err != nil {
			log.Println("Failed to send 'leave' to peer:", err)
		}

	}

	for _, r := range n.DeferredReplies {
		rep := &pb.AccessReply{
			NodeId:           n.Name,
			LamportTimestamp: uint64(n.incrementClock()),
		}
		_, err := n.Peers[r.NodeId].SendReply(n.Ctx, rep)
		if err != nil {
			log.Printf("[%s] Failed to send reply to peer %s: %v", n.Name, r.NodeId, err)
		} else {
			log.Printf("[%s] Sent deferred REPLY tp %s", n.Name, r.NodeId)
		}
	}

	n.DeferredReplies = nil
	n.InCriticalSec = false
	log.Printf("[%s] Released CS and sent deferred replies", n.Name)
	return nil
}

func (n *node) Start() error {
	srv, err := newServer(n.Port, n)
	if err != nil {
		return err
	}

	n.Srv = srv

	time.Sleep(30 * time.Second)

	go n.handleEvents()
	return nil
}
