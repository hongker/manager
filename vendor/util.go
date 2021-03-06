package vendor

import (
	"math/rand"
	"time"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/context/api/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/context/api/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/pkg/errors"

	cryptosuite "github.com/hyperledger/fabric-sdk-go/pkg/core/cryptosuite/bccsp/sw"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/identitymgr"
	kvs "github.com/hyperledger/fabric-sdk-go/pkg/fab/keyvaluestore"
	config "github.com/hyperledger/fabric-sdk-go/pkg/context/api/core"
)

const (
	adminUser      = "Admin"
	ordererOrgName = "ordererorg"
)

// GenerateRandomID generates random ID
func GenerateRandomID() string {
	return randomString(10)
}

// Utility to create random string of strlen length
func randomString(strlen int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

// InitializeChannel ...
func InitializeChannel(sdk *fabsdk.FabricSDK, orgID string, req resmgmt.SaveChannelRequest, targets []fab.ProposalProcessor) error {

	joinedTargets, err := FilterTargetsJoinedChannel(sdk, orgID, req.ChannelID, targets)
	if err != nil {
		return errors.WithMessage(err, "checking for joined targets failed")
	}

	if len(joinedTargets) != len(targets) {
		_, err := CreateChannel(sdk, req)
		if err != nil {
			return errors.Wrapf(err, "create channel failed")
		}

		_, err = JoinChannel(sdk, req.ChannelID, orgID)
		if err != nil {
			return errors.Wrapf(err, "join channel failed")
		}
	}
	return nil
}

// FilterTargetsJoinedChannel filters targets to those that have joined the named channel.
func FilterTargetsJoinedChannel(sdk *fabsdk.FabricSDK, orgID string, channelID string, targets []fab.ProposalProcessor) ([]fab.ProposalProcessor, error) {
	joinedTargets := []fab.ProposalProcessor{}

	//prepare context
	clientContext := sdk.Context(fabsdk.WithUser(adminUser), fabsdk.WithOrg(orgID))

	rc, err := resmgmt.New(clientContext)
	if err != nil {
		return nil, errors.WithMessage(err, "failed getting admin user session for org")
	}

	for _, target := range targets {
		// Check if primary peer has joined channel
		alreadyJoined, err := HasPeerJoinedChannel(rc, target, channelID)
		if err != nil {
			return nil, errors.WithMessage(err, "failed while checking if primary peer has already joined channel")
		}
		if alreadyJoined {
			joinedTargets = append(joinedTargets, target)
		}
	}
	return joinedTargets, nil
}

// CreateChannel attempts to save the named channel.
func CreateChannel(sdk *fabsdk.FabricSDK, req resmgmt.SaveChannelRequest) (bool, error) {

	//prepare context
	clientContext := sdk.Context(fabsdk.WithUser(adminUser), fabsdk.WithOrg(ordererOrgName))

	// Channel management client is responsible for managing channels (create/update)
	resMgmtClient, err := resmgmt.New(clientContext)
	if err != nil {
		return false, errors.WithMessage(err, "Failed to create new channel management client")
	}

	// Create channel (or update if it already exists)
	if err = resMgmtClient.SaveChannel(req); err != nil {
		return false, err
	}

	time.Sleep(time.Second * 5)
	return true, nil
}

// JoinChannel attempts to save the named channel.
func JoinChannel(sdk *fabsdk.FabricSDK, name, orgID string) (bool, error) {
	//prepare context
	clientContext := sdk.Context(fabsdk.WithUser(adminUser), fabsdk.WithOrg(orgID))

	// Resource management client is responsible for managing resources (joining channels, install/instantiate/upgrade chaincodes)
	resMgmtClient, err := resmgmt.New(clientContext)
	if err != nil {
		return false, errors.WithMessage(err, "Failed to create new resource management client")
	}

	if err = resMgmtClient.JoinChannel(name); err != nil {
		return false, nil
	}
	return true, nil
}

// CreateProposalProcessors initializes target peers based on config
func CreateProposalProcessors(config core.Config, orgs []string) ([]fab.Peer, error) {
	peers := []fab.Peer{}
	for _, org := range orgs {
		peerConfig, err := config.PeersConfig(org)
		if err != nil {
			return nil, errors.WithMessage(err, "reading peer config failed")
		}
		for _, p := range peerConfig {
			endorser, err := peer.New(config, peer.FromPeerConfig(&core.NetworkPeer{PeerConfig: p}))
			if err != nil {
				return nil, errors.WithMessage(err, "NewPeer failed")
			}
			peers = append(peers, endorser)
			if err != nil {
				return nil, errors.WithMessage(err, "adding peer failed")
			}
		}
	}
	return peers, nil
}

// HasPeerJoinedChannel checks whether the peer has already joined the channel.
// It returns true if it has, false otherwise, or an error
func HasPeerJoinedChannel(client *resmgmt.Client, peer fab.ProposalProcessor, channel string) (bool, error) {
	foundChannel := false
	response, err := client.QueryChannels(peer)
	if err != nil {
		return false, errors.WithMessage(err, "failed to query channel for peer")
	}
	for _, responseChannel := range response.Channels {
		if responseChannel.ChannelId == channel {
			foundChannel = true
		}
	}

	return foundChannel, nil
}

//ProposalProcessors utility to convert []fab.Peer to []fab.ProposalProcessor
func ProposalProcessors(targets []fab.Peer) []fab.ProposalProcessor {
	proposalProcessors := []fab.ProposalProcessor{}
	for _, peer := range targets {
		proposalProcessors = append(proposalProcessors, peer)
	}
	return proposalProcessors
}

// Enroll for fabric ca user
func Enroll(fabricConfig config.Config, username string, password string, orgName string) error {
	cryptoSuiteProvider, err := cryptosuite.GetSuiteByConfig(fabricConfig)
	if err != nil {
		return err
	}

	stateStore, err := kvs.New(&kvs.FileKeyValueStoreOptions{Path: fabricConfig.CredentialStorePath()})
	if err != nil {
		return err
	}

	caClient, err := identitymgr.New(orgName, stateStore, cryptoSuiteProvider, fabricConfig)
	if err != nil {
		return err
	}

	err = caClient.Enroll(username, password)
	if err != nil {
		return err
	}
	return nil
}
