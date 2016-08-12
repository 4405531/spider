package spider

import (
	"net"
	"sync"
)

//define size
const (
	TableMaxSize  = 1024
	FinderMaxSize = 100 * 1000
)

var (
	finder = make(map[string]bool, FinderMaxSize)
	mutex  sync.Mutex
)

type node struct {
	id   ID
	ip   net.IP
	port int
}

type table struct {
	nodes []*node
	cap   int64

	//用于响应find_node请求
	fnodes []*node
}

func (p *table) put(node *node) {
	ids := string([]byte(node.id))
	mutex.Lock()
	defer mutex.Unlock()
	if _, ok := finder[ids]; !ok {
		p.nodes = append(p.nodes, node)
		finder[ids] = true
	}
	if len(p.fnodes) < 8 {
		p.fnodes = append(p.fnodes, node)
	}
}

func (p *table) pop() *node {
	if len(p.nodes) > 0 {
		n := p.nodes[0]
		p.nodes = p.nodes[1:]
		return n
	}
	return nil
}
