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
		log.Fatal(err)
	}

	clusterOptions := new(zookeeper.ClusterOptions)

	if err = yaml.Unmarshal(raw, clusterOptions); err != nil {
		log.Fatal(err)
	}

	service := zookeeper.New(clusterOptions)

	service.Run()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	<-ctx.Done()

	log.Println("Shutting down...")
	service.Shutdown()
}
