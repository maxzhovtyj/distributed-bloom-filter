package zookeeper

import (
	"github.com/spaolacci/murmur3"
	"slices"
	"sync"
)

type Ring struct {
	Nodes   map[uint32]*Node
	nodesMX sync.RWMutex

	Hashes []uint32
}

func NewRing() *Ring {
	return &Ring{
		Nodes:  make(map[uint32]*Node),
		Hashes: []uint32{},
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
	chr.Nodes[hash], err = NewNode(id, uri)
	if err != nil {
		panic(err)
	}

	chr.Hashes = append(chr.Hashes, hash)

	slices.Sort(chr.Hashes)

	return chr.Nodes[hash]
}

func (chr *Ring) RemoveNode(id []byte) {
	chr.nodesMX.Lock()
	defer chr.nodesMX.Unlock()

	h := Hash(id)

	delete(chr.Nodes, h)

	slices.DeleteFunc(chr.Hashes, func(u uint32) bool {
		return u == h
	})
}

func (chr *Ring) GetNextNodeIndex(hash uint32) int {
	if len(chr.Hashes) == 0 {
		return -1
	}

	for i, h := range chr.Hashes {
		if h > hash {
			return i
		}
	}

	return 0
}

func (chr *Ring) GetNodeByHash(hash uint32) *Node {
	chr.nodesMX.RLock()
	defer chr.nodesMX.RUnlock()

	if len(chr.Hashes) == 0 {
		return nil
	}

	for _, h := range chr.Hashes {
		if h > hash {
			return chr.Nodes[h]
		}
	}

	return nil
}

func (chr *Ring) GetNode(element []byte) *Node {
	chr.nodesMX.RLock()
	defer chr.nodesMX.RUnlock()

	if len(chr.Hashes) == 0 {
		return nil
	}

	idx := chr.GetNextNodeIndex(Hash(element))

	if idx == -1 {
		return nil
	}

	return chr.Nodes[chr.Hashes[idx]]
}

func (chr *Ring) CopyTo(dst *Ring) {
	chr.nodesMX.RLock()
	defer chr.nodesMX.RUnlock()

	for _, node := range chr.Nodes {
		h := Hash(node.ID)
		dst.Nodes[h] = node
		dst.Hashes = append(dst.Hashes, h)
	}

	slices.Sort(dst.Hashes)
}

func (chr *Ring) Close() {
	chr.nodesMX.Lock()
	defer chr.nodesMX.Unlock()

	for _, n := range chr.Nodes {
		n.Close()
	}
}
