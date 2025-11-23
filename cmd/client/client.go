package main

import (
	"context"
	"fmt"
	"github.com/maxzhovtyj/distributed-bloom-filter/pkg/bloomproto"
	"google.golang.org/grpc"
)

func main() {
	cl, err := grpc.NewClient("localhost:8001", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	conn := bloomproto.NewDistributedBloomFilterClient(cl)

	res, err := conn.Test(context.Background(), &bloomproto.TestRequest{
		Key: []byte("7b84e6b5-82ea-47a6-bc6a-bc019fb69fad"),
	})

	if err != nil {
		panic(err)
	}

	fmt.Println(res)
}
