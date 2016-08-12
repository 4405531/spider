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

	message.T, ok = val["t"].(string) //请求tid
	if !ok {
		return nil
	}

	message.Y, ok = val["y"].(string) //请求类型
	if !ok {
		return nil
	}

	message.Addr = raddr

	switch message.Y {
	case "q":
		query := new(query)
		if q, ok := val["q"].(string); ok {
			query.Y = q
		} else {
			return nil
		}
		if a, ok := val["a"].(map[string]interface{}); ok {
			query.A = a
			message.Addion = query
		} else {
			return nil
		}
	case "r":
		res := new(response)
		if r, ok := val["r"].(map[string]interface{}); ok {
			res.R = r
			message.Addion = res
		} else {
			return nil
		}
	default:
		return nil
	}

	switch message.Y {
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
		if response, ok := msg.Addion.(*response); ok {
			if nodestr, ok := response.R["nodes"].(string); ok {
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
	if query, ok := msg.Addion.(*query); ok {
		if query.Y == "get_peers" {
			countGetPeers++
			if infohash, ok := query.A["info_hash"].(string); ok {
				if len(infohash) != 20 || len(msg.T) == 0 {
					return
				}

				if fromID, ok := query.A["id"].(string); !ok {
					return
				} else if len(fromID) != 20 {
					return
				}

				p.dht.out <- Infohash{
					Infohash: ID(infohash).string(),
					IP:       msg.Addr.IP}

				nodes := convertByteStream(p.dht.table.fnodes)
				data, _ := p.encodingNodeResult(msg.T, "asdf13e", nodes)
				go p.dht.network.send(data, msg.Addr)
			}
		}
		if query.Y == "announce_peer" {
			if infohash, ok := query.A["info_hash"].(string); ok {
				if token, ok := query.A["token"].(string); !ok {
					return
				} else if token != "asdf13e" {
					return
				}

				var port int
				var impliedPort int64
				impliedPort, ok = query.A["implied_port"].(int64)
				if ok {
					if impliedPort != 0 {
						port = msg.Addr.Port
					}
				} else {
					pport, ok := query.A["port"].(int64)
					if ok {
						port = int(pport)
					}
				}

				countAnnounce++

				p.dht.out <- Infohash{
					Infohash:       ID(infohash).string(),
					IP:             msg.Addr.IP,
					Port:           port,
					ImpliedPort:    int(impliedPort),
					IsAnnouncePeer: true}

				var data []byte
				if id, ok := query.A["id"].(string); ok {
					newID := neightor(id, p.dht.node.id.string())
					data, _ = p.encodingCommonResult(msg.T, newID)
				} else {
					data, _ = p.encodingCommonResult(msg.T, p.dht.node.id.string())
				}
				go p.dht.network.send(data, msg.Addr)
			}
		}

		if query.Y == "ping" {
			var data []byte
			if id, ok := query.A["id"].(string); ok && len(id) == 20 {
				newID := neightor(id, p.dht.node.id.string())
				data, _ = p.encodingCommonResult(msg.T, newID)
			} else {
				data, _ = p.encodingCommonResult(msg.T, p.dht.node.id.string())
			}
			countPing++
			go p.dht.network.send(data, msg.Addr)
		}

		if query.Y == "find_node" {
			if msg.T == "" {
				return
			}
			countFindNode++
			nodes := convertByteStream(p.dht.table.fnodes)
			data, _ := p.encodingNodeResult(msg.T, "", nodes)
			go p.dht.network.send([]byte(data), msg.Addr)
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
	T      string
	Y      string
	Addion interface{}
	Addr   *net.UDPAddr
}

type query struct {
	Y string
	A map[string]interface{}
}

type response struct {
	R map[string]interface{}
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
