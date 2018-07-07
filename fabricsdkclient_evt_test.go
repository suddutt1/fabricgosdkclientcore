package fabricgosdkclientcore_test

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"testing"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
)

func Test_EventListener(t *testing.T) {
	osSigChan := make(chan os.Signal)
	signal.Notify(osSigChan, os.Interrupt, syscall.SIGTERM)

	clientsMap := initializeClients(t)
	defer cleanup(clientsMap)
	var wg sync.WaitGroup
	wg.Add(1)
	clientsMap["dist"].RegisterForBlockEvents("settlementchannel", "User1", nil, &wg, checkEvents)
	go func() {
		<-osSigChan
		fmt.Println("Ctrl-C detected..")
		clientsMap["dist"].DegisterBlockevent("settlementchannel", "User1")

	}()
	wg.Wait()

}
func checkEvents(eventChan <-chan *fab.BlockEvent, wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Println("Started listening....")
	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				fmt.Printf("unexpected closed channel while waiting for Tx Status event")
			}
			//fmt.Printf("Received block event: %+v\n", event)
			if event.Block == nil {
				fmt.Printf("Expecting block in block event but got nil")
			}
			fmt.Printf("Received block event: %+v\n", event.Block.Header.GetNumber())
		}
	}
}
