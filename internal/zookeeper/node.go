package zookeeper

import (
	"context"
	"fmt"
	"github.com/maxzhovtyj/distributed-bloom-filter/pkg/bloomproto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"time"
)

type Node struct {
	ID         []byte
	URI        string
	GRPCPort   int
	HTTPPort   int
	ReplicaURI string

	client *grpc.ClientConn
	conn   bloomproto.DistributedBloomFilterClient
}

func NewNode(options NodeOption) (*Node, error) {
	return &Node{
		ID:         append([]byte{}, options.ID...),
		URI:        options.URI,
		GRPCPort:   options.GRPCPort,
		HTTPPort:   options.HTTPPort,
		ReplicaURI: options.ReplicationURI,
	}, nil
}

func (n *Node) Init() error {
	insecureCredentials := insecure.NewCredentials()

	grpcURI := fmt.Sprintf("%s:%d", n.URI, n.GRPCPort)

	cl, err := grpc.NewClient(grpcURI, grpc.WithTransportCredentials(insecureCredentials))
	if err != nil {
		return err
	}

	log.Printf("Connection established %s — %s\n", n.ID, grpcURI)

	n.client = cl
	n.conn = bloomproto.NewDistributedBloomFilterClient(cl)

	return nil
}

func (n *Node) Test(element []byte) (bool, error) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()

	response, err := n.conn.Test(ctx, &bloomproto.TestRequest{
		Key: element,
	})
	if err != nil {
		return false, err
	}

	return response.IsPresent, nil
}

func (n *Node) PrepareNode(elementsCount int) error {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()

	req := &bloomproto.PrepareBloomFilterRequest{
		ElementsCount: int64(elementsCount),
	}

	_, err := n.conn.PrepareBloomFilter(ctx, req)
	if err != nil {
		return err
	}

	return nil
}

func (n *Node) InsertElements(elements <-chan []byte) error {
	insertStream, err := n.conn.Insert(context.Background())
	if err != nil {
		return err
	}

	for uid := range elements {
		err = insertStream.SendMsg(&bloomproto.InsertRequest{
			Key: uid,
		})
		if err != nil {
			return err
		}
	}

	_, err = insertStream.CloseAndRecv()
	if err != nil {
		return err
	}

	return nil
}

func (n *Node) Close() {
	err := n.client.Close()
	if err != nil {
		log.Printf("Failed to close client connection: %v", err)
	}
}

func (n *Node) CopyTo(node *Node) {
	node.ID = append(node.ID[:0], n.ID...)
	node.URI = n.URI
	node.client = n.client
	node.conn = n.conn
}
