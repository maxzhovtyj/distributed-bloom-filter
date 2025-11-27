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

func (chr *Ring) AddNode(id []byte, uri string) *Node {
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
	return chr.nodes[hash]
}

func (chr *Ring) RemoveNode(id []byte) {
	chr.nodesMX.Lock()
	defer chr.nodesMX.Unlock()

	h := Hash(id)

	delete(chr.nodes, h)

	slices.DeleteFunc(chr.hashes, func(u uint32) bool {
		return u == h
	})
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

func (chr *Ring) GetNodeByHash(hash uint32) *Node {
	chr.nodesMX.RLock()
	defer chr.nodesMX.RUnlock()

	if len(chr.hashes) == 0 {
		return nil
	}

	for _, h := range chr.hashes {
		if h > hash {
			return chr.nodes[h]
		}
	}

	return nil
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

func (chr *Ring) CopyTo(dst *Ring) {
	chr.nodesMX.RLock()
	defer chr.nodesMX.RUnlock()

	for _, node := range chr.nodes {
		h := Hash(node.ID)
		dst.nodes[h] = node
		dst.hashes = append(dst.hashes, h)
	}

	slices.Sort(dst.hashes)
}

func (chr *Ring) Close() {
	chr.nodesMX.Lock()
	defer chr.nodesMX.Unlock()

	for _, n := range chr.nodes {
		n.Close()
	}
}
