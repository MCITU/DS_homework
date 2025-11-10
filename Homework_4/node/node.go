package node

/*
	Have an IP which is the channel that it retrieve data and listens on
	node struct that has the parameters of both a server and client
	request access to the critical section

*/

import (
	proto "ITUserver/grpc"
	"fmt"

	"google.golang.org/grpc"
)

var (
	ID int // server IP ?
)

func main() {
	fmt.Println("Hello World!")

}

type node struct {
	NodeNr string
	conn   *grpc.ClientConn
	client proto.ITUDatabaseClient
	ctx    context.Context
	cancel context.CancelFunc
}
