package fabricgosdkclientcore_test

import (
	"testing"

	hlfsdkutil "github.com/suddutt1/fabricgosdkclientcore"
)

func Test_FabricSDKClient_Init(t *testing.T) {

	clientsMap := initializeClients(t)
	cleanup(clientsMap)

}
func initializeClients(t *testing.T) map[string]*hlfsdkutil.FabricSDKClient {
	fabricNetworkClientManuf := new(hlfsdkutil.FabricSDKClient)
	rslt := fabricNetworkClientManuf.Init("./manuf-client-config.yaml")
	if !rslt {
		t.Logf("Error in sdk initialization manufacturer")
		t.FailNow()
	}
	fabricNetworkClientRetail := new(hlfsdkutil.FabricSDKClient)
	if !fabricNetworkClientRetail.Init("./retailer-client-config.yaml") {
		t.Logf("Error in sdk initialization retailer")
		t.FailNow()
	}
	fabricNetworkClientDist := new(hlfsdkutil.FabricSDKClient)
	if !fabricNetworkClientDist.Init("./dist-client-config.yaml") {
		t.Logf("Error in sdk initialization distributer")
		t.FailNow()
	}
	clientsMap := make(map[string]*hlfsdkutil.FabricSDKClient)
	clientsMap["retail"] = fabricNetworkClientRetail
	clientsMap["dist"] = fabricNetworkClientDist
	clientsMap["manuf"] = fabricNetworkClientManuf
	return clientsMap
}
func cleanup(clientMap map[string]*hlfsdkutil.FabricSDKClient) {
	clientMap["retail"].Shutdown()
	clientMap["dist"].Shutdown()
	clientMap["manuf"].Shutdown()

}
