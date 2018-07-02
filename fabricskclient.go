package fabricgosdkclientcore

import (
	"fmt"
	"sync"
	"time"

	channel "github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	resourceMgmnt "github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	context "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
	core "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	sdkConfig "github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	packager "github.com/hyperledger/fabric-sdk-go/pkg/fab/ccpackager/gopackager"
	fabsdk "github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	cauthdsl "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/common/cauthdsl"
	logging "github.com/op/go-logging"
)

var _logger = logging.MustGetLogger("fabric-sdk-client")

//FabricSDKClient defines an easy to use wrapper to access a fabric blockchain environment.
//Each client is dedicated for a given organization. All the accesses are specific to a given
//organization.
type FabricSDKClient struct {
	sdk                       *fabsdk.FabricSDK
	channelContextProviderMap map[string]context.ChannelProvider
	channelContextMap         map[string]context.Channel
	channelClientMap          map[string]*channel.Client

	//orgResrcMgmtClient     *resourceMgmnt.Client
	configPath     string
	configProvider core.ConfigProvider
	clientOrg      string
}

//Shutdown Shutdown the client
func (fsc *FabricSDKClient) Shutdown() {
	defer fsc.sdk.Close()
}

//Init initializes the FabricSDK Client and its structure
func (fsc *FabricSDKClient) Init(configPath string) bool {
	var err error
	fsc.configPath = configPath
	fsc.configProvider = sdkConfig.FromFile(fsc.configPath)
	fsc.sdk, err = fabsdk.New(fsc.configProvider)
	if err != nil {
		_logger.Errorf("Error in initialization of SDK %+v", err)
		return false
	}
	if _logger.IsEnabledFor(logging.DEBUG) {
		_logger.Debugf("Configuration %+v\n", fsc.configProvider)
		_logger.Debugf("SDK %+v\n", fsc.sdk)
	}
	//Initialize maps
	fsc.channelContextProviderMap = make(map[string]context.ChannelProvider)
	fsc.channelContextMap = make(map[string]context.Channel)
	fsc.channelClientMap = make(map[string]*channel.Client)
	configs, _ := fsc.configProvider()
	for _, cnfBackend := range configs {
		orgNameConfig, isFound := cnfBackend.Lookup("client.organization")
		if isFound {
			fsc.clientOrg, _ = orgNameConfig.(string)
			_logger.Infof("Client organization found the in the configuration %s", fsc.clientOrg)
		}
		//To load a channel clients user and channel namesa are. If the x-preloadedUsers list is
		//set in the configuration then they are loaded in init, else it is loaded.
		if usersConf, loadUsers := cnfBackend.Lookup("x-preloadedUsers"); loadUsers {
			users, _ := usersConf.([]interface{})

			for _, userid := range users {
				user, _ := userid.(string)
				if conf, isOk := cnfBackend.Lookup("channels"); isOk {
					channelDetailsMap, _ := conf.(map[string]interface{})
					for channelName, _ := range channelDetailsMap {
						if _, isSetup := fsc.setupChannelClient(channelName, user); !isSetup {
							_logger.Errorf("Error in loading channels with given users")
							return false
						}
					}

				}
			}
		}

	}
	_logger.Info("Init complete")
	return true
}

//setupChannelClient setup channel clients for a given chanel and user
func (fsc *FabricSDKClient) setupChannelClient(channelName, user string) (*channel.Client, bool) {
	_logger.Debugf("Processing channel %s for user %s", channelName, user)
	key := fmt.Sprintf("%s_%s", channelName, user)
	fsc.channelContextProviderMap[key] = fsc.sdk.ChannelContext(channelName, fabsdk.WithUser(user), fabsdk.WithOrg(fsc.clientOrg))
	channelClient, err := channel.New(fsc.channelContextProviderMap[key])
	if err != nil {
		_logger.Errorf("Error in creating channel client %+v", err)
		return nil, false
	}
	fsc.channelClientMap[key] = channelClient
	return channelClient, true
}

//getChannelClient returns an existing channel client. If not setup , setup is done internally
func (fsc *FabricSDKClient) getChannelClient(channelName, user string) (*channel.Client, bool) {
	key := fmt.Sprintf("%s_%s", channelName, user)
	client, isExisting := fsc.channelClientMap[key]
	if !isExisting {
		_logger.Debugf("Not existing in the cnannel client map. Going to load %s", key)
		return fsc.setupChannelClient(channelName, user)
	}
	return client, isExisting
}

//Query method runs a query in the input channel. Returns the result , true/false and error object.
//2nd bool return equals to true means no problem in executing the query.
func (fsc *FabricSDKClient) Query(channelName, user, ccID, ccfuncName string, ccArgs [][]byte, targetPeers []string, wg *sync.WaitGroup) ([]byte, bool, error) {
	if wg != nil {
		defer wg.Done()
	}
	if channelClient, isFound := fsc.getChannelClient(channelName, user); isFound {
		response, err := channelClient.Query(channel.Request{ChaincodeID: ccID, Fcn: ccfuncName, Args: ccArgs}, channel.WithTargetEndpoints(targetPeers...))
		if err != nil {
			_logger.Errorf("Failed to query trxn: %+v\n", err)
			return nil, false, err
		}
		_logger.Debugf("Query response %s\n", string(response.Payload))
		return response.Payload, true, nil
	}
	return nil, false, fmt.Errorf("Channel cound not be found for %s channelname and user %s", channelName, user)

}

