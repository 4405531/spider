package spider

import (
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/zeebo/bencode"
)

//define entry
var (
	Bootstrap       = []string{"router.bittorrent.com:6881", "router.utorrent.com:6881", "dht.transmissionbt.com:6881"}
	FinderDelayTime = time.Millisecond * 50
)

func (p *dht) findNode(v map[string]interface{}, args map[string]string, node *node) {
	countFindRequest++

	var id ID
	if node.id != nil {
		id = node.id.neighbor(p.node.id)
	} else {
		id = p.node.id
	}

	v["t"] = fmt.Sprintf("%d", rand.Intn(100))
	v["y"] = "q"
	v["q"] = "find_node"

	args["id"] = string(id)
	args["target"] = string(GenerateID())
	v["a"] = args
	data, err := bencode.EncodeBytes(v)
	if err != nil {
		logger(err)
		return
	}

	err = p.network.send(data, &net.UDPAddr{IP: node.ip, Port: node.port})
	if err != nil {
		logger(err)
		return
	}
}

func (p *dht) find() {
	if len(p.table.nodes) == 0 {
		val := make(map[string]interface{})
		args := make(map[string]string)
		for _, host := range Bootstrap {
			raddr, err := net.ResolveUDPAddr("udp", host)
			if err != nil {
				logger("Resolve DNS error, %s\n", err)
				return
			}
			node := new(node)
			node.port = raddr.Port
			node.ip = raddr.IP
			node.id = nil
			p.findNode(val, args, node)
		}
	}
	val := make(map[string]interface{})
	args := make(map[string]string)
	for {
		node := p.table.pop()
		if node != nil {
			p.findNode(val, args, node)
			time.Sleep(FinderDelayTime)
			continue
		}
		time.Sleep(time.Second)
	}
}
