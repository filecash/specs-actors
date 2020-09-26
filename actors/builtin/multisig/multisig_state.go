package multisig

import (
	address "github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/exitcode"
	cid "github.com/ipfs/go-cid"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/specs-actors/v2/actors/util/adt"
)

type State struct {
	// Signers may be either public-key or actor ID-addresses. The ID address is canonical, but doesn't exist
	// for a public key that has not yet received a message on chain.
	// If any signer address is a public-key address, it will be resolved to an ID address and persisted
	// in this state when the address is used.
	Signers               []address.Address
	NumApprovalsThreshold uint64
	NextTxnID             TxnID

	// Linear unlock
	InitialBalance abi.TokenAmount
	StartEpoch     abi.ChainEpoch
	UnlockDuration abi.ChainEpoch

	PendingTxns cid.Cid
}

func (st *State) SetLocked(startEpoch abi.ChainEpoch, unlockDuration abi.ChainEpoch, lockedAmount abi.TokenAmount) {
	st.StartEpoch = startEpoch
	st.UnlockDuration = unlockDuration
	st.InitialBalance = lockedAmount
}

func (st *State) AmountLocked(elapsedEpoch abi.ChainEpoch) abi.TokenAmount {
	if elapsedEpoch >= st.UnlockDuration {
		return abi.NewTokenAmount(0)
	}
	if elapsedEpoch <= 0 {
		return st.InitialBalance
	}

	unlockDuration := big.NewInt(int64(st.UnlockDuration))
	remainingLockDuration := big.Sub(unlockDuration, big.NewInt(int64(elapsedEpoch)))

	// locked = ceil(InitialBalance * remainingLockDuration / UnlockDuration)
	numerator := big.Mul(st.InitialBalance, remainingLockDuration)
	denominator := unlockDuration
	quot := big.Div(numerator, denominator)
	rem := big.Mod(numerator, denominator)

	locked := quot
	if !rem.IsZero() {
		locked = big.Add(locked, big.NewInt(1))
	}
	return locked
}

// return nil if MultiSig maintains required locked balance after spending the amount, else return an error.
func (st *State) assertAvailable(currBalance abi.TokenAmount, amountToSpend abi.TokenAmount, currEpoch abi.ChainEpoch) error {
	if amountToSpend.LessThan(big.Zero()) {
		return xerrors.Errorf("amount to spend %s less than zero", amountToSpend.String())
	}
	if currBalance.LessThan(amountToSpend) {
		return xerrors.Errorf("current balance %s less than amount to spend %s", currBalance.String(), amountToSpend.String())
	}
	if amountToSpend.IsZero() {
		// Always permit a transaction that sends no value, even if the lockup exceeds the current balance.
		return nil
	}

	remainingBalance := big.Sub(currBalance, amountToSpend)
	amountLocked := st.AmountLocked(currEpoch - st.StartEpoch)
	if remainingBalance.LessThan(amountLocked) {
		return xerrors.Errorf("balance %s if spent %s would be less than locked amount %s",
			remainingBalance.String(), amountToSpend, amountLocked.String())
	}

	return nil
}

func getPendingTransaction(ptx *adt.Map, txnID TxnID) (Transaction, error) {
	var out Transaction
	found, err := ptx.Get(txnID, &out)
	if err != nil {
		return Transaction{}, xerrors.Errorf("failed to read transaction: %w", err)
	}
	if !found {
		return Transaction{}, exitcode.ErrNotFound.Wrapf("failed to find transaction %v", txnID)
	}
	return out, nil
}
