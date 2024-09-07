package connection

import (
	"fmt"
	"sync"
	"testing"
)

func TestGossip(t *testing.T) {

	wg := sync.WaitGroup{}
	wg.Add(3)
	for i := 1; i <= 3; i++ {
		//generate config path
		//../resources/config_i.ini
		go func(i int) {
			configPath := fmt.Sprintf("../resources/config_%d.ini", i)
			println(configPath)
			peerServer, gossipServer := NewBothServer(configPath)
			StartServer(peerServer, gossipServer)
			defer CloseServer(peerServer, gossipServer)
			wg.Done()
		}(i)
	}
	wg.Wait()

}
