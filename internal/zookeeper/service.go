package zookeeper

import (
	"fmt"
	"github.com/maxzhovtyj/distributed-bloom-filter/pkg/bloomdata"
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

const input = "/Users/maksymzhovtaniuk/Desktop/Дисертація/distributed-bloom-filter/data/idfa1.csv"

type Service struct {
	ring atomic.Pointer[Ring]
}

func New() *Service {
	r := NewRing()
	s := &Service{}
	s.ring.Store(r)

	return s
}

func (s *Service) GetRing() *Ring {
	return s.ring.Load()
}

func (s *Service) Run() {
	start := time.Now()
	s.InitClusterBloomFilter()
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

		s.AddNode(nodeID, uri)
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

	if err := http.ListenAndServe(":7000", mux); err != nil {
		panic(err)
	}
}

func (s *Service) InitClusterBloomFilter() {
	ring := s.GetRing()

	prepareClusterBloomFilter(ring)

	setupDistributedBloomFilter(ring, map[string]struct{}{})
}

func (s *Service) AddNode(id []byte, uri string) {
	ring := s.GetRing()

	nodeToReBalance := ring.GetNode(id)

	newRing := NewRing()

	ring.CopyTo(newRing)
	newNode := newRing.AddNode(id, uri)

	nodesToSkip := make(map[string]struct{})

	for _, node := range ring.nodes {
		if node == nodeToReBalance || node == newNode {
			continue
		}

		nodesToSkip[node.URI] = struct{}{}
	}

	prepareClusterBloomFilter(newRing)

	setupDistributedBloomFilter(newRing, nodesToSkip)

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

	for _, node := range newRing.nodes {
		if node == nextNode {
			continue
		}

		nodesToSkip[node.URI] = struct{}{}
	}

	prepareClusterBloomFilter(newRing)
	setupDistributedBloomFilter(newRing, nodesToSkip)

	s.ring.Store(newRing)
}

func prepareClusterBloomFilter(ring *Ring) {
	uidsPerNode := runEstimation(ring)

	fmt.Println("===========Estimation results")
	for k, v := range uidsPerNode {
		fmt.Printf("~ %s: %d\n", k, v)
	}
	fmt.Println("===========")

	ring.nodesMX.RLock()
	for _, node := range ring.nodes {
		elements := uidsPerNode[string(node.ID)]

		err := node.PrepareNode(elements)
		if err != nil {
			log.Println(err)
			continue
		}

		log.Printf("Prepared bloom filter %s — %d", node.ID, elements)
	}
	ring.nodesMX.RUnlock()
}

func (s *Service) InitRing() {
	start := time.Now()

	ring := s.GetRing()

	ring.AddNode([]byte("server1"), "localhost:8000")
	ring.AddNode([]byte("server2"), "localhost:8001")
	ring.AddNode([]byte("server3"), "localhost:8002")
	ring.AddNode([]byte("server4"), "localhost:8003")

	log.Printf("Done InitRing in %s\n", time.Since(start))
}

func runEstimation(ring *Ring) map[string]int {
	ch := make(chan []byte, 10000)

	uidsPerNode := make(map[string]int)

	go func() {
		err := bloomdata.Read(input, ch)
		if err != nil {
			log.Println(err)
			return
		}
	}()

	for uid := range ch {
		node := ring.GetNode(uid)
		uidsPerNode[string(node.ID)]++
	}

	return uidsPerNode
}

func setupDistributedBloomFilter(ring *Ring, nodesToSkip map[string]struct{}) {
	ch := make(chan []byte, 10000)

	go func() {
		err := bloomdata.Read(input, ch)
		if err != nil {
			log.Println(err)
			return
		}
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
	for _, node := range ring.nodes {
		nodeID := string(node.ID)

		if _, ok := nodesToSkip[nodeID]; ok {
			log.Printf("Skip node: %s. No need to rebalance\n", nodeID)
			continue
		}

		nodeCh := make(chan []byte, 10000)

		go func() {
			err := node.InsertElements(nodeCh)
			if err != nil {
				log.Println(err)
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

	return
}

func (s *Service) Shutdown() {
	s.GetRing().Close()
}
