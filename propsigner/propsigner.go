package propsigner

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/filecoin-project/specs-actors/actors/builtin/market"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	pb "github.com/textileio/go-auctions-client/gen/wallet"
	logger "github.com/textileio/go-log/v2"
	"google.golang.org/protobuf/proto"
)

const (
	v1Protocol             = "/auctions/proposal-signer/1.0.0"
	maxProposalMessageSize = 100 << 10 // 100KiB

	// filecoinDealProtocolV1 refers exactly to the libp2p protocol used to send deal proposals.
	// This value is used to know how to unmarshal proposal payloads. Future deal proposal versions
	// might be supported, and thus we need to be able to distinguish them to do proper unmarshaling.
	filecoinDealProtocolV1 = "/fil/storage/mk/1.1.0"
)

var (
	log            = logger.Logger("propsigner")
	streamDeadline = time.Minute
)

type Wallet interface {
	Has(addr string) (bool, error)
	Sign(payload []byte) ([]byte, error)
}

type dealSignerService struct {
	authToken string
	wallet    Wallet
}

func NewDealSignerService(h host.Host, authToken string, wallet Wallet) error {
	dss := dealSignerService{
		authToken: authToken,
		wallet:    wallet,
	}
	h.SetStreamHandler(v1Protocol, dss.streamHandler)

	return nil
}

func (dss *dealSignerService) streamHandler(s network.Stream) {
	defer s.Reset()
	s.SetDeadline(time.Now().Add(streamDeadline))

	r := io.LimitReader(s, maxProposalMessageSize)
	buf, err := io.ReadAll(r)
	if err != nil {
		replyWithError(s, "reading proposal signing request: %s", err)
		return
	}

	var req pb.ProposalSigningRequest
	if err := proto.Unmarshal(buf, &req); err != nil {
		replyWithError(s, "unmarshaling proposal signing request: %s", err)
		return
	}
	if req.AuthToken != dss.authToken {
		replyWithError(s, "invalid auth token")
		return
	}

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
	default:
		replyWithError(s, "unsupported filecoin deal proposal protocol")
		return
	}

	sig, err := dss.wallet.Sign(req.Payload)
	if err != nil {
		replyWithError(s, "signing proposal: %s", err)
		return
	}
	res := pb.ProposalSigningResponse{
		Signature: sig,
	}
	buf, err = proto.Marshal(&res)
	if err != nil {
		replyWithError(s, "marshaling error response: %s", err)
		return
	}
	if _, err := s.Write(buf); err != nil {
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
		return fmt.Errorf("wallet doesn't have keys for %s", proposal.Client)
	}

	return nil
}

func replyWithError(s network.Stream, format string, params ...interface{}) {
	str := fmt.Sprintf(format, params...)
	log.Errorf(str)

	res := pb.ProposalSigningResponse{
		Error: str,
	}
	buf, err := proto.Marshal(&res)
	if err != nil {
		log.Errorf("marshaling error response: %s", err)
		return
	}
	if _, err := s.Write(buf); err != nil {
		log.Errorf("writing error response to stream: %s", err)
		return
	}
}
