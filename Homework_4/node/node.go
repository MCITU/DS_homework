package node

/*
	Have an IP which is the channel that it retrieve data and listens on
	node struct that has the parameters of both a server and client
	request access to the critical section

*/

import (
	proto "ITUserver/grpc"
	"context"
	"fmt"
	"log"
	"os"

	apiv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	ID int // server IP ?
)

func main() {
	fmt.Println("Hello World!")

}

type node struct {
	name   string
	conn   *grpc.ClientConn
	client proto.ITUDatabaseClient
	ctx    context.Context
	cancel context.CancelFunc
	srv    *grpc.Server
	peers  map[string]proto.ITUDatabaseClient
}

type ituServer struct {
	proto.UnimplementedITUDatabaseServer
	n *node
}

func (s *ituServer) SendRequest(ctx context.Context, r *proto.AccessRequest) (*proto.Confirm, error) {
	return &proto.Confirm{}, nil
}

func (s *ituServer) SendReply(ctx context.Context, r *proto.AccessReply) (*proto.Confirm, error) {
	return &proto.Confirm{}, nil
}

func (s *ituServer) SendLeave(ctx context.Context, r *proto.AccessRequest) (*proto.Confirm, error) {
	return &proto.Confirm{}, nil
}

func createNode(name string, connection string) (*node, error) {
	logFile, err := os.OpenFile(fmt.Sprintf("client_%s.log", name), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return nil, err
	}
	log.SetOutput(logFile)

	conn, err := grpc.Dial("localhost"+connection, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &node{
		name:   name,
		conn:   conn,
		client: proto.NewITUDatabaseClient(conn),
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

func newServer(serverId int64) {

}
