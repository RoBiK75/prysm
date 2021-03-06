package kv

import (
	"context"
	"encoding/hex"
	"flag"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	dbTypes "github.com/prysmaticlabs/prysm/slasher/db/types"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
	"github.com/urfave/cli/v2"
)

type spansTestStruct struct {
	name           string
	epoch          uint64
	spansHex       string
	spansResultHex string
	validator1Span types.Span
	err            error
}

var spanNewTests []spansTestStruct

func init() {
	spanNewTests = []spansTestStruct{
		{
			name:           "span too small",
			epoch:          1,
			spansHex:       "00000000",
			spansResultHex: "",
			validator1Span: types.Span{},
			err:            types.ErrWrongSize,
		},
		{
			name:           "no validator 1 in spans",
			epoch:          2,
			spansHex:       "00000000000000",
			spansResultHex: "00000000000000",
			validator1Span: types.Span{},
			err:            nil,
		},
		{
			name:           "validator 1 in spans",
			epoch:          3,
			spansHex:       "0000000000000001000000000000",
			spansResultHex: "0000000000000001000000000000",
			validator1Span: types.Span{MinSpan: 1},
			err:            nil,
		},
	}

}

func TestValidatorSpans_NilDB(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(&app, set, nil))
	ctx := context.Background()

	validatorIdx := uint64(1)
	es, err := db.EpochSpans(ctx, validatorIdx, false)
	require.NoError(t, err, "Nil EpochSpansMap should not return error")
	cleanStore, err := types.NewEpochStore([]byte{})
	require.NoError(t, err)
	require.DeepEqual(t, es, cleanStore, "EpochSpans should return empty byte array if no record exists in the db")
}

func TestStore_SaveReadEpochSpans(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(&app, set, nil))
	ctx := context.Background()

	for _, tt := range spanNewTests {
		t.Run(tt.name, func(t *testing.T) {
			spans, err := hex.DecodeString(tt.spansHex)
			require.NoError(t, err)
			es, err := types.NewEpochStore(spans)
			if tt.err != nil {
				require.ErrorContains(t, tt.err.Error(), err)
			} else {
				require.NoError(t, err)
			}
			require.NoError(t, db.SaveEpochSpans(ctx, tt.epoch, es, false))
			sm, err := db.EpochSpans(ctx, tt.epoch, false)
			require.NoError(t, err, "Failed to get validator spans")
			spansResult, err := hex.DecodeString(tt.spansResultHex)
			require.NoError(t, err)
			esr, err := types.NewEpochStore(spansResult)
			require.NoError(t, err)
			require.DeepEqual(t, sm, esr, "Get should return validator spans: %v", spansResult)

			s, err := es.GetValidatorSpan(1)
			require.NoError(t, err, "Failed to get validator 1 span")
			require.DeepEqual(t, tt.validator1Span, s, "Get should return validator span for validator 2: %v", tt.validator1Span)
		})
	}
}

func TestStore_SaveEpochSpans_ToCache(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(&app, set, nil))
	ctx := context.Background()

	spansToSave := map[uint64]types.Span{
		0:     {MinSpan: 5, MaxSpan: 69, SigBytes: [2]byte{40, 219}, HasAttested: false},
		10:    {MinSpan: 43, MaxSpan: 32, SigBytes: [2]byte{10, 13}, HasAttested: true},
		1000:  {MinSpan: 40, MaxSpan: 36, SigBytes: [2]byte{61, 151}, HasAttested: false},
		10000: {MinSpan: 40, MaxSpan: 64, SigBytes: [2]byte{190, 215}, HasAttested: true},
		50000: {MinSpan: 40, MaxSpan: 64, SigBytes: [2]byte{190, 215}, HasAttested: true},
		100:   {MinSpan: 49, MaxSpan: 96, SigBytes: [2]byte{11, 98}, HasAttested: true},
	}
	epochStore, err := types.EpochStoreFromMap(spansToSave)
	require.NoError(t, err)

	epoch := uint64(9)
	require.NoError(t, db.SaveEpochSpans(ctx, epoch, epochStore, dbTypes.UseCache))

	esFromCache, err := db.EpochSpans(ctx, epoch, dbTypes.UseCache)
	require.NoError(t, err)
	require.DeepEqual(t, epochStore.Bytes(), esFromCache.Bytes())

	esFromDB, err := db.EpochSpans(ctx, epoch, dbTypes.UseDB)
	require.NoError(t, err)
	require.DeepEqual(t, esFromDB.Bytes(), esFromCache.Bytes())
}

func TestStore_SaveEpochSpans_ToDB(t *testing.T) {
	app := cli.App{}
	set := flag.NewFlagSet("test", 0)
	db := setupDB(t, cli.NewContext(&app, set, nil))
	ctx := context.Background()

	spansToSave := map[uint64]types.Span{
		0:      {MinSpan: 5, MaxSpan: 69, SigBytes: [2]byte{40, 219}, HasAttested: false},
		10:     {MinSpan: 43, MaxSpan: 32, SigBytes: [2]byte{10, 13}, HasAttested: true},
		1000:   {MinSpan: 40, MaxSpan: 36, SigBytes: [2]byte{61, 151}, HasAttested: false},
		10000:  {MinSpan: 40, MaxSpan: 64, SigBytes: [2]byte{190, 215}, HasAttested: true},
		100000: {MinSpan: 20, MaxSpan: 64, SigBytes: [2]byte{170, 215}, HasAttested: false},
		100:    {MinSpan: 49, MaxSpan: 96, SigBytes: [2]byte{11, 98}, HasAttested: true},
	}
	epochStore, err := types.EpochStoreFromMap(spansToSave)
	require.NoError(t, err)

	epoch := uint64(9)
	require.NoError(t, db.SaveEpochSpans(ctx, epoch, epochStore, dbTypes.UseDB))

	// Expect cache to retrieve from DB if its not in cache.
	esFromCache, err := db.EpochSpans(ctx, epoch, dbTypes.UseCache)
	require.NoError(t, err)
	require.DeepEqual(t, esFromCache.Bytes(), epochStore.Bytes())

	esFromDB, err := db.EpochSpans(ctx, epoch, dbTypes.UseDB)
	require.NoError(t, err)
	require.DeepEqual(t, epochStore.Bytes(), esFromDB.Bytes())
}
