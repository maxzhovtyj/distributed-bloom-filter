package main

import (
	"context"
	"github.com/maxzhovtyj/distributed-bloom-filter/internal/zookeeper"
	"log"
	"os/signal"
	"syscall"
)

func main() {
	service := zookeeper.New()
	service.InitRing()
	service.Run()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	<-ctx.Done()

	log.Println("Shutting down...")
	service.Shutdown()
}
