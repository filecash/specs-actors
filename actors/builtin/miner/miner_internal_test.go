package miner

import (
	"testing"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/minio/blake2b-simd"
	"github.com/stretchr/testify/assert"

	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/filecoin-project/specs-actors/actors/util/smoothing"
	tutils "github.com/filecoin-project/specs-actors/support/testing"
)

func TestAssignProvingPeriodBoundary(t *testing.T) {
	addr1 := tutils.NewActorAddr(t, "a")
	addr2 := tutils.NewActorAddr(t, "b")
	startEpoch := abi.ChainEpoch(1)

	// ensure the values are different for different addresses
	b1, err := assignProvingPeriodOffset(addr1, startEpoch, blake2b.Sum256)
	assert.NoError(t, err)
	assert.True(t, b1 >= 0)
	assert.True(t, b1 < WPoStProvingPeriod)

	b2, err := assignProvingPeriodOffset(addr2, startEpoch, blake2b.Sum256)
	assert.NoError(t, err)
	assert.True(t, b2 >= 0)
	assert.True(t, b2 < WPoStProvingPeriod)

	assert.NotEqual(t, b1, b2)

	// Ensure boundaries are always less than a proving period.
	for i := 0; i < 10_000; i++ {
		boundary, err := assignProvingPeriodOffset(addr1, abi.ChainEpoch(i), blake2b.Sum256)
		assert.NoError(t, err)
		assert.True(t, boundary >= 0)
		assert.True(t, boundary < WPoStProvingPeriod)
	}
}

func TestNextProvingPeriodStart(t *testing.T) {
	// At epoch zero...
	curr := e(0)
	// ... with offset zero, the first period start skips one period ahead, ...
	assert.Equal(t, WPoStProvingPeriod, nextProvingPeriodStart(curr, 0))

	// ... and all non-zero offsets are simple.
	assert.Equal(t, e(1), nextProvingPeriodStart(curr, 1))
	assert.Equal(t, e(10), nextProvingPeriodStart(curr, 10))
	assert.Equal(t, WPoStProvingPeriod-1, nextProvingPeriodStart(curr, WPoStProvingPeriod-1))

	// At epoch 1, offsets 0 and 1 start a long way forward, but offsets 2 and later start soon.
	curr = 1
	assert.Equal(t, WPoStProvingPeriod, nextProvingPeriodStart(curr, 0))
	assert.Equal(t, WPoStProvingPeriod+1, nextProvingPeriodStart(curr, 1))
	assert.Equal(t, e(2), nextProvingPeriodStart(curr, 2))
	assert.Equal(t, e(3), nextProvingPeriodStart(curr, 3))
	assert.Equal(t, WPoStProvingPeriod-1, nextProvingPeriodStart(curr, WPoStProvingPeriod-1))

	// An arbitrary mid-period epoch.
	curr = 123
	assert.Equal(t, WPoStProvingPeriod, nextProvingPeriodStart(curr, 0))
	assert.Equal(t, WPoStProvingPeriod+1, nextProvingPeriodStart(curr, 1))
	assert.Equal(t, WPoStProvingPeriod+122, nextProvingPeriodStart(curr, 122))
	assert.Equal(t, WPoStProvingPeriod+123, nextProvingPeriodStart(curr, 123))
	assert.Equal(t, e(124), nextProvingPeriodStart(curr, 124))
	assert.Equal(t, WPoStProvingPeriod-1, nextProvingPeriodStart(curr, WPoStProvingPeriod-1))

	// The final epoch in the chain's first full period
	curr = WPoStProvingPeriod - 1
	assert.Equal(t, WPoStProvingPeriod, nextProvingPeriodStart(curr, 0))
	assert.Equal(t, WPoStProvingPeriod+1, nextProvingPeriodStart(curr, 1))
	assert.Equal(t, WPoStProvingPeriod+2, nextProvingPeriodStart(curr, 2))
	assert.Equal(t, WPoStProvingPeriod+WPoStProvingPeriod-2, nextProvingPeriodStart(curr, WPoStProvingPeriod-2))
	assert.Equal(t, WPoStProvingPeriod+WPoStProvingPeriod-1, nextProvingPeriodStart(curr, WPoStProvingPeriod-1))

	// Into the chain's second period
	curr = WPoStProvingPeriod
	assert.Equal(t, 2*WPoStProvingPeriod, nextProvingPeriodStart(curr, 0))
	assert.Equal(t, WPoStProvingPeriod+1, nextProvingPeriodStart(curr, 1))
	assert.Equal(t, WPoStProvingPeriod+2, nextProvingPeriodStart(curr, 2))
	assert.Equal(t, WPoStProvingPeriod+WPoStProvingPeriod-1, nextProvingPeriodStart(curr, WPoStProvingPeriod-1))

	curr = WPoStProvingPeriod + 234
	assert.Equal(t, 2*WPoStProvingPeriod, nextProvingPeriodStart(curr, 0))
	assert.Equal(t, 2*WPoStProvingPeriod+1, nextProvingPeriodStart(curr, 1))
	assert.Equal(t, 2*WPoStProvingPeriod+233, nextProvingPeriodStart(curr, 233))
	assert.Equal(t, 2*WPoStProvingPeriod+234, nextProvingPeriodStart(curr, 234))
	assert.Equal(t, WPoStProvingPeriod+235, nextProvingPeriodStart(curr, 235))
	assert.Equal(t, WPoStProvingPeriod+WPoStProvingPeriod-1, nextProvingPeriodStart(curr, WPoStProvingPeriod-1))
}

