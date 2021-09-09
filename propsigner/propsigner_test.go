package propsigner

import (
	"bytes"
	"context"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/specs-actors/actors/builtin/market"
	"github.com/ipfs/go-cid"
	libwal "github.com/jsign/go-filsigner/wallet"
	"github.com/libp2p/go-libp2p-core/peer"
	swarmt "github.com/libp2p/go-libp2p-swarm/testing"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	"github.com/stretchr/testify/require"
	"github.com/textileio/go-auctions-client/localwallet"
)

var (
	walletKeys = []string{
		// Secp256k1 exported private key in Lotus format.
		"7b2254797065223a22736563703235366b31222c22507269766174654b6579223a226b35507976337148327349586343595a58594f5775453149326e32554539436861556b6c4e36695a5763453d227d", // nolint:lll
		// BLS exported private key in Lotus format.
		"7b2254797065223a22626c73222c22507269766174654b6579223a226862702f794666527439514c43716b6d566171415752436f50556777314b776971716e73684e49704e57513d227d", // nolint:lll
	}
)

type testCase struct {
	name      string
	proposal  market.DealProposal
	authToken string
	err       error
}

func TestProposalSigning(t *testing.T) {
	t.Parallel()

	authToken := "veryhardtokentoguess"
	wallet, err := localwallet.New(walletKeys)
	require.NoError(t, err)

	testCases := []testCase{
		{
			name:      "success secp256k1",
			proposal:  correctProposalSecp256k1(t),
			authToken: authToken,
			err:       nil,
		},
		{
			name:      "success bls",
			proposal:  correctProposalBLS(t),
			authToken: authToken,
			err:       nil,
		},
		{
			name:      "invalid auth token",
			proposal:  correctProposalSecp256k1(t),
			authToken: "wrongToken",
			err:       errInvalidAuthToken,
		},
		{
			name:      "invalid auth token",
			proposal:  proposalWithUnknownAddress(t),
			authToken: authToken,
			err:       errWalletMissingKeys,
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Remote wallet libp2p host.
			h1 := bhost.New(swarmt.GenSwarm(t, ctx))
			err = NewDealSignerService(h1, authToken, wallet)
			require.NoError(t, err)

			// Client (dealerd) libp2p2 host.
			h2 := bhost.New(swarmt.GenSwarm(t, ctx))
			err := h2.Connect(ctx, peer.AddrInfo{ID: h1.ID(), Addrs: h1.Addrs()})
			require.NoError(t, err)

			sig, err := RequestSignatureV1(ctx, h2, test.authToken, test.proposal, h1.ID())
			if test.err != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.err.Error())
			} else {
				require.NoError(t, err)
				requireValidSignature(t, test.proposal, sig)
			}
		})
	}
}

func requireValidSignature(t *testing.T, proposal market.DealProposal, sig *crypto.Signature) {
	msg := &bytes.Buffer{}
	err := proposal.MarshalCBOR(msg)
	require.NoError(t, err)
	sigBytes, err := sig.MarshalBinary()
	require.NoError(t, err)
	ok, err := libwal.WalletVerify(proposal.Client, msg.Bytes(), sigBytes)
	require.NoError(t, err)
	require.True(t, ok)
}

func correctProposalSecp256k1(t *testing.T) market.DealProposal {
	secpAddr, err := libwal.PublicKey(walletKeys[0])
	require.NoError(t, err)

	minerAddr, err := address.NewFromString("f0001")
	require.NoError(t, err)

	return market.DealProposal{
		PieceCID:     castCid("QmWc1T3ZMtAemjdt7Z87JmFVGjtxe4S6sNwn9zhvcNP1Fs"),
		PieceSize:    1000,
		VerifiedDeal: true,
		Client:       secpAddr,
		Provider:     minerAddr,

		Label: "this is a fake label",

		StartEpoch:           100,
		EndEpoch:             200,
		StoragePricePerEpoch: big.NewInt(3000),
	}
}

func correctProposalBLS(t *testing.T) market.DealProposal {
	blsAddr, err := libwal.PublicKey(walletKeys[1])
	require.NoError(t, err)

	proposal := correctProposalSecp256k1(t)
	proposal.Client = blsAddr

	return proposal
}

func proposalWithUnknownAddress(t *testing.T) market.DealProposal {
	fakeSecp256k1Addr := "f3wmv7nhiqosmlr6mis2mr4xzupdhe3rtvw5ntis4x6yru7jhm35pfla2pkwgwfa3t62kdmoylssczmf74yika"
	unknownSecp256k1Addr, err := address.NewFromString(fakeSecp256k1Addr)
	require.NoError(t, err)

	proposal := correctProposalSecp256k1(t)
	proposal.Client = unknownSecp256k1Addr

	return proposal
}

func castCid(cidStr string) cid.Cid {
	c, _ := cid.Decode(cidStr)
	return c
}
