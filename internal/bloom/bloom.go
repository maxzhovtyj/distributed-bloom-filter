package dbf

import (
	"encoding/csv"
	"errors"
	"fmt"
	"github.com/bits-and-blooms/bloom/v3"
	"github.com/maxzhovtyj/distributed-bloom-filter/pkg/consistenthash"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

var (
	ErrNodeNotFound    = errors.New("node not found")
	ErrNodeBFDNotFound = errors.New("node bfd not found")
)

var s *Service

type Service struct {
	blooms   []*bloom.BloomFilter
	bloomsMX sync.RWMutex

	ring *consistenthash.Ring
}

func (s *Service) AddBloom(bfd *bloom.BloomFilter) {
	s.bloomsMX.Lock()
	s.blooms = append(s.blooms, bfd)
	s.bloomsMX.Unlock()
}

func Init() {
	s = &Service{
		blooms: make([]*bloom.BloomFilter, 0),
		ring:   getRing(),
	}
}

func Collect() {
	start := time.Now()

	f, err := os.Open("/Users/maksymzhovtaniuk/Desktop/Дисертація/distributed-bloom-filter/data/idfa1.csv")
	if err != nil {
		panic(err)
	}

	countPerNode := make(map[string]int)

	reader := csv.NewReader(f)
	reader.Comma = '\t'

	for {
		read, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			panic(err)
		}

		if len(read) != 2 {
			continue
		}

		uid := []byte(read[0])
		node := s.ring.GetNode(uid)
		if node == nil {
			log.Fatalf("node not found for key '%s'", uid)
		}

		node.BFD.AddString(read[0])
		countPerNode[string(node.ID)]++
	}

	fmt.Printf("Done collecting bloom in %s\n", time.Since(start))
	for k, count := range countPerNode {
		fmt.Printf("NODE %s: %d\n", k, count)
	}
}

func getRing() *consistenthash.Ring {
	ring := consistenthash.NewRing()

	ring.AddNode([]byte("server1"))
	ring.AddNode([]byte("server2"))
	ring.AddNode([]byte("server3"))
	ring.AddNode([]byte("server4"))

	return ring
}

func Test(key []byte) (bool, []byte, error) {
	node := s.ring.GetNode(key)
	if node == nil {
		return false, nil, ErrNodeNotFound
	}

	if node.BFD == nil {
		return false, nil, ErrNodeBFDNotFound
	}

	return node.BFD.Test(key), node.ID, nil
}
