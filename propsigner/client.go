package propsigner

import (
	"bytes"
	"context"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/specs-actors/actors/builtin/market"
	"github.com/jsign/go-filsigner/wallet"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	pb "github.com/textileio/go-auctions-client/gen/wallet"
)

const (
	maxResponseMessageSize = 100 << 10
)

// RequestDealProposalSignatureV1 request a signature for a deal proposal to a remote wallet.
func RequestDealProposalSignatureV1(
	ctx context.Context,
	h host.Host,
	authToken string,
	proposal market.DealProposal,
	rwPeerID peer.ID) (*crypto.Signature, error) {
	proposalCborBytes := &bytes.Buffer{}
	if err := proposal.MarshalCBOR(proposalCborBytes); err != nil {
		return nil, fmt.Errorf("marshaling deal proposal to cbor: %s", err)
	}

	req := &pb.SigningRequest{
		AuthToken:            authToken,
		WalletAddress:        proposal.Client.String(),
		FilecoinDealProtocol: filDealProposalProtocolV1,
		Payload:              proposalCborBytes.Bytes(),
	}

	sig, err := sendToRemoteWallet(ctx, h, rwPeerID, req)
	if err != nil {
		return nil, fmt.Errorf("sending signing request to wallet: %s", err)
	}

	if err := ValidateDealProposalSignature(proposal, sig); err != nil {
		return nil, fmt.Errorf("validating signature: %s", err)
	}

	return sig, nil
}

// RequestDealStatusSignatureV1 request a signature for a deal status request to a remote wallet.
func RequestDealStatusSignatureV1(
	ctx context.Context,
	h host.Host,
	authToken string,
	walletAddr string,
	payload []byte,
	rwPeerID peer.ID) (*crypto.Signature, error) {
	req := &pb.SigningRequest{
		AuthToken:            authToken,
		WalletAddress:        walletAddr,
		FilecoinDealProtocol: filDealStatusProtocol,
		Payload:              payload,
	}

	sig, err := sendToRemoteWallet(ctx, h, rwPeerID, req)
	if err != nil {
		return nil, fmt.Errorf("sending signing request to wallet: %s", err)
	}

	return sig, nil
}

func sendToRemoteWallet(
	ctx context.Context,
	h host.Host,
	rwPeerID peer.ID,
	req *pb.SigningRequest) (*crypto.Signature, error) {
	s, err := h.NewStream(network.WithUseTransient(ctx, "relayed"), rwPeerID, v1Protocol)
	if err != nil {
		return nil, fmt.Errorf("creating libp2p stream: %s", err)
	}
	defer func() {
		if err := s.Close(); err != nil {
			log.Errorf("closing deal proposal signer stream: %s", err)
		}
	}()

	if err := writeMsg(s, req); err != nil {
		return nil, fmt.Errorf("sending deal signing request to stream: %s", err)
	}

	var res pb.SigningResponse
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

// ValidateDealProposalSignature validates that the signature is valid for the provided deal proposal.
func ValidateDealProposalSignature(proposal market.DealProposal, sig *crypto.Signature) error {
	msg := &bytes.Buffer{}
	err := proposal.MarshalCBOR(msg)
	if err != nil {
		return fmt.Errorf("marshaling proposal: %s", err)
	}
	sigBytes, err := sig.MarshalBinary()
	if err != nil {
		return fmt.Errorf("marshaling signature: %s", err)
	}
	ok, err := wallet.WalletVerify(proposal.Client, msg.Bytes(), sigBytes)
	if err != nil {
		return fmt.Errorf("verifying signature: %s", err)
	}
	if !ok {
		return fmt.Errorf("signature is invalid")
	}
	return nil
}

// ValidateDealStatusSignature validates that the signature is valid for the payload.
func ValidateDealStatusSignature(walletAddr string, payload []byte, sig *crypto.Signature) error {
	sigBytes, err := sig.MarshalBinary()
	if err != nil {
		return fmt.Errorf("marshaling signature: %s", err)
	}
	waddr, err := address.NewFromString(walletAddr)
	if err != nil {
		return fmt.Errorf("parsing wallet address: %s", err)
	}
	ok, err := wallet.WalletVerify(waddr, payload, sigBytes)
	if err != nil {
		return fmt.Errorf("verifying signature: %s", err)
	}
	if !ok {
		return fmt.Errorf("signature is invalid")
	}
	return nil
}
