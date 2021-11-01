package propsigner

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	cborutil "github.com/filecoin-project/go-cbor-util"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/specs-actors/actors/builtin/market"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	pb "github.com/textileio/go-auctions-client/gen/wallet"
	logger "github.com/textileio/go-log/v2"
)

const (
	v1Protocol = "/auctions/fil-signer/1.0.0"

	maxRequestMessageSize = 100 << 10 // 100KiB

	// filDealProposalProtocolV1 is the libp2p protocol used to send deal proposals in Filecoin.
	// This value is used to know how to unmarshal proposal payloads. Future deal proposal versions
	// might be supported, and thus we need to be able to distinguish them to do proper unmarshaling.
	filDealProposalProtocolV1 = "/fil/storage/mk/1.1.0"
	// filDealStatusProtocol is the libp2p protocol used to send deal status requests in Filecoin.
	filDealStatusProtocol = "/fil/storage/status/1.1.0"
)

var (
	log            = logger.Logger("propsigner")
	streamDeadline = time.Minute

	errInvalidAuthToken  = errors.New("invalid auth token")
	errWalletMissingKeys = errors.New("wallet doesn't have keys for address")
)

// Wallet contains private keys for Filecoin addresses.
type Wallet interface {
	Has(addr string) (bool, error)
	Sign(addr string, payload []byte) (*crypto.Signature, error)
}

type dealSignerService struct {
	authToken string
	wallet    Wallet
}

// NewDealSignerService configures a stream handler for the proposal signer protocol.
func NewDealSignerService(h host.Host, authToken string, wallet Wallet) error {
	if authToken == "" {
		return fmt.Errorf("authorization token is empty")
	}
	dss := dealSignerService{
		authToken: authToken,
		wallet:    wallet,
	}

	h.SetStreamHandler(v1Protocol, dss.streamDealProposalHandler)

	return nil
}

func (dss *dealSignerService) streamDealProposalHandler(s network.Stream) {
	log.Infof("handling signing request...")
	defer func() {
		if err := s.Close(); err != nil {
			log.Errorf("closing deal proposal signer stream: %s", err)
		}
	}()
	if err := s.SetDeadline(time.Now().Add(streamDeadline)); err != nil {
		log.Errorf("set deadline in stream: %s", err)
	}

	var req pb.SigningRequest
	if err := readMsg(s, maxRequestMessageSize, &req); err != nil {
		replyWithError(s, "unmarshaling proposal signing request: %s", err)
		return
	}
	if req.AuthToken != dss.authToken {
		replyWithError(s, errInvalidAuthToken.Error())
		return
	}

	var payloadToBeSigned []byte
	switch req.FilecoinDealProtocol {
	case filDealProposalProtocolV1:
		var proposal market.DealProposal
		if err := proposal.UnmarshalCBOR(bytes.NewReader(req.Payload)); err != nil {
			replyWithError(s, "unmarshaling proposal payload: %s", err)
			return
		}
		if err := dss.validateDealProposalV1(proposal); err != nil {
			replyWithError(s, "validating deal proposal: %s", err)
			return
		}
		log.Infof("signing deal proposal for storage-provider %s", proposal.Provider)
		payloadToBeSigned = req.Payload
	case filDealStatusProtocol:
		var proposalCid cid.Cid
		if err := proposalCid.UnmarshalBinary(req.Payload); err != nil {
			replyWithError(s, "unmarshaling proposal cid: %s", err)
		}

		log.Infof("signing deal status request for proposal %s", proposalCid)
		propCidCbor, err := cborutil.Dump(proposalCid)
		if err != nil {
			replyWithError(s, "marshaling proposal cid to cbor: %s", err)
			return
		}
		payloadToBeSigned = propCidCbor
	default:
		replyWithError(s, "unsupported filecoin deal proposal protocol")
		return
	}

	sig, err := dss.wallet.Sign(req.WalletAddress, payloadToBeSigned)
	if err != nil {
		replyWithError(s, "signing proposal: %s", err)
		return
	}
	sigBytes, err := sig.MarshalBinary()
	if err != nil {
		replyWithError(s, "marshaling signature: %s", err)
		return
	}
	res := pb.SigningResponse{
		Signature: sigBytes,
	}
	if err := writeMsg(s, &res); err != nil {
		log.Errorf("writing error response to stream: %s", err)
		return
	}
	log.Infof("request signed successfully")
}

func (dss *dealSignerService) validateDealProposalV1(proposal market.DealProposal) error {
	ok, err := dss.wallet.Has(proposal.Client.String())
	if err != nil {
		return fmt.Errorf("checking wallet keys: %s", err)
	}
	if !ok {
		return errWalletMissingKeys
	}

	return nil
}

func replyWithError(s network.Stream, format string, params ...interface{}) {
	str := fmt.Sprintf(format, params...)
	log.Errorf(str)

	res := &pb.SigningResponse{
		Error: str,
	}
	if err := writeMsg(s, res); err != nil {
		log.Errorf("writing error response to stream: %s", err)
		return
	}
}
