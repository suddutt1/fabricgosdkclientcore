package fabricgosdkclientcore_test

import (
	"fmt"
	"testing"
	"time"

	hlfsdkutil "github.com/suddutt1/fabricgosdkclientcore"
)

func Test_AdminEnroll(t *testing.T) {
	sdkClient := initializeIBPClient(t, "admin")
	defer sdkClient.Shutdown()
}
func Test_NormalUserEnroll(t *testing.T) {
	sdkClient := initializeIBPClient(t, "admin")
	defer sdkClient.Shutdown()
	if !sdkClient.EnrollOrgUser("suddutt3", "cnp4test", "org1") {
		t.Logf("Enrollment failed")
	}
}
func Test_InstallCC(t *testing.T) {
	sdkClient := initializeIBPClient(t, "admin")
	defer sdkClient.Shutdown()

	version := fmt.Sprintf("%d", time.Now().UnixNano())
	ccPath := "github.com/suddutt1/chaincode"
	goPath := "/home/ibmdev/go"
	ccPolicy := "And ('org1.member' )"
	ccID := fmt.Sprintf("CC_%s", version)
	rslt := sdkClient.InstallChainCode(ccID, "1.0", goPath, ccPath, nil)
	if !rslt {
		t.Logf("Chaincode installation failure ")
	}
	rslt, err := sdkClient.InstantiateCC("defaultchannel", ccID, ccPath, "1.0", [][]byte{[]byte("init")}, ccPolicy, nil)
	if err != nil || !rslt {
		t.Logf("Instantiation failed")
	}

}
func Test_InstallAndUpgradeCC(t *testing.T) {
	sdkClient := initializeIBPClient(t, "admin")
	defer sdkClient.Shutdown()

	version := fmt.Sprintf("%d", time.Now().UnixNano())
	ccPath := "github.com/suddutt1/chaincode"
	goPath := "/home/ibmdev/go"
	ccPolicy := "And ('org1.member' )"
	ccID := fmt.Sprintf("CC_%s", version)
	rslt := sdkClient.InstallChainCode(ccID, "1.0", goPath, ccPath, nil)
	if !rslt {
		t.Logf("Chaincode installation failure ")
	}
	rslt, err := sdkClient.InstantiateCC("defaultchannel", ccID, ccPath, "1.0", [][]byte{[]byte("init")}, ccPolicy, nil)
	if err != nil || !rslt {
		t.Logf("Instantiation failed")
	}
	time.Sleep(10 ^ time.Second)
	rslt = sdkClient.InstallChainCode(ccID, "2.0", goPath, ccPath, nil)
	if !rslt {
		t.Logf("Chaincode installation failure ")
	}
	rslt, err = sdkClient.UpdateCC("defaultchannel", ccID, ccPath, "2.0", [][]byte{[]byte("init")}, ccPolicy, nil)
	if err != nil || !rslt {
		t.Logf("Upgrade failed")
	}

}
func initializeIBPClient(t *testing.T, adminUID string) *hlfsdkutil.FabricSDKClient {
	fabricSDKClient := new(hlfsdkutil.FabricSDKClient)
	rslt := fabricSDKClient.Init("./ibp-client-config.json")
	if !rslt {
		t.Logf("Error in sdk initialization")
		t.FailNow()
	}
	if !fabricSDKClient.ErollOrgAdmin(false, adminUID) {
		t.Logf("Error in enrolling admin %s", adminUID)
	}
	return fabricSDKClient
}
