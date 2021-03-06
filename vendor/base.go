package vendor

import (
	"os"
	"path"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/context"
	"github.com/hyperledger/fabric-sdk-go/pkg/context/api/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/context/api/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	packager "github.com/hyperledger/fabric-sdk-go/pkg/fab/ccpackager/gopackager"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/pkg/errors"
)

// BaseSetupImpl implementation of BaseTestSetup
type BaseSetupImpl struct {
	SDK           *fabsdk.FabricSDK
	Identity      context.Identity
	Targets       []fab.ProposalProcessor
	ConfigFile    string
	OrgID         string
	ChannelID     string
	ChannelConfig string
}

// Initial B values for ExampleCC
const (
	ExampleCCInitB    = "200"
	ExampleCCUpgradeB = "400"
	AdminUser         = "Admin"
	ConfigFile        = "config.yaml"
)

// ExampleCC query and transaction arguments
var queryArgs = [][]byte{[]byte("query"), []byte("b")}
var txArgs = [][]byte{[]byte("move"), []byte("a"), []byte("b"), []byte("1")}

// ExampleCC init and upgrade args
var initArgs = [][]byte{[]byte("init"), []byte("a"), []byte("100"), []byte("b"), []byte(ExampleCCInitB)}
var upgradeArgs = [][]byte{[]byte("init"), []byte("a"), []byte("100"), []byte("b"), []byte(ExampleCCUpgradeB)}

// ExampleCCQueryArgs returns example cc query args
func ExampleCCQueryArgs() [][]byte {
	return queryArgs
}

// ExampleCCTxArgs returns example cc move funds args
func ExampleCCTxArgs() [][]byte {
	return txArgs
}

//ExampleCCInitArgs returns example cc initialization args
func ExampleCCInitArgs() [][]byte {
	return initArgs
}

//ExampleCCUpgradeArgs returns example cc upgrade args
func ExampleCCUpgradeArgs() [][]byte {
	return upgradeArgs
}

// Initialize reads configuration from file and sets up client, channel and event hub
func (setup *BaseSetupImpl) Initialize() error {
	// Create SDK setup for the integration tests
	sdk, err := fabsdk.New(config.FromFile(setup.ConfigFile))
	if err != nil {
		return errors.WithMessage(err, "SDK init failed")
	}
	setup.SDK = sdk

	clientChannelContextProvider := sdk.ChannelContext(setup.ChannelID, fabsdk.WithUser(AdminUser), fabsdk.WithOrg(setup.OrgID))

	clientContext, err := clientChannelContextProvider()
	if err != nil {
		return errors.WithMessage(err, "failed to get client context")
	}
	setup.Identity = clientContext

	targets, err := getOrgTargets(sdk.Config(), setup.OrgID)
	if err != nil {
		return errors.Wrapf(err, "loading target peers from config failed")
	}
	setup.Targets = targets

	// Create channel for tests
	req := resmgmt.SaveChannelRequest{ChannelID: setup.ChannelID, ChannelConfig: setup.ChannelConfig, SigningIdentities: []context.Identity{clientContext}}
	if err = InitializeChannel(sdk, setup.OrgID, req, targets); err != nil {
		return errors.WithMessage(err, "failed to initialize channel")
	}

	return nil
}

func getOrgTargets(config core.Config, org string) ([]fab.ProposalProcessor, error) {
	targets := []fab.ProposalProcessor{}

	peerConfig, err := config.PeersConfig(org)
	if err != nil {
		return nil, errors.WithMessage(err, "reading peer config failed")
	}
	for _, p := range peerConfig {
		target, err := peer.New(config, peer.FromPeerConfig(&core.NetworkPeer{PeerConfig: p}))
		if err != nil {
			return nil, errors.WithMessage(err, "NewPeer failed")
		}
		targets = append(targets, target)
	}
	return targets, nil
}

// InitConfig ...
func (setup *BaseSetupImpl) InitConfig() core.ConfigProvider {
	return config.FromFile(setup.ConfigFile)
}

// GetDeployPath ..
func GetDeployPath() string {
	pwd, _ := os.Getwd()
	return path.Join(pwd, "../../fixtures/testdata")
}

// InstallAndInstantiateExampleCC install and instantiate using resource management client
func InstallAndInstantiateExampleCC(sdk *fabsdk.FabricSDK, user fabsdk.ContextOption, orgName string, chainCodeID string) error {
	return InstallAndInstantiateCC(sdk, user, orgName, chainCodeID, "github.com/example_cc", "v0", GetDeployPath(), initArgs)
}

// InstallAndInstantiateCC install and instantiate using resource management client
func InstallAndInstantiateCC(sdk *fabsdk.FabricSDK, user fabsdk.ContextOption, orgName string, ccName, ccPath, ccVersion, goPath string, ccArgs [][]byte) error {

	ccPkg, err := packager.NewCCPackage(ccPath, goPath)
	if err != nil {
		return errors.WithMessage(err, "creating chaincode package failed")
	}

	mspID, err := sdk.Config().MspID(orgName)
	if err != nil {
		return errors.WithMessage(err, "looking up MSP ID failed")
	}

	//prepare context
	clientContext := sdk.Context(user, fabsdk.WithOrg(orgName))

	// Resource management client is responsible for managing resources (joining channels, install/instantiate/upgrade chaincodes)
	resMgmtClient, err := resmgmt.New(clientContext)
	if err != nil {
		return errors.WithMessage(err, "Failed to create new resource management client")
	}

	_, err = resMgmtClient.InstallCC(resmgmt.InstallCCRequest{Name: ccName, Path: ccPath, Version: ccVersion, Package: ccPkg})
	if err != nil {
		return err
	}

	ccPolicy := cauthdsl.SignedByMspMember(mspID)
	return resMgmtClient.InstantiateCC("mychannel", resmgmt.InstantiateCCRequest{Name: ccName, Path: ccPath, Version: ccVersion, Args: ccArgs, Policy: ccPolicy})
}
