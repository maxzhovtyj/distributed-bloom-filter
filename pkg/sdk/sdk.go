package sdk

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync/atomic"

	ring "github.com/maxzhovtyj/distributed-bloom-filter/internal/zookeeper"
)

type DistributedBloomFilter struct {
	ZookeeperURI string

	ring atomic.Pointer[ring.Ring]
}

func NewDistributedBloomFilter(uri string) *DistributedBloomFilter {
	return &DistributedBloomFilter{
		ZookeeperURI: uri,
	}
}

func (d *DistributedBloomFilter) Init() error {
	r, err := syncDistributedBloomFilterRing(d.ZookeeperURI)
	if err != nil {
		return err
	}

	for _, node := range r.Nodes {
		err = node.Init()
		if err != nil {
			log.Panic(err)
		}
	}

	d.ring.Store(r)

	return nil
}

func (d *DistributedBloomFilter) Test(element []byte) (bool, error) {
	r := d.ring.Load()

	n := r.GetNode(element)

	test, err := n.Test(element)
	if err != nil {
		return false, err
	}

	return test, nil
}

func syncDistributedBloomFilterRing(host string) (*ring.Ring, error) {
	uri := fmt.Sprintf("http://%s/sync", host)

	resp, err := http.Get(uri)
	if err != nil {
		return nil, err
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			log.Println("Failed to close bloom filter for sync", err)
		}
	}()

	var r *ring.Ring

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(raw, &r); err != nil {
		return nil, err
	}

	return r, nil
}
