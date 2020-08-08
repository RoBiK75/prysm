package depositutil_test

import (
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/depositutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestDepositInput_GeneratesPb(t *testing.T) {
	k1 := bls.RandKey()
	k2 := bls.RandKey()

	result, _, err := depositutil.DepositInput(k1, k2, 0)
	if err != nil {
		t.Fatal(err)
	}
	assert.DeepEqual(t, k1.PublicKey().Marshal(), result.PublicKey)

	sig, err := bls.SignatureFromBytes(result.Signature)
	require.NoError(t, err)
	sr, err := ssz.SigningRoot(result)
	require.NoError(t, err)
	domain, err := helpers.ComputeDomain(
		params.BeaconConfig().DomainDeposit,
		nil, /*forkVersion*/
		nil, /*genesisValidatorsRoot*/
	)
	require.NoError(t, err)
	root, err := &pb.SigningData{ObjectRoot: sr[:], Domain: domain[:]}.HashTreeRoot()
	require.NoError(t, err)
	assert.Equal(t, true, sig.Verify(k1.PublicKey(), root[:]))
}

func TestVerifyDepositSignature_ValidSig(t *testing.T) {
	deposits, _, err := testutil.DeterministicDepositsAndKeys(1)
	if err != nil {
		t.Fatalf("Error Generating Deposits and Keys - %v", err)
	}
	deposit := deposits[0]
	domain, err := helpers.ComputeDomain(
		params.BeaconConfig().DomainDeposit,
		params.BeaconConfig().GenesisForkVersion,
		params.BeaconConfig().ZeroHash[:],
	)
	if err != nil {
		t.Fatalf("Error Computing Domain - %v", err)
	}
	err = depositutil.VerifyDepositSignature(deposit.Data, domain)
	if err != nil {
		t.Fatal("Deposit Verification fails with a valid signature")
	}
}

func TestVerifyDepositSignature_InvalidSig(t *testing.T) {
	deposits, _, err := testutil.DeterministicDepositsAndKeys(1)
	if err != nil {
		t.Fatalf("Error Generating Deposits and Keys - %v", err)
	}
	deposit := deposits[0]
	domain, err := helpers.ComputeDomain(
		params.BeaconConfig().DomainDeposit,
		params.BeaconConfig().GenesisForkVersion,
		params.BeaconConfig().ZeroHash[:],
	)
	if err != nil {
		t.Fatalf("Error Computing Domain - %v", err)
	}
	deposit.Data.Signature = deposit.Data.Signature[1:]
	err = depositutil.VerifyDepositSignature(deposit.Data, domain)
	if err == nil {
		t.Fatal("Deposit Verification succeeds with a invalid signature")
	}
}
