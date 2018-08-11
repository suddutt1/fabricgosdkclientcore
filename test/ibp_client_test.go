package fabricgosdkclientcore_test

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	hlfsdkutil "github.com/suddutt1/fabricgosdkclientcore"
)

func Test_AdminEnrollNew(t *testing.T) {
	sdkClient := initializeIBPClient(t, "admin", true)
	defer sdkClient.Shutdown()
}
func Test_NormalUserEnroll(t *testing.T) {
	sdkClient := initializeIBPClient(t, "admin", false)
	defer sdkClient.Shutdown()
	if !sdkClient.EnrollOrgUser("suddutt6", "cnp4test", "org1") {
		t.Logf("Enrollment failed")
	}
}
func Test_InstallCC(t *testing.T) {
	sdkClient := initializeIBPClient(t, "admin", false)
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
	sdkClient := initializeIBPClient(t, "admin", false)
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
	time.Sleep(300 ^ time.Second)
	rslt = sdkClient.InstallChainCode(ccID, "2.0", goPath, ccPath, nil)
	if !rslt {
		t.Logf("Chaincode installation failure ")
	}
	rslt, err = sdkClient.UpdateCC("defaultchannel", ccID, ccPath, "2.0", [][]byte{[]byte("init")}, ccPolicy, nil)
	if err != nil || !rslt {
		t.Logf("Upgrade failed")
	}

}
func Test_InvokeTrxn_Query_Loop_IBP(t *testing.T) {
	sdkClient := initializeIBPClient(t, "admin", false)
	defer sdkClient.Shutdown()

	channelName := "defaultchannel"
	osSigChan := make(chan os.Signal)
	signal.Notify(osSigChan, os.Interrupt, syscall.SIGTERM)
	notStopped := false
	go func() {
		<-osSigChan
		fmt.Println("Ctrl-C detected..")
		notStopped = true
	}()
	for !notStopped {
		userID := "suddutt6"
		ccFn := "save"
		ccID := "CC_1534003045223505714"
		key := fmt.Sprintf("KEY_%d", time.Now().Nanosecond())
		//value := fmt.Sprintf("VALUE%d", time.Now().Nanosecond())

		//invokeArgs := [][]byte{[]byte(key), []byte(value)}
		peers := []string{"org1-peer1"}
		/*rsltBytes, isSuccess, err := sdkClient.InvokeTrxn(channelName, userID, ccID, ccFn, invokeArgs, peers, nil)
		if !isSuccess || err != nil {
			t.Logf("Error in Invoke Trxn %v", err)
			t.FailNow()
		}
		t.Logf("Result Invoke Trxn %s", string(rsltBytes))
		time.Sleep(15 * time.Second)
		//Need to query and verify
		*/
		queryArgs := [][]byte{[]byte(key)}
		ccFn = "probe"
		queryRsltBytes, isSuccess, err := sdkClient.Query(channelName, userID, ccID, ccFn, queryArgs, peers, nil)
		if !isSuccess || err != nil {
			t.Logf("Error in Query Trxn %v", err)
			t.FailNow()
		}
		t.Logf("Result Query Trxn %s", string(queryRsltBytes))

	}

}

func initializeIBPClient(t *testing.T, adminUID string, isNewEnrollment bool) *hlfsdkutil.FabricSDKClient {
	fabricSDKClient := new(hlfsdkutil.FabricSDKClient)
	rslt := fabricSDKClient.Init("./config/ibp-client-config.json")
	if !rslt {
		t.Logf("Error in sdk initialization")
		t.FailNow()
	}
	if !fabricSDKClient.ErollOrgAdmin(isNewEnrollment, adminUID) {
		t.Logf("Error in enrolling admin %s", adminUID)
	}
	return fabricSDKClient
}
