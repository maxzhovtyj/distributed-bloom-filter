package consistenthash

import (
	"github.com/bits-and-blooms/bloom/v3"
	"github.com/spaolacci/murmur3"
	"slices"
	"sync"
)

type Node struct {
	ID  []byte
	BFD *bloom.BloomFilter
}

type Ring struct {
	nodes   map[uint32]*Node
	nodesMX sync.RWMutex

	hashes []uint32
}

func NewRing() *Ring {
	return &Ring{
		nodes:  make(map[uint32]*Node),
		hashes: []uint32{},
	}
}

func Hash(key []byte) uint32 {
	return murmur3.Sum32(key)
}

func (chr *Ring) AddNode(id []byte) {
	chr.nodesMX.Lock()
	defer chr.nodesMX.Unlock()

	hash := Hash(id)
	chr.nodes[hash] = &Node{
		ID:  id,
		BFD: bloom.NewWithEstimates(25025, 0.01),
	}

	chr.hashes = append(chr.hashes, hash)

	slices.Sort(chr.hashes)
}

func (chr *Ring) GetNextNodeIndex(hash uint32) int {
	if len(chr.hashes) == 0 {
		return -1
	}

	for i, h := range chr.hashes {
		if h > hash {
			return i
		}
	}

	return 0
}

func (chr *Ring) GetNode(hash []byte) *Node {
	chr.nodesMX.RLock()
	defer chr.nodesMX.RUnlock()

	if len(chr.hashes) == 0 {
		return nil
	}

	idx := chr.GetNextNodeIndex(Hash(hash))

	if idx == -1 {
		return nil
	}

	return chr.nodes[chr.hashes[idx]]
}
