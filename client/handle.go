package client

import (
	"context"
	"fmt"
	"log"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/client"
	asset2 "perun.network/perun-demo-tui/asset"
)

// HandleProposal is the callback for incoming channel proposals.
func (p *PaymentClient) HandleProposal(prop client.ChannelProposal, r *client.ProposalResponder) {
	lcp, tuiAssets, err := func() (*client.LedgerChannelProposalMsg, []asset2.TUIAsset, error) {
		// Ensure that we got a ledger channel proposal.
		lcp, ok := prop.(*client.LedgerChannelProposalMsg)
		if !ok {
			return nil, nil, fmt.Errorf("invalid proposal type: %T", p)
		}

		// Check that we have the correct number of participants.
		if lcp.NumPeers() != 2 {
			return nil, nil, fmt.Errorf("invalid number of participants: %d", lcp.NumPeers())
		}
		// Check that the channel has the expected assets and funding balances.
		tuiAssets := make([]asset2.TUIAsset, len(lcp.FundingAgreement))
		for i, assetAlloc := range lcp.FundingAgreement {
			if assetAlloc[0].Cmp(assetAlloc[1]) != 0 {
				return nil, nil, fmt.Errorf("invalid funding balance for asset %d: %v", i, assetAlloc)
			}
			tuiAsset, err := GetTuiAsset(p.availableAssets, lcp.InitBals.Assets[i])
			if err != nil {
				return nil, nil, err
			}
			tuiAssets[i] = tuiAsset
		}
		return lcp, tuiAssets, nil
	}()
	if err != nil {
		_ = r.Reject(context.TODO(), err.Error())
	}

	// Create a channel accept message and send it.
	accept := lcp.Accept(
		p.WalletAddress(),        // The Account we use in the channel.
		client.WithRandomNonce(), // Our share of the channel nonce.
	)
	ch, err := r.Accept(context.TODO(), accept)
	if err != nil {
		log.Printf("Error accepting channel proposal: %v", err)
		return
	}

	//TODO: startWatching
	// Start the on-chain event watcher. It automatically handles disputes.
	p.startWatching(ch)

	// Store channel.
	p.channels <- newPaymentChannel(ch, tuiAssets)
	p.AcceptedChannel()
}

// HandleUpdate is the callback for incoming channel updates.
func (p *PaymentClient) HandleUpdate(cur *channel.State, next client.ChannelUpdate, r *client.UpdateResponder) {
	// We accept every update that increases our balance.
	err := func() error {
		err := channel.AssertAssetsEqual(cur.Assets, next.State.Assets)
		if err != nil {
			return fmt.Errorf("Invalid assets: %v", err)
		}

		receiverIdx := 1 - next.ActorIdx // This works because we are in a two-party channel.
		for _, a := range cur.Assets {
			curBal := cur.Allocation.Balance(receiverIdx, a)
			nextBal := next.State.Allocation.Balance(receiverIdx, a)
			if nextBal.Cmp(curBal) < 0 {
				return fmt.Errorf("invalid balance: %v", nextBal)
			}
		}

		return nil
	}()
	if err != nil {
		_ = r.Reject(context.TODO(), err.Error())
	}

	// Send the acceptance message.
	err = r.Accept(context.TODO())
	if err != nil {
		panic(err)
	}
}

// HandleAdjudicatorEvent is the callback for smart contract events.
func (p *PaymentClient) HandleAdjudicatorEvent(e channel.AdjudicatorEvent) {
	log.Printf("Adjudicator event: type = %T, client = %v", e, p.Account)
}
