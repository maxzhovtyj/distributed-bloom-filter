package zookeeper

import (
	"encoding/json"
	"fmt"
	"github.com/maxzhovtyj/distributed-bloom-filter/pkg/bloomdata"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

var (
	bloomFilterInitializationLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "bloom_filter_initialization_latency",
		Help:    "The total latency of bloom filter initialization",
		Buckets: prometheus.ExponentialBuckets(100, 1.6, 10),
	})
	bloomFilterPrepareLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "bloom_filter_prepare_latency",
		Help:    "The total latency of bloom filter preparation",
		Buckets: prometheus.ExponentialBuckets(100, 1.6, 10),
	})
	bloomFilterAddNodeLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "bloom_filter_add_node_latency",
		Help:    "The total latency of bloom filter add node operation",
		Buckets: prometheus.ExponentialBuckets(100, 1.6, 10),
	})
	bloomFilterRemoveNodeLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "bloom_filter_remove_node_latency",
		Help:    "The total latency of bloom filter remove node operation",
		Buckets: prometheus.ExponentialBuckets(100, 1.6, 10),
	})

	bloomFilterTotalSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bloom_filter_total_size",
		Help: "The total size of bloom filter in bytes",
	})
)

type ClusterOptions struct {
	BloomFilterPath string       `yaml:"bloomFilterPath"`
	Nodes           []NodeOption `yaml:"nodes"`
}

type NodeOption struct {
	ID             string `yaml:"id" json:"id"`
	URI            string `yaml:"uri" json:"uri"`
	ReplicationURI string `yaml:"replicationUri" json:"replicationUri"`
	GRPCPort       int    `yaml:"grpcPort" json:"grpcPort"`
	HTTPPort       int    `yaml:"httpPort" json:"httpPort"`
	VMNodes        int    `yaml:"vmNodes" json:"vmNodes"`

	CPUCores     int `yaml:"cpuCores" json:"cpuCores"`
	MemoryLimit  int `yaml:"memoryLimit" json:"memoryLimit"`
	NetworkLimit int `yaml:"networkLimit" json:"networkLimit"`
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

	log.Printf("Cluster ring created in %s\n", time.Since(start))

	s := &Service{
		cfg: cluster,
	}

	s.ring.Store(r)

	return s, nil
}

func (s *Service) GetRing() *Ring {
	return s.ring.Load()
}

func (s *Service) RunHTTPHandler() {
	mux := http.NewServeMux()

	mux.HandleFunc("/init-cluster", func(w http.ResponseWriter, r *http.Request) {
		go func() {
			start := time.Now()

			err := s.InitClusterBloomFilter()
			if err != nil {
				log.Panicf("Failed to init cluster bloom filter: %v", err)
			}

			log.Printf("Cluster Bloom Filter initialized in %s\n", time.Since(start))
		}()
	})
	mux.HandleFunc("/debug", func(w http.ResponseWriter, r *http.Request) {
		uid := r.URL.Query().Get("uid")

		node := s.GetRing().GetNode([]byte(uid))
		if node == nil {
			_, _ = w.Write([]byte("node not found"))
			return
		}

		_, _ = w.Write([]byte(fmt.Sprintf("%s", node.ID)))
	})
	mux.HandleFunc("POST /add-node", func(w http.ResponseWriter, r *http.Request) {
		var node NodeOption

		err := json.NewDecoder(r.Body).Decode(&node)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		go s.AddNode(node)
	})
	mux.HandleFunc("/remove-node", func(w http.ResponseWriter, r *http.Request) {
		nodeID := []byte(r.URL.Query().Get("nodeID"))
		if len(nodeID) == 0 {
			_, _ = w.Write([]byte("node id not found"))
			return
		}

		go s.RemoveNode(nodeID)
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
	mux.Handle("/metrics", promhttp.Handler())

	log.Println("Start serving http on :7000")
	if err := http.ListenAndServe(":7000", mux); err != nil {
		panic(err)
	}
}

func (s *Service) Run() {
	go s.RunHTTPHandler()

	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "zookeeper_cluster_physical_size",
		Help: "Amount of nodes in cluster",
	}, func() float64 {
		r := s.GetRing()

		return float64(r.PhysicalNodes())
	})
	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "zookeeper_cluster_total_size",
		Help: "Amount of nodes in cluster",
	}, func() float64 {
		r := s.GetRing()

		return float64(r.Len())
	})
}

