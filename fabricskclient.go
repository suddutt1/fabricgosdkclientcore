package fabricgosdkclientcore

import (
	"fmt"
	"sync"
	"time"

	channel "github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	mspclient "github.com/hyperledger/fabric-sdk-go/pkg/client/msp"
	resourceMgmnt "github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"

	context "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
	core "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	msp "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/msp"
	sdkConfig "github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	packager "github.com/hyperledger/fabric-sdk-go/pkg/fab/ccpackager/gopackager"
	eventClient "github.com/hyperledger/fabric-sdk-go/pkg/fab/events/client"
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
	orgOrderer     string
	eventSubsReg   map[string]EventWaitGroup
	orgAdmin       string
	orgAdminSecret string
	orgMSPClient   *mspclient.Client
	remoteAdminID  string
	isRemoteAdmin  bool
}

//EventWaitGroup manages the event related wait groups
type EventWaitGroup struct {
	eventName    string
	wg           *sync.WaitGroup
	evtType      string
	eventService fab.EventService
	registration fab.Registration
}
type BlockEventListener func(<-chan *fab.BlockEvent, *sync.WaitGroup)
type CCEventListener func(<-chan *fab.CCEvent, *sync.WaitGroup)

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
	fsc.eventSubsReg = make(map[string]EventWaitGroup)
	configs, _ := fsc.configProvider()

	for _, cnfBackend := range configs {
		orgNameConfig, isFound := cnfBackend.Lookup("client.organization")
		if isFound {
			fsc.clientOrg, _ = orgNameConfig.(string)
			_logger.Infof("Client organization found the in the configuration %s", fsc.clientOrg)
		}
		//Orderer url and name
		orderersConfig, isFound := cnfBackend.Lookup("orderers")
		if isFound {
			orderersConfigMap, _ := orderersConfig.(map[string]interface{})
			for key := range orderersConfigMap {
				_logger.Infof("Orderer configuration for orderer %s is being retrieved", key)
				fsc.orgOrderer = key
				break
			}
		}
		//To load a channel clients user and channel namesa are. If the x-preloadedUsers list is
		//set in the configuration then they are loaded in init, else it is loaded.
		if usersConf, loadUsers := cnfBackend.Lookup("x-preloadedUsers"); loadUsers {
			users, _ := usersConf.([]interface{})

			for _, userid := range users {
				user, _ := userid.(string)
				if conf, isOk := cnfBackend.Lookup("channels"); isOk {
					channelDetailsMap, _ := conf.(map[string]interface{})
					for channelName := range channelDetailsMap {
						if _, isSetup := fsc.setupChannelClient(channelName, user); !isSetup {
							_logger.Errorf("Error in loading channels with given users")
							return false
						}
					}

				}
			}
		}
		if adminCert, loadAdminCert := cnfBackend.Lookup("x-remote-admin"); loadAdminCert {
			_logger.Infof("Loading pre generated admin cert")
			remoteAdmin, _ := adminCert.(string)
			fsc.remoteAdminID = remoteAdmin
			fsc.isRemoteAdmin = true
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
	channelContext, err := fsc.channelContextProviderMap[key]()
	if err != nil {
		_logger.Errorf("Error in creating channel cotext %+v", err)
		return nil, false
	}
	fsc.channelContextMap[key] = channelContext
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

	//si, _ := fsc.orgMSPClient.GetSigningIdentity(fsc.orgAdmin)

	adminContext := fsc.getAdminContext()

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
func (fsc *FabricSDKClient) InstantiateCC(channelName, ccID, ccPath, version string, initArgs [][]byte, ccPolicy string, wg *sync.WaitGroup) (bool, error) {
	if wg != nil {
		defer wg.Done()
	}
	adminContext := fsc.getAdminContext()

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
		resourceMgmnt.InstantiateCCRequest{Name: ccID, Path: ccPath, Version: version, Args: initArgs, Policy: policy})
	if err != nil {
		_logger.Errorf("Error in installation %+v", err)
		return false, err
	}
	_logger.Infof("Installation successful %+v", resp)
	return true, nil
}

//UpdateCC upgrades a chain code
func (fsc *FabricSDKClient) UpdateCC(channelName, ccID, ccPath, version string, initArgs [][]byte, ccPolicy string, wg *sync.WaitGroup) (bool, error) {
	if wg != nil {
		defer wg.Done()
	}
	adminContext := fsc.sdk.Context(fabsdk.WithUser(fsc.orgAdmin), fabsdk.WithOrg(fsc.clientOrg))

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
	// Org resource manager will upgrade
	resp, err := orgResrcMgmtClient.UpgradeCC(
		channelName,
		resourceMgmnt.UpgradeCCRequest{Name: ccID, Path: ccPath, Version: version, Args: initArgs, Policy: policy})
	if err != nil {
		_logger.Errorf("Error in upgrade %+v", err)
		return false, err
	}
	_logger.Infof("Installation upgrade %+v", resp)
	return true, nil
}

//SaveChannelInOrderer in Orderer. It sends the channelTx file to orderer
func (fsc *FabricSDKClient) SaveChannelInOrderer(channelID, pathToTxFile string, wg *sync.WaitGroup) bool {
	if wg != nil {
		defer wg.Done()
	}
	//First I need to save the channel the join with the others
	adminContext := fsc.sdk.Context(fabsdk.WithUser("Admin"), fabsdk.WithOrg("ordererorg"))

	// Org resource management client
	orgResrcMgmtClient, err := resourceMgmnt.New(adminContext)
	if err != nil {
		_logger.Errorf("Failed to create new resource management client: %+v", err)
		return false
	}
	mspClient, err := mspclient.New(fsc.sdk.Context(), mspclient.WithOrg(fsc.clientOrg))
	if err != nil {
		_logger.Errorf("Error in creating  msp client for org %s %+v", fsc.clientOrg, err)
		return false
	}
	adminIdentity, err := mspClient.GetSigningIdentity(fsc.orgAdmin)
	if err != nil {
		_logger.Errorf("Error in retriving the singing identity of the admin of org %s %+v", fsc.clientOrg, err)
		return false
	}
	_logger.Infof("Going to load tx file from %s", pathToTxFile)
	req := resourceMgmnt.SaveChannelRequest{ChannelID: channelID,
		ChannelConfigPath: pathToTxFile,
		SigningIdentities: []msp.SigningIdentity{adminIdentity}}
	saveChannelResp, err := orgResrcMgmtClient.SaveChannel(req, resourceMgmnt.WithRetry(retry.DefaultResMgmtOpts), resourceMgmnt.WithOrdererEndpoint(fsc.orgOrderer))
	if err != nil {
		_logger.Errorf("Error in savinf the channel for the org %s %+v", fsc.clientOrg, err)
		return false
	}
	_logger.Infof("Channel save of org %s is successful with trxnId %+v", fsc.clientOrg, saveChannelResp)
	return true
}

//JoinChannel should be called all the participanting peers of a given org. Should be called
//for each of the client SDK instances
func (fsc *FabricSDKClient) JoinChannel(channelID string, wg *sync.WaitGroup) bool {
	if wg != nil {
		defer wg.Done()
	}
	adminContext := fsc.sdk.Context(fabsdk.WithUser(fsc.orgAdmin), fabsdk.WithOrg(fsc.clientOrg))

	// Org resource management client
	orgResMgmtClient, err := resourceMgmnt.New(adminContext)
	if err != nil {
		_logger.Errorf("Failed to create new resource management client: %+v", err)
		return false
	}

	// Org peers join channel
	if err = orgResMgmtClient.JoinChannel(channelID, resourceMgmnt.WithRetry(retry.DefaultResMgmtOpts), resourceMgmnt.WithOrdererEndpoint(fsc.orgOrderer)); err != nil {
		_logger.Errorf("Org peers failed to JoinChannel for org  %s %+v", fsc.clientOrg, err)
		return false
	}
	_logger.Infof("Join channel with channelId %s for org %s is successful ", channelID, fsc.clientOrg)
	return true
}
func (fsc *FabricSDKClient) getAdminContext() context.ClientProvider {
	adminID := fsc.orgAdmin
	if fsc.isRemoteAdmin {
		adminID = fsc.remoteAdminID
	}
	adminContext := fsc.sdk.Context(fabsdk.WithUser(adminID), fabsdk.WithOrg(fsc.clientOrg))
	return adminContext
}
func (fsc *FabricSDKClient) addEventInRegistry(eventDetails EventWaitGroup) bool {
	if _, isOk := fsc.eventSubsReg[eventDetails.eventName]; isOk {
		_logger.Info("Event already registered %s", eventDetails.eventName)
		return false
	}
	fsc.eventSubsReg[eventDetails.eventName] = eventDetails
	return true
}

//RegisterForBlockEvents register for block events
func (fsc *FabricSDKClient) RegisterForBlockEvents(channelID string, userID string, wg, wgListenr *sync.WaitGroup, eventLister BlockEventListener) bool {
	if wg != nil {
		defer wg.Done()
	}
	if _, isFound := fsc.getChannelClient(channelID, userID); isFound {
		key := fmt.Sprintf("%s_%s", channelID, userID)
		eventService, err := fsc.channelContextMap[key].ChannelService().EventService(eventClient.WithBlockEvents())
		if err != nil {
			_logger.Errorf("Error getting event service: %+v", err)
			return false
		}
		var blockEventChan <-chan *fab.BlockEvent
		evtRegistration, blockEventChan, err := eventService.RegisterBlockEvent()
		if err != nil {
			_logger.Errorf("Error registering for block events: %+v", err)
			return false
		}
		eventName := fmt.Sprintf("%s_%s_BLOCKEVENT", channelID, userID)
		evntWg := EventWaitGroup{eventName: eventName, eventService: eventService, evtType: "BLOCK", registration: evtRegistration, wg: wgListenr}
		if !fsc.addEventInRegistry(evntWg) {
			_logger.Errorf("Event already registered and running .. Unregister the other listener")
			//Unregister right now
			eventService.Unregister(evtRegistration)
			return false
		}
		go eventLister(blockEventChan, wgListenr)
		return true
	}
	return false

}

//RegisterForCCEvent register for chain code event
func (fsc *FabricSDKClient) RegisterForCCEvent(channelID string, userID, ccID string, wg, wgListenr *sync.WaitGroup, eventLister CCEventListener) bool {
	if wg != nil {
		defer wg.Done()
	}
	if _, isFound := fsc.getChannelClient(channelID, userID); isFound {
		key := fmt.Sprintf("%s_%s", channelID, userID)
		eventService, err := fsc.channelContextMap[key].ChannelService().EventService(eventClient.WithBlockEvents())
		if err != nil {
			_logger.Errorf("Error getting event service: %+v", err)
			return false
		}
		var ccEventChan <-chan *fab.CCEvent
		evtRegistration, ccEventChan, err := eventService.RegisterChaincodeEvent(ccID, ".*")
		if err != nil {
			_logger.Errorf("Error registering for block events: %+v", err)
			return false
		}
		eventName := fmt.Sprintf("%s_%s_%s_CCEVENT", channelID, userID, ccID)
		evntWg := EventWaitGroup{eventName: eventName, eventService: eventService, evtType: "CCEVENT", registration: evtRegistration, wg: wgListenr}
		if !fsc.addEventInRegistry(evntWg) {
			_logger.Errorf("Event already registered and running .. Unregister the other listener")
			//Unregister right now
			eventService.Unregister(evtRegistration)
			return false
		}
		go eventLister(ccEventChan, wgListenr)
		return true
	}
	return false

}

//DegisterBlockevent dergisters a block event from the channel
func (fsc *FabricSDKClient) DegisterBlockevent(channelID, userID string) {
	if evtWtGrp, isFound := fsc.eventSubsReg[fmt.Sprintf("%s_%s_BLOCKEVENT", channelID, userID)]; isFound {
		evtWtGrp.Deregister()
	}
}

//DegisterCCevent deregister chain code event
func (fsc *FabricSDKClient) DegisterCCevent(channelID, userID, ccID string) {
	if evtWtGrp, isFound := fsc.eventSubsReg[fmt.Sprintf("%s_%s_%s_CCEVENT", channelID, userID, ccID)]; isFound {
		evtWtGrp.Deregister()
	}
}

//EnrollOrgUser reads the config and entrolls the registerer
func (fsc *FabricSDKClient) EnrollOrgUser(uid, secret, affiliationOrg string) bool {

	//First try to retrive the user
	err := fsc.orgMSPClient.Enroll(uid, mspclient.WithSecret(secret))
	if err == nil {
		_logger.Infof("User enrolled already : %s", uid)
		return true
	}
	//TODO: Study this following orgs once again
	userAttributes := []mspclient.Attribute{
		{
			Name:  "role1",
			Value: fmt.Sprintf("%s:ecert", "123"),
			ECert: true,
		},
		{
			Name:  "role2",
			Value: fmt.Sprintf("%s:ecert", "123"),
			ECert: true,
		},
	}
	//If user enrollment error then
	// Register the new user

	_, err = fsc.orgMSPClient.Register(&mspclient.RegistrationRequest{
		Name:           uid,
		Type:           "user",
		Attributes:     userAttributes,
		Affiliation:    affiliationOrg,
		MaxEnrollments: -1,
		Secret:         secret,
	})
	if err != nil {
		_logger.Fatalf("Registration failed: %s", err)
	}

	// Enroll the new user
	err = fsc.orgMSPClient.Enroll(uid, mspclient.WithSecret(secret))
	if err != nil {
		_logger.Fatalf("Enroll failed: %s", err)
	}

	// Get the new user's signing identity
	_, err = fsc.orgMSPClient.GetSigningIdentity(uid)
	if err != nil {
		_logger.Fatalf("GetSigningIdentity failed: %s", err)
	}
	return true
}

//ErollOrgAdmin will enroll the organization admin.
//if readFromConfig is true then it will be read from sdk config registerer
//entry. Else the userID given is used with the assumption that is it already pregenerated
func (fsc *FabricSDKClient) ErollOrgAdmin(readFromConfig bool, adminUID string) bool {
	ctxProvider := fsc.sdk.Context()
	mspClient, err := mspclient.New(ctxProvider)
	fsc.orgMSPClient = mspClient
	if !readFromConfig {
		_, err = fsc.orgMSPClient.GetSigningIdentity(adminUID)
		if err != nil {
			_logger.Fatalf("GetSigningIdentity failed: %s", err)
			return false
		}

		fsc.orgAdmin = adminUID
		_logger.Info("Enrolled registerer ", fsc.orgAdmin)
		return true
	}

	if err != nil {
		_logger.Errorf("Unable to create no-org MSPClient ")
		return false
	}

	ctx, err := ctxProvider()
	if err != nil {
		_logger.Fatalf("Failed to get context: %+v", err)
		return false
	}
	thisOrg := ctx.IdentityConfig().Client().Organization
	caConfig, ok := ctx.IdentityConfig().CAConfig(thisOrg)
	if !ok {
		_logger.Fatal("CAConfig failed")
		return false
	}

	err = mspClient.Enroll(caConfig.Registrar.EnrollID, mspclient.WithSecret(caConfig.Registrar.EnrollSecret))
	if err != nil {
		_logger.Fatalf("Registerer Enroll failed: %+v", err)
		return false
	}
	fsc.orgAdmin = caConfig.Registrar.EnrollID
	fsc.orgAdminSecret = caConfig.Registrar.EnrollSecret
	_logger.Info("Enrolled registerer ", fsc.orgAdmin)

	return true

}

//Deregister  de-registers evnt wait group
func (ewg *EventWaitGroup) Deregister() {
	ewg.eventService.Unregister(ewg.registration)
	if ewg.wg != nil {
		ewg.wg.Done()
	}
}

/*
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

//SleepFor util method ... Consider removing
func SleepFor(seconds time.Duration) {
	fmt.Println("Started waiting ....")
	select {
	case <-time.After(seconds * time.Second):
		fmt.Printf("timeout of %d seconds\n", seconds)
	}
	fmt.Println("Waiting ends")
}
