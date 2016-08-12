package spider

import (
	"fmt"
)

type dht struct {
	node    *node
	table   *table
	network *network
	krpc    *krpc
	out     chan Infohash
}

//RunDhtNode run a  dht node
func RunDhtNode(id *ID, out chan Infohash, address string) {
	dht := &dht{
		node:  &node{id: *id},
		out:   out,
		table: &table{}}
	dht.network = newNetwork(dht, address)
	dht.krpc = newKrpc(dht)
	go func() { dht.network.listen() }()
	go func() { dht.find() }()
	logger(fmt.Sprintf("爬虫节点开始运行,最大处理速度每秒:%d,监听地址:%s", RateLimit, dht.network.conn.LocalAddr().String()))
}
