package propsigner

import (
	"context"
	"fmt"
	"io"

	cborutil "github.com/filecoin-project/go-cbor-util"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/specs-actors/actors/builtin/market"
	"github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-peer"
	pb "github.com/textileio/go-auctions-client/gen/wallet"
	"google.golang.org/protobuf/proto"
)

const (
	maxResponseMessageSize = 100 << 10
)

func RequestSignatureV1(
	ctx context.Context,
	h host.Host,
	authToken string,
	proposal market.DealProposal,
	remoteWallet peer.ID) (*crypto.Signature, error) {
	proposalCborBytes, err := cborutil.Dump(proposal)
	if err != nil {
		return nil, fmt.Errorf("marshaling deal proposal to cbor: %s", err)
	}

	req := pb.ProposalSigningRequest{
		AuthToken:            authToken,
		FilecoinDealProtocol: filecoinDealProtocolV1,
		Payload:              proposalCborBytes,
	}
	reqBytes, err := proto.Marshal(&req)
	if err != nil {
		return nil, fmt.Errorf("marshaling proto deal signing request: %s", err)
	}

	s, err := h.NewStream(ctx, remoteWallet, v1Protocol)
	if err != nil {
		return nil, fmt.Errorf("creating libp2p stream: %s", err)
	}
	if _, err := s.Write(reqBytes); err != nil {
		return nil, fmt.Errorf("sending deal signing request to stream: %s", err)
	}

	var res pb.ProposalSigningResponse
	r := io.LimitReader(s, maxResponseMessageSize)
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading response from stream: %s", err)
	}
	if err := proto.Unmarshal(buf, &res); err != nil {
		return nil, fmt.Errorf("unmarshaling proto deal signing response: %s", err)
	}

	var sig crypto.Signature
	if err := sig.UnmarshalBinary(res.Signature); err != nil {
		return nil, fmt.Errorf("unmarshaling signature cbor bytes: %s", err)
	}

	return &sig, nil
}
