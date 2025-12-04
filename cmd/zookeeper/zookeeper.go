package main

import (
	"context"
	"flag"
	"github.com/maxzhovtyj/distributed-bloom-filter/internal/zookeeper"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var configPath = flag.String("config", "./cmd/zookeeper/local.yml", "Path to zookeeper config file")

func main() {
	flag.Parse()

	raw, err := os.ReadFile(*configPath)
	if err != nil {
		log.Fatalf("Failed to open config: %v", err)
		return
	}

	log.Println("Starting zookeeper service...")

	log.Printf("Config: %s\n", raw)

	clusterOptions := new(zookeeper.ClusterOptions)

	if err = yaml.Unmarshal(raw, clusterOptions); err != nil {
		log.Fatalf("Failed to unmarshal config: %v", err)
		return
	}

	service, err := zookeeper.New(clusterOptions)
	if err != nil {
		log.Fatalf("Failed to create zookeeper service: %v", err)
		return
	}

	go service.Run()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	<-ctx.Done()

	log.Println("Shutting down...")
	service.Shutdown()
}
