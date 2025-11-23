package zookeeper

import (
	"fmt"
	"github.com/maxzhovtyj/distributed-bloom-filter/pkg/bloomdata"
	"log"
	"net/http"
	"time"
)

type Service struct {
	ring *Ring
}

func New() *Service {
	return &Service{
		ring: NewRing(),
	}
}

func (s *Service) Run() {
	start := time.Now()
	s.InitClusterBloomFilter()
	log.Printf("Cluster Bloom Filter started in %s\n", time.Since(start))

	mux := http.NewServeMux()
	mux.HandleFunc("/debug", func(w http.ResponseWriter, r *http.Request) {
		uid := r.URL.Query().Get("uid")

		node := s.ring.GetNode([]byte(uid))
		if node == nil {
			_, _ = w.Write([]byte("node not found"))
			return
		}

		_, _ = w.Write([]byte(fmt.Sprintf("%s", node.ID)))
	})

	if err := http.ListenAndServe(":7000", mux); err != nil {
		panic(err)
	}
}

func (s *Service) InitClusterBloomFilter() {
	s.prepareClusterBloomFilter()

	s.setupDistributedBloomFilter()
}

func (s *Service) prepareClusterBloomFilter() {
	uidsPerNode := s.runEstimation()

	fmt.Println("===========Estimation results")
	for k, v := range uidsPerNode {
		fmt.Printf("~ %s: %d\n", k, v)
	}
	fmt.Println("===========")

	s.ring.nodesMX.RLock()
	for _, node := range s.ring.nodes {
		elements := uidsPerNode[string(node.ID)]

		err := node.PrepareNode(elements)
		if err != nil {
			log.Println(err)
			continue
		}

		log.Printf("Prepared bloom filter %s — %d", node.ID, elements)
	}
	s.ring.nodesMX.RUnlock()
}

func (s *Service) InitRing() {
	start := time.Now()

	s.ring.AddNode([]byte("server1"), "localhost:8000")
	s.ring.AddNode([]byte("server2"), "localhost:8001")
	s.ring.AddNode([]byte("server3"), "localhost:8002")
	s.ring.AddNode([]byte("server4"), "localhost:8003")

	log.Printf("Done InitRing in %s\n", time.Since(start))
}

func (s *Service) runEstimation() map[string]int {
	input := "/Users/maksymzhovtaniuk/Desktop/Дисертація/distributed-bloom-filter/data/idfa1.csv"
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
		node := s.ring.GetNode(uid)
		uidsPerNode[string(node.ID)]++
	}

	return uidsPerNode
}

func (s *Service) setupDistributedBloomFilter() {
	input := "/Users/maksymzhovtaniuk/Desktop/Дисертація/distributed-bloom-filter/data/idfa1.csv"
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

	s.ring.nodesMX.RLock()
	for _, node := range s.ring.nodes {
		nodeCh := make(chan []byte, 10000)

		go func() {
			err := node.InsertElements(nodeCh)
			if err != nil {
				log.Println(err)
				return
			}
		}()

		chanPerNode[string(node.ID)] = nodeCh
	}
	s.ring.nodesMX.RUnlock()

	for uid := range ch {
		node := s.ring.GetNode(uid)

		nodeCh := chanPerNode[string(node.ID)]

		nodeCh <- uid
	}

	return
}

func (s *Service) Shutdown() {
	s.ring.Close()
}
