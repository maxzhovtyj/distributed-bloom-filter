package main

import (
	"flag"
	"github.com/maxzhovtyj/distributed-bloom-filter/internal/bloomnode"
)

var (
	masterNodeURI = flag.String("masterNodeURI", "", "The URI of the master node to connect to")
	bfdPath       = flag.String("bfdPath", "/root/distributed-bloom-filter/bloom_filter.bfd", "Path to the bloom filter data file")
)

func main() {
	flag.Parse()

	bloomnode.Run(*masterNodeURI, *bfdPath)
}
