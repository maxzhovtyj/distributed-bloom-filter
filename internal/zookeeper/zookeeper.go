package zookeeper

import (
	"github.com/spaolacci/murmur3"
	"slices"
	"sync"
)

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

func (chr *Ring) AddNode(id []byte, uri string) {
	chr.nodesMX.Lock()
	defer chr.nodesMX.Unlock()

	var err error

	hash := Hash(id)
	chr.nodes[hash], err = NewNode(id, uri)
	if err != nil {
		panic(err)
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

func (chr *Ring) GetNode(element []byte) *Node {
	chr.nodesMX.RLock()
	defer chr.nodesMX.RUnlock()

	if len(chr.hashes) == 0 {
		return nil
	}

	idx := chr.GetNextNodeIndex(Hash(element))

	if idx == -1 {
		return nil
	}

	return chr.nodes[chr.hashes[idx]]
}

func (chr *Ring) Close() {
	chr.nodesMX.Lock()
	defer chr.nodesMX.Unlock()

	for _, n := range chr.nodes {
		n.Close()
	}
}
