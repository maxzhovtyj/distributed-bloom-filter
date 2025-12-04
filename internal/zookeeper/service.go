package zookeeper

import (
	"encoding/json"
	"fmt"
	"github.com/maxzhovtyj/distributed-bloom-filter/pkg/bloomdata"
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

type ClusterOptions struct {
	BloomFilterPath string       `yaml:"bloomFilterPath"`
	Nodes           []NodeOption `yaml:"nodes"`
}

type NodeOption struct {
	ID             string `yaml:"id"`
	URI            string `yaml:"uri"`
	ReplicationURI string `yaml:"replicationUri"`
	GRPCPort       int    `yaml:"grpcPort"`
	HTTPPort       int    `yaml:"httpPort"`

	CPUCores     int `yaml:"cpuCores"`
	MemoryLimit  int `yaml:"memoryLimit"`
	NetworkLimit int `yaml:"networkLimit"`
}

type Service struct {
	cfg *ClusterOptions

	ring atomic.Pointer[Ring]
}

func New(cluster *ClusterOptions) (*Service, error) {
	r := NewRing()

	start := time.Now()

	for _, node := range cluster.Nodes {
		_, err := r.AddNode(node)
		if err != nil {
			return nil, fmt.Errorf("failed to add node %s: %w", node.ID, err)
		}
	}

	log.Printf("Cluster ring cretead in %s\n", time.Since(start))

	s := &Service{
		cfg: cluster,
	}

	s.ring.Store(r)

	return s, nil
}

func (s *Service) GetRing() *Ring {
	return s.ring.Load()
}

func (s *Service) Run() {
	start := time.Now()
	err := s.InitClusterBloomFilter()
	if err != nil {
		log.Panicf("Failed to init cluster bloom filter: %v", err)
	}

	log.Printf("Cluster Bloom Filter started in %s\n", time.Since(start))

	mux := http.NewServeMux()
	mux.HandleFunc("/debug", func(w http.ResponseWriter, r *http.Request) {
		uid := r.URL.Query().Get("uid")

		node := s.GetRing().GetNode([]byte(uid))
		if node == nil {
			_, _ = w.Write([]byte("node not found"))
			return
		}

		_, _ = w.Write([]byte(fmt.Sprintf("%s", node.ID)))
	})
	mux.HandleFunc("/add-node", func(w http.ResponseWriter, r *http.Request) {
		// TODO
		//nodeID := []byte(r.URL.Query().Get("nodeID"))
		//if len(nodeID) == 0 {
		//	_, _ = w.Write([]byte("node id not found"))
		//	return
		//}
		//
		//uri := r.URL.Query().Get("uri")
		//if len(uri) == 0 {
		//	_, _ = w.Write([]byte("uri not found"))
		//	return
		//}
		//
		//s.AddNode(nodeID, uri)
	})
	mux.HandleFunc("/remove-node", func(w http.ResponseWriter, r *http.Request) {
		nodeID := []byte(r.URL.Query().Get("nodeID"))
		if len(nodeID) == 0 {
			_, _ = w.Write([]byte("node id not found"))
			return
		}

		uri := r.URL.Query().Get("uri")
		if len(uri) == 0 {
			_, _ = w.Write([]byte("uri not found"))
			return
		}

		s.RemoveNode(nodeID)
	})
	mux.HandleFunc("/cluster", func(w http.ResponseWriter, r *http.Request) {
		ring := s.ring.Load()

		cluster, err := json.Marshal(ring)
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(cluster)
	})

	log.Println("Start serving http on :7000")
	if err = http.ListenAndServe(":7000", mux); err != nil {
		panic(err)
	}
}

func (s *Service) InitClusterBloomFilter() error {
	ring := s.GetRing()

	err := prepareClusterBloomFilter(s.cfg.BloomFilterPath, ring)
	if err != nil {
		return err
	}

	err = setupDistributedBloomFilter(s.cfg.BloomFilterPath, ring, map[string]struct{}{})
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) AddNode(opt NodeOption) {
	ring := s.GetRing()

	nodeToReBalance := ring.GetNode([]byte(opt.ID))

	newRing := NewRing()

	ring.CopyTo(newRing)
	newNode, err := newRing.AddNode(opt)
	if err != nil {
		panic(err)
	}

	nodesToSkip := make(map[string]struct{})

	for _, node := range ring.Nodes {
		if node == nodeToReBalance || node == newNode {
			continue
		}

		nodesToSkip[node.URI] = struct{}{}
	}

	err = prepareClusterBloomFilter(s.cfg.BloomFilterPath, newRing)
	if err != nil {
		return
	}

	err = setupDistributedBloomFilter(s.cfg.BloomFilterPath, newRing, nodesToSkip)
	if err != nil {
		return
	}

	s.ring.Store(newRing)

	log.Println("Nodes has been successfully rebalanced")
}

func (s *Service) RemoveNode(id []byte) {
	ring := s.GetRing()

	nextNode := ring.GetNodeByHash(Hash(id) + 1)

	newRing := NewRing()

	ring.CopyTo(newRing)
	newRing.RemoveNode(id)

	nodesToSkip := make(map[string]struct{})

	for _, node := range newRing.Nodes {
		if node == nextNode {
			continue
		}

		nodesToSkip[node.URI] = struct{}{}
	}

	err := prepareClusterBloomFilter(s.cfg.BloomFilterPath, newRing)
	if err != nil {
		return
	}

	err = setupDistributedBloomFilter(s.cfg.BloomFilterPath, newRing, nodesToSkip)
	if err != nil {
		return
	}

	s.ring.Store(newRing)
}

func prepareClusterBloomFilter(input string, ring *Ring) error {
	uidsPerNode, err := runEstimation(input, ring)
	if err != nil {
		return err
	}

	log.Println("===========Estimation results")
	for k, v := range uidsPerNode {
		log.Printf("~ %s: %d\n", k, v)
	}
	log.Println("===========")

	ring.nodesMX.RLock()
	for _, node := range ring.Nodes {
		elements := uidsPerNode[string(node.ID)]

		err = node.PrepareNode(elements)
		if err != nil {
			log.Println(err)
			continue
		}

		log.Printf("Prepared bloom filter %s — %d", node.ID, elements)
	}
	ring.nodesMX.RUnlock()

	return nil
}

func runEstimation(input string, ring *Ring) (map[string]int, error) {
	ch := make(chan []byte, 10000)

	uidsPerNode := make(map[string]int)
	errBuf := make(chan error, 1)

	go func(i string) {
		err := bloomdata.Read(input, ch)
		if err != nil {
			errBuf <- err
			return
		}

		close(errBuf)
	}(input)

	for uid := range ch {
		node := ring.GetNode(uid)
		uidsPerNode[string(node.ID)]++
	}

	err := <-errBuf
	if err != nil {
		return nil, err
	}

	return uidsPerNode, nil
}

func setupDistributedBloomFilter(input string, ring *Ring, nodesToSkip map[string]struct{}) error {
	ch := make(chan []byte, 10000)
	errBuf := make(chan error, 1)

	go func() {
		err := bloomdata.Read(input, ch)
		if err != nil {
			errBuf <- err
			return
		}

		close(errBuf)
	}()

	chanPerNode := make(map[string]chan []byte)
	defer func() {
		for _, c := range chanPerNode {
			if c != nil {
				close(c)
			}
		}
	}()

	ring.nodesMX.RLock()
	for _, node := range ring.Nodes {
		nodeID := string(node.ID)

		if _, ok := nodesToSkip[nodeID]; ok {
			log.Printf("Skip node: %s. No need to rebalance\n", nodeID)
			continue
		}

		nodeCh := make(chan []byte, 10000)

		go func() {
			err := node.InsertElements(nodeCh)
			if err != nil {
				log.Printf("Failed to insert elements: %v", err)
				return
			}
		}()

		chanPerNode[nodeID] = nodeCh
	}
	ring.nodesMX.RUnlock()

	for uid := range ch {
		node := ring.GetNode(uid)
		nodeID := string(node.ID)

		if _, ok := nodesToSkip[nodeID]; ok {
			continue
		}

		nodeCh := chanPerNode[nodeID]

		nodeCh <- uid
	}

	err := <-errBuf
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) Shutdown() {
	s.GetRing().Close()
}
