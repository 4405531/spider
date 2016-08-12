package spider

import (
	"net"
	"os"
	"time"

	"github.com/juju/ratelimit"
)

//RateLimit limit speed
var RateLimit int64 = 100

type network struct {
	dht       *dht
	conn      *net.UDPConn
	rateLimit *ratelimit.Bucket
}

func newNetwork(dht *dht, address string) *network {
	nw := &network{dht: dht,
		//默认限速：每个节点每秒最多处理100个请求
		rateLimit: ratelimit.NewBucketWithRate(float64(RateLimit), RateLimit)}
	nw.init(address)
	return nw
}

func (p *network) init(address string) {
	addr := new(net.UDPAddr)
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		logger(err)
		os.Exit(1)
	}
	p.conn, err = net.ListenUDP("udp", addr)
	if err != nil {
		logger(err)
		os.Exit(1)
	}

	laddr := p.conn.LocalAddr().(*net.UDPAddr)
	p.dht.node.ip = laddr.IP
	p.dht.node.port = laddr.Port
}

func (p *network) listen() {
	val := make(map[string]interface{})
	buf := make([]byte, 1024)
	for {
		time.Sleep(p.rateLimit.Take(1))
		n, raddr, err := p.conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		p.dht.krpc.decode(buf[:n], val, raddr)
	}
}

func (p *network) send(m []byte, addr *net.UDPAddr) error {
	if addr != nil && addr.Port != 0 {
		_, err := p.conn.WriteToUDP(m, addr)
		if err != nil {
			logger(err)
		}
		return err
	}
	return nil
}
