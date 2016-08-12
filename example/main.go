package main

import (
	"fmt"

	"github.com/btlike/spider"
	_ "net/http/pprof"
)

var (
	hashChan         = make(chan spider.Infohash, 100)
	nodeNumber int64 = 10
)

func main() {
	idList := spider.GenerateIDList(nodeNumber)
	for k, id := range idList {
		go func(port int, id spider.ID) {
			spider.RunDhtNode(&id, hashChan, fmt.Sprintf(":%v", 20000+port))
		}(k, id)
	}

	go spider.Monitor()

	for {
		select {
		case hashID := <-hashChan:
			fmt.Println("magnet:?xt=urn:btih:" + hashID.Infohash)
		}
	}
}
