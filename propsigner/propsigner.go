package propsigner

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/specs-actors/actors/builtin/market"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	pb "github.com/textileio/go-auctions-client/gen/wallet"
	logger "github.com/textileio/go-log/v2"
)

const (
	v1Protocol            = "/auctions/proposal-signer/1.0.0"
	maxRequestMessageSize = 100 << 10 // 100KiB

	// filecoinDealProtocolV1 refers exactly to the libp2p protocol used to send deal proposals.
	// This value is used to know how to unmarshal proposal payloads. Future deal proposal versions
	// might be supported, and thus we need to be able to distinguish them to do proper unmarshaling.
	filecoinDealProtocolV1 = "/fil/storage/mk/1.1.0"
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
	h.SetStreamHandler(v1Protocol, dss.streamHandler)

	return nil
}

func (dss *dealSignerService) streamHandler(s network.Stream) {
	defer func() {
		if err := s.Close(); err != nil {
			log.Errorf("closing deal proposal signer stream: %s", err)
		}
	}()
	if err := s.SetDeadline(time.Now().Add(streamDeadline)); err != nil {
		log.Errorf("set deadline in stream: %s", err)
	}

	var req pb.ProposalSigningRequest
	if err := readMsg(s, maxRequestMessageSize, &req); err != nil {
		replyWithError(s, "unmarshaling proposal signing request: %s", err)
		return
	}
	if req.AuthToken != dss.authToken {
		replyWithError(s, errInvalidAuthToken.Error())
		return
	}

	var walletAddr string
	switch req.FilecoinDealProtocol {
	case filecoinDealProtocolV1:
		var proposal market.DealProposal
		if err := proposal.UnmarshalCBOR(bytes.NewReader(req.Payload)); err != nil {
			replyWithError(s, "unmarshaling proposal payload: %s", err)
			return
		}
		if err := dss.validateDealProposalV1(proposal); err != nil {
			replyWithError(s, "validating deal proposal: %s", err)
			return
		}
		walletAddr = proposal.Client.String()
	default:
		replyWithError(s, "unsupported filecoin deal proposal protocol")
		return
	}

	sig, err := dss.wallet.Sign(walletAddr, req.Payload)
	if err != nil {
		replyWithError(s, "signing proposal: %s", err)
		return
	}
	sigBytes, err := sig.MarshalBinary()
	if err != nil {
		replyWithError(s, "marshaling signature: %s", err)
		return
	}
	res := pb.ProposalSigningResponse{
		Signature: sigBytes,
	}
	if err := writeMsg(s, &res); err != nil {
		log.Errorf("writing error response to stream: %s", err)
		return
	}
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

	res := &pb.ProposalSigningResponse{
		Error: str,
	}
	if err := writeMsg(s, res); err != nil {
		log.Errorf("writing error response to stream: %s", err)
		return
	}
}