//InvokeTrxn invokes a transaction
func (fsc *FabricSDKClient) InvokeTrxn(channelName, user, ccID, ccfuncName string, ccArgs [][]byte, targetPeers []string, wg *sync.WaitGroup) ([]byte, bool, error) {
	if wg != nil {
		defer wg.Done()
	}

	if channelClient, isFound := fsc.getChannelClient(channelName, user); isFound {
		response, err := channelClient.Execute(channel.Request{ChaincodeID: ccID, Fcn: ccfuncName, Args: ccArgs}, channel.WithTargetEndpoints(targetPeers...))
		if err != nil {
			_logger.Errorf("Failed to execute trxn: %+v\n", err)
			return nil, false, err
		}

		_logger.Debugf("Execution response %s\n", string(response.Payload))
		if response.TxValidationCode == 0 {
			return response.Payload, true, nil
		}
		return response.Payload, false, fmt.Errorf("Transaction executed but not valid with reason code %d", response.TxValidationCode)
	}
	return nil, false, fmt.Errorf("Channel cound not be found for %s channelname and user %s", channelName, user)

}

//InstallChainCode Installs a chain code in the organization node.
//For each organization it has to be called separately as the admin credentials will not be available
//for a diffrent organization other this the client's organization
func (fsc *FabricSDKClient) InstallChainCode(ccID, version, goPath, ccPath string, wg *sync.WaitGroup) bool {
	if wg != nil {
		defer wg.Done()
	}
	ccPkg, err := packager.NewCCPackage(ccPath, goPath)
	if err != nil {
		_logger.Errorf("Packing error %+v\n", err)
		return false
	}
	adminContext := fsc.sdk.Context(fabsdk.WithUser("Admin"), fabsdk.WithOrg(fsc.clientOrg))

	// Org resource management client
	orgResrcMgmtClient, err := resourceMgmnt.New(adminContext)
	if err != nil {
		_logger.Errorf("Failed to create new resource management client: %+v", err)
		return false
	}
	// Install example cc to org peers
	installCCReq := resourceMgmnt.InstallCCRequest{Name: ccID, Path: ccPath, Version: version, Package: ccPkg}
	insResp, err := orgResrcMgmtClient.InstallCC(installCCReq)
	if err != nil {
		_logger.Errorf("Error in installing chain code  %+v", err)
		return false
	}
	_logger.Infof("Chain code installed %+v\n", insResp)
	return true

}

//InstantiateCC instantiates a chaincode
//As of now endorsement policy implemented is Any one of the participanting orgs
func (fsc *FabricSDKClient) InstantiateCC(channelName, ccId, ccPath, version string, initArgs [][]byte, ccPolicy string, wg *sync.WaitGroup) (bool, error) {
	if wg != nil {
		defer wg.Done()
	}
	adminContext := fsc.sdk.Context(fabsdk.WithUser("Admin"), fabsdk.WithOrg(fsc.clientOrg))

	// Org resource management client
	orgResrcMgmtClient, err := resourceMgmnt.New(adminContext)
	if err != nil {
		_logger.Errorf("Failed to create new resource management client: %+v", err)
		return false, err
	}
	policy, err := cauthdsl.FromString(ccPolicy)
	if err != nil {
		_logger.Errorf("Invalid chain code policy provided: %s error %+v", ccPolicy, err)
		return false, err
	}
	// Org resource manager will instantiate 'example_cc' on channel
	resp, err := orgResrcMgmtClient.InstantiateCC(
		channelName,
		resourceMgmnt.InstantiateCCRequest{Name: ccId, Path: ccPath, Version: version, Args: initArgs, Policy: policy})
	if err != nil {
		_logger.Errorf("Error in installation %+v", err)
		return false, err
	}
	_logger.Infof("Installation successful %+v", resp)
	return true, nil
}

/*
func ListenBlockEvents(channelContext context.Channel, wg *sync.WaitGroup) {
	defer wg.Done()
	eventService, err := channelContext.ChannelService().EventService(eventClient.WithBlockEvents())
	if err != nil {
		fmt.Printf("Error getting event service: %s\n", err)
		return
	}
	fmt.Println("Got the event service instance")
	var blockEventChan <-chan *fab.BlockEvent
	breg, blockEventChan, err := eventService.RegisterBlockEvent()
	if err != nil {
		fmt.Printf("Error registering for block events: %+v\n", err)
	}
	fmt.Println("Got the registering to Block Events")
	defer eventService.Unregister(breg)
	var subWg sync.WaitGroup

	if blockEventChan != nil {
		subWg.Add(1)
		go CheckEvents(blockEventChan, &subWg)

	}
	fmt.Println("Waiting for the block events to happen")
	subWg.Wait()
}
func CheckEvents(eventChan <-chan *fab.BlockEvent, wg *sync.WaitGroup) {
	defer wg.Done()
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
*/

func SleepFor(seconds time.Duration) {
	fmt.Println("Started waiting ....")
	select {
	case <-time.After(seconds * time.Second):
		fmt.Printf("timeout of %d seconds\n", seconds)
	}
	fmt.Println("Waiting ends")
}
