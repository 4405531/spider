package spider

import (
	"bytes"
	"net"

	"github.com/zeebo/bencode"
)

type krpc struct{ dht *dht }

func newKrpc(dht *dht) *krpc { return &krpc{dht: dht} }

func (p *krpc) decode(data []byte, val map[string]interface{}, raddr *net.UDPAddr) error {
	if err := bencode.DecodeBytes(data, &val); err != nil {
		return err
	}
	var ok bool
	message := new(krpcMessage)

	message.t, ok = val["t"].(string) //请求tid
	if !ok {
		return nil
	}

	message.y, ok = val["y"].(string) //请求类型
	if !ok {
		return nil
	}

	message.addr = raddr

	switch message.y {
	case "q":
		query := new(query)
		if q, ok := val["q"].(string); ok {
			query.y = q
		} else {
			return nil
		}
		if a, ok := val["a"].(map[string]interface{}); ok {
			query.a = a
			message.addion = query
		} else {
			return nil
		}
	case "r":
		res := new(response)
		if r, ok := val["r"].(map[string]interface{}); ok {
			res.r = r
			message.addion = res
		} else {
			return nil
		}
	default:
		return nil
	}

	switch message.y {
	case "q":
		p.query(message)
		break
	case "r":
		p.response(message)
		break
	}
	return nil
}

//Response message
func (p *krpc) response(msg *krpcMessage) {
	countFindResponse++
	//当table还有空余位置再解析，避免内存浪费
	if len(p.dht.table.nodes) <= TableMaxSize && len(finder) <= FinderMaxSize {
		if response, ok := msg.addion.(*response); ok {
			if nodestr, ok := response.r["nodes"].(string); ok {
				nodes := parseBytesStream([]byte(nodestr))
				for _, node := range nodes {
					if node.port > 0 && node.port <= (1<<16) {
						p.dht.table.put(node)
					}
				}
			}
		}
	}
}

//Infohash define data to storage
type Infohash struct {
	Infohash       string
	IP             net.IP
	Port           int
	ImpliedPort    int
	IsAnnouncePeer bool
}

func (p *krpc) query(msg *krpcMessage) {
	if query, ok := msg.addion.(*query); ok {
		if query.y == "get_peers" {
			countGetPeers++
			if infohash, ok := query.a["info_hash"].(string); ok {
				if len(infohash) != 20 || len(msg.t) == 0 {
					return
				}

				if fromID, ok := query.a["id"].(string); !ok {
					return
				} else if len(fromID) != 20 {
					return
				}

				p.dht.out <- Infohash{
					Infohash: ID(infohash).string(),
					IP:       msg.addr.IP}

				nodes := convertByteStream(p.dht.table.fnodes)
				data, _ := p.encodingNodeResult(msg.t, "asdf13e", nodes)
				go p.dht.network.send(data, msg.addr)
			}
		}
		if query.y == "announce_peer" {
			if infohash, ok := query.a["info_hash"].(string); ok {
				if token, ok := query.a["token"].(string); !ok {
					return
				} else if token != "asdf13e" {
					return
				}

				var port int
				var impliedPort int64
				impliedPort, ok = query.a["implied_port"].(int64)
				if ok {
					if impliedPort != 0 {
						port = msg.addr.Port
					}
				} else {
					pport, ok := query.a["port"].(int64)
					if ok {
						port = int(pport)
					}
				}

				countAnnounce++

				p.dht.out <- Infohash{
					Infohash:       ID(infohash).string(),
					IP:             msg.addr.IP,
					Port:           port,
					ImpliedPort:    int(impliedPort),
					IsAnnouncePeer: true}

				var data []byte
				if id, ok := query.a["id"].(string); ok {
					newID := neightor(id, p.dht.node.id.string())
					data, _ = p.encodingCommonResult(msg.t, newID)
				} else {
					data, _ = p.encodingCommonResult(msg.t, p.dht.node.id.string())
				}
				go p.dht.network.send(data, msg.addr)
			}
		}

		if query.y == "ping" {
			var data []byte
			if id, ok := query.a["id"].(string); ok && len(id) == 20 {
				newID := neightor(id, p.dht.node.id.string())
				data, _ = p.encodingCommonResult(msg.t, newID)
			} else {
				data, _ = p.encodingCommonResult(msg.t, p.dht.node.id.string())
			}
			countPing++
			go p.dht.network.send(data, msg.addr)
		}

		if query.y == "find_node" {
			if msg.t == "" {
				return
			}
			countFindNode++
			nodes := convertByteStream(p.dht.table.fnodes)
			data, _ := p.encodingNodeResult(msg.t, "", nodes)
			go p.dht.network.send([]byte(data), msg.addr)
		}
	}
}

func convertByteStream(nodes []*node) []byte {
	buf := bytes.NewBuffer(nil)
	for _, v := range nodes {
		convertNodeInfo(buf, v)
	}
	return buf.Bytes()
}

func convertNodeInfo(buf *bytes.Buffer, v *node) {
	buf.Write(v.id)
	convertIPPort(buf, v.ip, v.port)
}
func convertIPPort(buf *bytes.Buffer, ip net.IP, port int) {
	buf.Write(ip.To4())
	buf.WriteByte(byte((port & 0xFF00) >> 8))
	buf.WriteByte(byte(port & 0xFF))
}

func parseBytesStream(data []byte) []*node {
	var nodes []*node
	for j := 0; j < len(data); j = j + 26 {
		if j+26 > len(data) {
			break
		}
		kn := data[j : j+26]
		node := new(node)
		node.id = ID(kn[0:20])
		node.ip = kn[20:24]
		port := kn[24:26]
		node.port = int(port[0])<<8 + int(port[1])
		nodes = append(nodes, node)
	}
	return nodes
}

//KRPCMessage define
type krpcMessage struct {
	t      string
	y      string
	addion interface{}
	addr   *net.UDPAddr
}

type query struct {
	y string
	a map[string]interface{}
}

type response struct {
	r map[string]interface{}
}

func (p *krpc) encodingNodeResult(tid string, token string, nodes []byte) ([]byte, error) {
	v := make(map[string]interface{})
	defer func() { v = nil }()
	v["t"] = tid
	v["y"] = "r"
	args := make(map[string]string)
	defer func() { args = nil }()
	args["id"] = string(p.dht.node.id)
	if token != "" {
		args["token"] = token
	}
	args["nodes"] = bytes.NewBuffer(nodes).String()
	v["r"] = args
	return bencode.EncodeBytes(v)
}

func (p *krpc) encodingCommonResult(tid string, id string) ([]byte, error) {
	v := make(map[string]interface{})
	defer func() { v = nil }()
	v["t"] = tid
	v["y"] = "r"
	args := make(map[string]string)
	defer func() { args = nil }()
	args["id"] = id
	v["r"] = args
	return bencode.EncodeBytes(v)
}
