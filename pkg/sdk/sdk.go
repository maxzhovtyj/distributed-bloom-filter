package sdk

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync/atomic"
	"time"

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

func getNewRing(uri string) (*ring.Ring, error) {
	r, err := syncDistributedBloomFilterRing(uri)
	if err != nil {
		return nil, err
	}

	for _, node := range r.Nodes {
		if node.IsVM {
			continue
		}

		err = node.Init()
		if err != nil {
			return nil, fmt.Errorf("failed to init node: %v", err)
		}

		log.Printf("Node %s initialized\n", node.ID)
	}

	for _, node := range r.Nodes {
		if !node.IsVM {
			continue
		}

		n := r.Nodes[ring.Hash(node.PhysicalNodeID)]
		n.CopyConnTo(node)
	}

	return r, nil
}

func (d *DistributedBloomFilter) Init() error {
	newRing, err := getNewRing(d.ZookeeperURI)
	if err != nil {
		return err
	}

	d.ring.Store(newRing)

	go d.runSyncRingWorker()

	return nil
}

func (d *DistributedBloomFilter) runSyncRingWorker() {
	for range time.Tick(5 * time.Minute) {
		newRing, err := getNewRing(d.ZookeeperURI)
		if err != nil {
			log.Println("Failed to sync ring", err)
			continue
		}

		d.ring.Store(newRing)
	}
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
	uri := fmt.Sprintf("http://%s/cluster", host)

	resp, err := http.Get(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to do http request for sync: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to sync bloom filter: %s", resp.Status)
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
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if err = json.Unmarshal(raw, &r); err != nil {
		return nil, err
	}

	return r, nil
}
