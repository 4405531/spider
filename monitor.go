package spider

import (
	"log"
	"os"
	"time"
)

var (
	countFindRequest  int64
	countFindResponse int64
	countFindNode     int64
	countPing         int64
	countAnnounce     int64
	countGetPeers     int64

	l = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)
)

//Monitor the network
func Monitor() {
	var preCountGetPeers int64
	for {
		if len(finder) >= FinderMaxSize {
			mutex.Lock()
			for k := range finder {
				delete(finder, k)
			}
			finder = nil
			finder = make(map[string]bool, FinderMaxSize)
			mutex.Unlock()
		}
		adjustFinderSpeed(countGetPeers - preCountGetPeers)
		preCountGetPeers = countGetPeers
		logger("发出find_node请求数量", countFindRequest)
		logger("收到find_node回复数量", countFindResponse)
		logger("收到find_node请求数量", countFindNode)
		logger("收到ping请求数量", countPing)
		logger("收到announce_peer请求数量", countAnnounce)
		logger("收到get_peers请求数量", preCountGetPeers)
		logger("-------------------------------")
		time.Sleep(time.Second * 60)
	}
}

func adjustFinderSpeed(count int64) {
	//阶梯调整频率
	//每分钟getpeer增加一万个，find sleep时间增加100ms
	delay := count / 10000
	if delay > 1 {
		FinderDelayTime = time.Duration(50*delay) * time.Millisecond
	}
}

func logger(v ...interface{}) {
	l.Println(v)
}