func (s *Service) InitClusterBloomFilter() error {
	start := time.Now()

	ring := s.GetRing()

	err := prepareClusterBloomFilter(s.cfg.BloomFilterPath, ring)
	if err != nil {
		return err
	}

	err = setupDistributedBloomFilter(s.cfg.BloomFilterPath, ring, map[string]struct{}{})
	if err != nil {
		return err
	}

	bloomFilterInitializationLatency.Observe(time.Since(start).Seconds())

	return nil
}

func (s *Service) AddNode(opt NodeOption) {
	start := time.Now()
	log.Println("Start adding node:", string(opt.ID))

	ring := s.GetRing()
	newRing := NewRing()
	ring.CopyTo(newRing)

	_, err := newRing.AddNode(opt)
	if err != nil {
		panic(err)
	}

	err = prepareClusterBloomFilter(s.cfg.BloomFilterPath, newRing)
	if err != nil {
		log.Printf("Add Node failed to prepare cluster: %v ", err)
		return
	}

	err = setupDistributedBloomFilter(s.cfg.BloomFilterPath, newRing, map[string]struct{}{})
	if err != nil {
		log.Printf("Add Node failed to setup distributed bloom filter: %v", err)
		return
	}

	s.ring.Store(newRing)

	bloomFilterAddNodeLatency.Observe(time.Since(start).Seconds())
	log.Println("Nodes has been successfully rebalanced")
}

func (s *Service) RemoveNode(id []byte) {
	log.Println("Start removing node:", string(id))

	start := time.Now()
	ring := s.GetRing()

	newRing := NewRing()

	ring.CopyTo(newRing)
	newRing.RemoveNode(id)

	err := prepareClusterBloomFilter(s.cfg.BloomFilterPath, newRing)
	if err != nil {
		log.Printf("Remove Node failed to prepare cluster: %v\n", err)
		return
	}

	err = setupDistributedBloomFilter(s.cfg.BloomFilterPath, newRing, make(map[string]struct{}))
	if err != nil {
		log.Printf("Remove Node failed to setup distributed bloom filter: %v\n", err)
		return
	}

	s.ring.Store(newRing)

	bloomFilterRemoveNodeLatency.Observe(time.Since(start).Seconds())
	log.Println("Node has been successfully deleted")
}

func prepareClusterBloomFilter(input string, ring *Ring) error {
	start := time.Now()

	uidsPerNode, err := runEstimation(input, ring)
	if err != nil {
		return err
	}

	//log.Println("===========Estimation results")
	//for k, v := range uidsPerNode {
	//	log.Printf("~ %s: %d\n", k, v)
	//}
	//log.Println("===========")

	totalSize := 0
	for _, v := range uidsPerNode {
		totalSize += v
	}
	bloomFilterTotalSize.Set(float64(totalSize))

	ring.nodesMX.RLock()
	for _, node := range ring.Nodes {
		if node.IsVM {
			continue
		}

		elements := uidsPerNode[string(node.ID)]

		err = node.PrepareNode(elements)
		if err != nil {
			log.Println(err)
			continue
		}

		log.Printf("Prepared bloom filter %s — %d", node.ID, elements)
	}
	ring.nodesMX.RUnlock()

	bloomFilterPrepareLatency.Observe(time.Since(start).Seconds())

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

	var id []byte

	for uid := range ch {
		node := ring.GetNode(uid)
		if node.IsVM {
			id = append(id[:0], node.PhysicalNodeID...)
		} else {
			id = append(id[:0], node.ID...)
		}

		uidsPerNode[string(id)]++
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
		if node.IsVM {
			continue
		}

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

	var nodeID string

	for uid := range ch {
		node := ring.GetNode(uid)

		if node.IsVM {
			nodeID = string(node.PhysicalNodeID)
		} else {
			nodeID = string(node.ID)
		}

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
