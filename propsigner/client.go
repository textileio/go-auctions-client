package propsigner

import (
	"bytes"
	"context"
	"fmt"

	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/specs-actors/actors/builtin/market"
	"github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-peer"
	pb "github.com/textileio/go-auctions-client/gen/wallet"
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
	proposalCborBytes := &bytes.Buffer{}
	if err := proposal.MarshalCBOR(proposalCborBytes); err != nil {
		return nil, fmt.Errorf("marshaling deal proposal to cbor: %s", err)
	}

	req := pb.ProposalSigningRequest{
		AuthToken:            authToken,
		FilecoinDealProtocol: filecoinDealProtocolV1,
		Payload:              proposalCborBytes.Bytes(),
	}

	s, err := h.NewStream(ctx, remoteWallet, v1Protocol)
	if err != nil {
		return nil, fmt.Errorf("creating libp2p stream: %s", err)
	}
	defer s.Close()

	if err := writeMsg(s, &req); err != nil {
		return nil, fmt.Errorf("sending deal signing request to stream: %s", err)
	}

	var res pb.ProposalSigningResponse
	if err := readMsg(s, maxResponseMessageSize, &res); err != nil {
		return nil, fmt.Errorf("unmarshaling proto deal signing response: %s", err)
	}

	if res.Error != "" {
		return nil, fmt.Errorf("response managed error: %s", res.Error)
	}

	var sig crypto.Signature
	if err := sig.UnmarshalBinary(res.Signature); err != nil {
		return nil, fmt.Errorf("unmarshaling signature cbor bytes: %s", err)
	}

	return &sig, nil
}
