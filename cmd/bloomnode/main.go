package main

import (
	"flag"
	"github.com/maxzhovtyj/distributed-bloom-filter/internal/bloomnode"
)

var masterNodeURI = flag.String("masterNodeURI", "", "The URI of the master node to connect to")

func main() {
	flag.Parse()

	bloomnode.Run(*masterNodeURI)
}
