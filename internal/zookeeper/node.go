package zookeeper

import (
	"context"
	"github.com/maxzhovtyj/distributed-bloom-filter/pkg/bloomproto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"time"
)

type Node struct {
	ID  []byte
	URI string

	client *grpc.ClientConn
	conn   bloomproto.DistributedBloomFilterClient
}

func NewNode(id []byte, uri string) (*Node, error) {
	insecureCredentials := insecure.NewCredentials()

	cl, err := grpc.NewClient(uri, grpc.WithTransportCredentials(insecureCredentials))
	if err != nil {
		return nil, err
	}

	log.Printf("Connection established %s — %s\n", id, uri)

	return &Node{
		ID:     append([]byte{}, id...),
		URI:    uri,
		client: cl,
		conn:   bloomproto.NewDistributedBloomFilterClient(cl),
	}, nil
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
