package zookeeper

import (
	"fmt"
	"strings"

	//"github.com/spaolacci/murmur3"
	"github.com/twmb/murmur3"
	"slices"
	"sync"
)

type Ring struct {
	Nodes   map[uint64]*Node
	nodesMX sync.RWMutex

	Hashes []uint64
}

func NewRing() *Ring {
	return &Ring{
		Nodes:  make(map[uint64]*Node),
		Hashes: []uint64{},
	}
}

func (chr *Ring) String() string {
	chr.nodesMX.RLock()
	defer chr.nodesMX.RUnlock()

	var buf strings.Builder

	for i, h := range chr.Hashes {
		buf.WriteString(fmt.Sprintf("#%d — %d — node id %s\n", i, h, chr.Nodes[h].ID))
	}

	return buf.String()
}

func Hash(key []byte) uint64 {
	return murmur3.Sum64(key)
}

func (chr *Ring) AddNode(opt NodeOption) (*Node, error) {
	chr.nodesMX.Lock()
	defer chr.nodesMX.Unlock()

	hash := Hash([]byte(opt.ID))

	n := NewNode(opt)
	n.Hash = hash

	err := n.Init()
	if err != nil {
		return nil, err
	}

	chr.Nodes[hash] = n
	chr.Hashes = append(chr.Hashes, hash)
	slices.Sort(chr.Hashes)

	if opt.VMNodes > 0 {
		chr.initVirtualNodes(opt, n)
	}

	return chr.Nodes[hash], nil
}

func (chr *Ring) initVirtualNodes(opt NodeOption, n *Node) {
	for i := range opt.VMNodes {
		vmNode := new(Node)

		n.CopyTo(vmNode)

		vmNode.ID = []byte(fmt.Sprintf("%s_%d", opt.ID, i))
		vmNode.Hash = Hash(vmNode.ID)
		vmNode.IsVM = true
		vmNode.PhysicalNodeID = append(vmNode.PhysicalNodeID, n.ID...)

		n.VMNodes = append(n.VMNodes, VMNode{
			ID:   append([]byte{}, vmNode.ID...),
			Hash: vmNode.Hash,
		})

		chr.Nodes[vmNode.Hash] = vmNode
		chr.Hashes = append(chr.Hashes, vmNode.Hash)

		slices.Sort(chr.Hashes)
	}
}

func (chr *Ring) RemoveNode(id []byte) {
	chr.nodesMX.Lock()
	defer chr.nodesMX.Unlock()

	h := Hash(id)

	delete(chr.Nodes, h)

	slices.DeleteFunc(chr.Hashes, func(u uint64) bool {
		return u == h
	})
}

func (chr *Ring) GetNextNodeIndex(hash uint64) int {
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

func (chr *Ring) GetNodeByHash(hash uint64) *Node {
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
