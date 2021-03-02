package builtin

import (
	stabi "github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/pkg/errors"
)

// Policy values associated with a seal proof type.
type SealProofPolicy struct {
	SectorMaxLifetime      stabi.ChainEpoch
	ConsensusMinerMinPower stabi.StoragePower
}

// For all Stacked DRG sectors, the max is 5 years
const EpochsInFiveYears = stabi.ChainEpoch(5 * EpochsInYear)

var SealProofPolicies = map[stabi.RegisteredSealProof]*SealProofPolicy{
	stabi.RegisteredSealProof_StackedDrg2KiBV1: {
		SectorMaxLifetime:      EpochsInFiveYears,
		ConsensusMinerMinPower: stabi.NewStoragePower(0),
	},
	stabi.RegisteredSealProof_StackedDrg8MiBV1: {
		SectorMaxLifetime:      EpochsInFiveYears,
		ConsensusMinerMinPower: stabi.NewStoragePower(16 << 20),
	},
	stabi.RegisteredSealProof_StackedDrg512MiBV1: {
		SectorMaxLifetime:      EpochsInFiveYears,
		ConsensusMinerMinPower: stabi.NewStoragePower(1 << 30),
	},
	stabi.RegisteredSealProof_StackedDrg32GiBV1: {
		SectorMaxLifetime:      EpochsInFiveYears,
		ConsensusMinerMinPower: stabi.NewStoragePower(10 << 40),
	},
	stabi.RegisteredSealProof_StackedDrg64GiBV1: {

		SectorMaxLifetime:      EpochsInFiveYears,
		ConsensusMinerMinPower: stabi.NewStoragePower(200 << 40),
	},
	stabi.RegisteredSealProof_StackedDrg4GiBV1: {
		SectorMaxLifetime:      EpochsInFiveYears,
		ConsensusMinerMinPower: stabi.NewStoragePower(1 << 33),
	},
	stabi.RegisteredSealProof_StackedDrg16GiBV1: {
		SectorMaxLifetime:      EpochsInFiveYears,
		ConsensusMinerMinPower: stabi.NewStoragePower(1 << 35),
	},

	stabi.RegisteredSealProof_StackedDrg2KiBV1_1: {
		SectorMaxLifetime:      EpochsInFiveYears,
		ConsensusMinerMinPower: stabi.NewStoragePower(0),
	},
	stabi.RegisteredSealProof_StackedDrg8MiBV1_1: {
		SectorMaxLifetime:      EpochsInFiveYears,
		ConsensusMinerMinPower: stabi.NewStoragePower(16 << 20),
	},
	stabi.RegisteredSealProof_StackedDrg512MiBV1_1: {
		SectorMaxLifetime:      EpochsInFiveYears,
		ConsensusMinerMinPower: stabi.NewStoragePower(1 << 30),
	},
	stabi.RegisteredSealProof_StackedDrg32GiBV1_1: {
		SectorMaxLifetime:      EpochsInFiveYears,
		ConsensusMinerMinPower: stabi.NewStoragePower(10 << 40),
	},
	stabi.RegisteredSealProof_StackedDrg64GiBV1_1: {

		SectorMaxLifetime:      EpochsInFiveYears,
		ConsensusMinerMinPower: stabi.NewStoragePower(200 << 40),
	},
	stabi.RegisteredSealProof_StackedDrg4GiBV1_1: {
		SectorMaxLifetime:      EpochsInFiveYears,
		ConsensusMinerMinPower: stabi.NewStoragePower(1 << 33),
	},
	stabi.RegisteredSealProof_StackedDrg16GiBV1_1: {
		SectorMaxLifetime:      EpochsInFiveYears,
		ConsensusMinerMinPower: stabi.NewStoragePower(1 << 35),
	},
}

// Returns the partition size, in sectors, associated with a seal proof type.
// The partition size is the number of sectors proved in a single PoSt proof.
func SealProofWindowPoStPartitionSectors(p stabi.RegisteredSealProof, nv network.Version) (uint64, error) {
	wPoStProofType, err := p.RegisteredWindowPoStProof()
	if err != nil {
		return 0, err
	}
	return PoStProofWindowPoStPartitionSectors(wPoStProofType, nv)
}

// SectorMaximumLifetime is the maximum duration a sector sealed with this proof may exist between activation and expiration
func SealProofSectorMaximumLifetime(p stabi.RegisteredSealProof) (stabi.ChainEpoch, error) {
	info, ok := SealProofPolicies[p]
	if !ok {
		return 0, errors.Errorf("7 unsupported proof type: %v", p)
	}
	return info.SectorMaxLifetime, nil
}

// The minimum power of an individual miner to meet the threshold for leader election (in bytes).
// Motivation:
// - Limits sybil generation
// - Improves consensus fault detection
// - Guarantees a minimum fee for consensus faults
// - Ensures that a specific soundness for the power table
// Note: We may be able to reduce this in the future, addressing consensus faults with more complicated penalties,
// sybil generation with crypto-economic mechanism, and PoSt soundness by increasing the challenges for small miners.
func ConsensusMinerMinPower(p stabi.RegisteredSealProof) (stabi.StoragePower, error) {
	info, ok := SealProofPolicies[p]
	if !ok {
		return stabi.NewStoragePower(0), errors.Errorf("8 unsupported proof type: %v", p)
	}
	return info.ConsensusMinerMinPower, nil
}

// Policy values associated with a PoSt proof type.
type PoStProofPolicy struct {
	WindowPoStPartitionSectors uint64
}

// Partition sizes must match those used by the proofs library.
// See https://github.com/filecoin-project/rust-fil-proofs/blob/master/filecoin-proofs/src/constants.rs#L85
var PoStProofPolicies = map[stabi.RegisteredPoStProof]*PoStProofPolicy{
	stabi.RegisteredPoStProof_StackedDrgWindow2KiBV1: {
		WindowPoStPartitionSectors: 2,
	},
	stabi.RegisteredPoStProof_StackedDrgWindow8MiBV1: {
		WindowPoStPartitionSectors: 2,
	},
	stabi.RegisteredPoStProof_StackedDrgWindow512MiBV1: {
		WindowPoStPartitionSectors: 2,
	},
	stabi.RegisteredPoStProof_StackedDrgWindow32GiBV1: {
		WindowPoStPartitionSectors: 2349,
	},
	stabi.RegisteredPoStProof_StackedDrgWindow64GiBV1: {
		WindowPoStPartitionSectors: 2300,
	},
	stabi.RegisteredPoStProof_StackedDrgWindow4GiBV1: {
		WindowPoStPartitionSectors: 600,
	},
	stabi.RegisteredPoStProof_StackedDrgWindow16GiBV1: {
		WindowPoStPartitionSectors: 2500,
	},
	// Winning PoSt proof types omitted.
}

// Returns the partition size, in sectors, associated with a Window PoSt proof type.
// The partition size is the number of sectors proved in a single PoSt proof.
func PoStProofWindowPoStPartitionSectors(p stabi.RegisteredPoStProof, nv network.Version) (uint64, error) {
	info, ok := PoStProofPolicies[p]
	if !ok {
		return 0, errors.Errorf("2 unsupported proof type: %v", p)
	}
	if nv < network.Version4 {
		if p == stabi.RegisteredPoStProof_StackedDrgWindow16GiBV1 {
			return uint64(2300), nil
		}
	}
	return info.WindowPoStPartitionSectors, nil
}
