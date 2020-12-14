package nv9

import (
	"context"

	amt3 "github.com/filecoin-project/go-amt-ipld/v3"
	hamt3 "github.com/filecoin-project/go-hamt-ipld/v3"
	power2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/power"
	cid "github.com/ipfs/go-cid"
	cbor "github.com/ipfs/go-ipld-cbor"

	builtin3 "github.com/filecoin-project/specs-actors/v3/actors/builtin"
	power3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/power"
	adt3 "github.com/filecoin-project/specs-actors/v3/actors/util/adt"
	smoothing3 "github.com/filecoin-project/specs-actors/v3/actors/util/smoothing"
)

type powerMigrator struct{}

func (m powerMigrator) MigrateState(ctx context.Context, store cbor.IpldStore, in StateMigrationInput) (*StateMigrationResult, error) {
	var inState power2.State
	if err := store.Get(ctx, in.head, &inState); err != nil {
		return nil, err
	}

	var proofValidationBatchOut *cid.Cid
	if inState.ProofValidationBatch != nil {
		proofValidationBatchAmtOpts := append(adt3.DefaultAmtOptions, amt3.UseTreeBitWidth(power3.ProofValidationBatchAmtBitwidth))
		proofValidationBatchOutCID, err := migrateHAMTAMTRaw(ctx, store, *inState.ProofValidationBatch, adt3.DefaultHamtOptionsWithDefaultBitwidth, proofValidationBatchAmtOpts)
		if err != nil {
			return nil, err
		}
		proofValidationBatchOut = &proofValidationBatchOutCID
	}

	claimsOut, err := migrateHAMTRaw(ctx, store, inState.Claims, adt3.DefaultHamtOptionsWithDefaultBitwidth)
	if err != nil {
		return nil, err
	}

	cronEventQueueHamtOpts := append(adt3.DefaultHamtOptions, hamt3.UseTreeBitWidth(power3.CronQueueHamtBitwidth))
	cronEventQueueAmtOpts := append(adt3.DefaultAmtOptions, amt3.UseTreeBitWidth(power3.CronQueueAmtBitwidth))
	cronEventQueueOut, err := migrateHAMTAMTRaw(ctx, store, inState.CronEventQueue, cronEventQueueHamtOpts, cronEventQueueAmtOpts)
	if err != nil {
		return nil, err
	}

	outState := power3.State{
		TotalRawBytePower:         inState.TotalRawBytePower,
		TotalBytesCommitted:       inState.TotalBytesCommitted,
		TotalQualityAdjPower:      inState.TotalQualityAdjPower,
		TotalQABytesCommitted:     inState.TotalQABytesCommitted,
		TotalPledgeCollateral:     inState.TotalPledgeCollateral,
		ThisEpochRawBytePower:     inState.ThisEpochRawBytePower,
		ThisEpochQualityAdjPower:  inState.ThisEpochQualityAdjPower,
		ThisEpochPledgeCollateral: inState.ThisEpochPledgeCollateral,
		ThisEpochQAPowerSmoothed:  smoothing3.FilterEstimate(inState.ThisEpochQAPowerSmoothed),
		MinerCount:                inState.MinerCount,
		MinerAboveMinPowerCount:   inState.MinerAboveMinPowerCount,
		CronEventQueue:            cronEventQueueOut,
		FirstCronEpoch:            inState.FirstCronEpoch,
		Claims:                    claimsOut,
		ProofValidationBatch:      proofValidationBatchOut,
	}
	newHead, err := store.Put(ctx, &outState)
	return &StateMigrationResult{
		NewCodeCID: builtin3.StoragePowerActorCodeID,
		NewHead:    newHead,
	}, err
}