type e = abi.ChainEpoch

func TestFaultFeeInvariants(t *testing.T) {

	// Construct plausible reward and qa power filtered estimates
	nv := network.Version0
	epochReward := abi.NewTokenAmount(100 << 53)
	rewardEstimate := smoothing.TestingConstantEstimate(epochReward) // not too much growth over ~3000 epoch projection in BR

	networkPower := abi.NewStoragePower(100 << 50)
	powerEstimate := smoothing.TestingConstantEstimate(networkPower)
	chainEpoch := int64(5000)

	t.Run("Undeclared faults are more expensive than declared faults", func(t *testing.T) {
		faultySectorPower := abi.NewStoragePower(1 << 50)

		ff := PledgePenaltyForDeclaredFault(rewardEstimate, powerEstimate, faultySectorPower, chainEpoch)
		sp := PledgePenaltyForUndeclaredFault(rewardEstimate, powerEstimate, faultySectorPower, nv, chainEpoch)
		assert.True(t, sp.GreaterThan(ff))
	})

	// constant filter estimate cumsum ratio is just multiplication and division
	// test that internal precision of BR calculation does not cost accuracy 
	// compared to simple multiplication in this case.
	t.Run("br looks right in plausible (sectorPower, networkPower, reward) range", func(t *testing.T) {
		// between 10 and 100 FIL is reasonable for near-mid future
		tensOfFIL := big.Mul(abi.NewTokenAmount(1e18), big.NewInt(50))
		rewardEstimate := smoothing.TestingConstantEstimate(tensOfFIL)
		smallPower := big.NewInt(32 << 30) // 32 GiB
		hugePower := big.NewInt(1 << 60)   // 1 EiB
		epochsPerDay := big.NewInt(builtin.EpochsInDay)
		smallPowerBRNum := big.Mul(big.Mul(smallPower, epochsPerDay), tensOfFIL)
		hugePowerBRNum := big.Mul(big.Mul(hugePower, epochsPerDay), tensOfFIL)

		ChainEpoch := int64(5000)
		// QAPower = Space * AverageQuality
		// 10s of EiBs -- lower range
		// 1.2e18 * 10 bytes * 1 quality ~ 1e19
		tensOfEiBs := big.Mul(abi.NewStoragePower(1e18), big.NewInt(10))
		lowPowerEstimate := smoothing.TestingConstantEstimate(tensOfEiBs)
		brSmallLow := ExpectedRewardForPower(rewardEstimate, lowPowerEstimate, smallPower, builtin.EpochsInDay, ChainEpoch)
		brHugeLow := ExpectedRewardForPower(rewardEstimate, lowPowerEstimate, hugePower, builtin.EpochsInDay, ChainEpoch)
		assert.Equal(t, big.Div(smallPowerBRNum, tensOfEiBs), brSmallLow)
		assert.Equal(t, big.Div(hugePowerBRNum, tensOfEiBs), brHugeLow)

		// 100s of EiBs
		// 1.2e18 * 100 bytes * 5 quality ~ 6e20
		hundredsOfEiBs := big.Mul(abi.NewStoragePower(1e18), big.NewInt(6e2))
		midPowerEstimate := smoothing.TestingConstantEstimate(hundredsOfEiBs)
		brSmallMid := ExpectedRewardForPower(rewardEstimate, midPowerEstimate, smallPower, builtin.EpochsInDay, ChainEpoch)
		brHugeMid := ExpectedRewardForPower(rewardEstimate, midPowerEstimate, hugePower, builtin.EpochsInDay, ChainEpoch)
		assert.Equal(t, big.Div(smallPowerBRNum, hundredsOfEiBs), brSmallMid)
		assert.Equal(t, big.Div(hugePowerBRNum, hundredsOfEiBs), brHugeMid)

		// 1000s of EiBs -- upper range 
		// 1.2e18 * 1000 bytes * 10 quality = 1.2e22 ~ 2e22
		thousandsOfEiBs := big.Mul(abi.NewStoragePower(1e18), big.NewInt(2e4))
		upperPowerEstimate := smoothing.TestingConstantEstimate(thousandsOfEiBs)
		brSmallUpper := ExpectedRewardForPower(rewardEstimate, upperPowerEstimate, smallPower, builtin.EpochsInDay, ChainEpoch)
		brHugeUpper := ExpectedRewardForPower(rewardEstimate, upperPowerEstimate, hugePower, builtin.EpochsInDay, ChainEpoch)
		assert.Equal(t, big.Div(smallPowerBRNum, thousandsOfEiBs), brSmallUpper)
		assert.Equal(t, big.Div(hugePowerBRNum, thousandsOfEiBs), brHugeUpper)
	})

	t.Run("Declared and Undeclared fault penalties are linear over sectorQAPower term", func(t *testing.T) {

		faultySectorAPower := abi.NewStoragePower(1 << 50)
		faultySectorBPower := abi.NewStoragePower(19 << 50)
		faultySectorCPower := abi.NewStoragePower(63 << 50)
		totalFaultPower := big.Add(big.Add(faultySectorAPower, faultySectorBPower), faultySectorCPower)

		// Declared faults
		ffA := PledgePenaltyForDeclaredFault(rewardEstimate, powerEstimate, faultySectorAPower, chainEpoch)
		ffB := PledgePenaltyForDeclaredFault(rewardEstimate, powerEstimate, faultySectorBPower, chainEpoch)
		ffC := PledgePenaltyForDeclaredFault(rewardEstimate, powerEstimate, faultySectorCPower, chainEpoch)

		ffAll := PledgePenaltyForDeclaredFault(rewardEstimate, powerEstimate, totalFaultPower, chainEpoch)

		// Because we can introduce rounding error between 1 and zero for every penalty calculation
		// we can at best expect n calculations of 1 power to be within n of 1 calculation of n powers.
		diff := big.Sub(ffAll, big.Add(ffC, big.Add(ffA, ffB)))
		assert.True(t, diff.GreaterThanEqual(big.Zero()))
		assert.True(t, diff.LessThan(big.NewInt(3)))

		// Undeclared faults
		spA := PledgePenaltyForUndeclaredFault(rewardEstimate, powerEstimate, faultySectorAPower, nv, chainEpoch)
		spB := PledgePenaltyForUndeclaredFault(rewardEstimate, powerEstimate, faultySectorBPower, nv, chainEpoch)
		spC := PledgePenaltyForUndeclaredFault(rewardEstimate, powerEstimate, faultySectorCPower, nv, chainEpoch)

		spAll := PledgePenaltyForUndeclaredFault(rewardEstimate, powerEstimate, totalFaultPower, nv, chainEpoch)

		// Because we can introduce rounding error between 1 and zero for every penalty calculation
		// we can at best expect n calculations of 1 power to be within n of 1 calculation of n powers.
		diff = big.Sub(spAll, big.Add(spC, big.Add(spA, spB)))
		assert.True(t, diff.GreaterThanEqual(big.Zero()))
		assert.True(t, diff.LessThan(big.NewInt(3)))

	})
}
