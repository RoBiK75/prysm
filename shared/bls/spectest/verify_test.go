package spectest

import (
	"encoding/hex"
	"path"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestVerifyMessageYaml(t *testing.T) {
	testFolders, testFolderPath := testutil.TestFolders(t, "general", "bls/verify/small")

	for i, folder := range testFolders {
		t.Run(folder.Name(), func(t *testing.T) {
			file, err := testutil.BazelFileBytes(path.Join(testFolderPath, folder.Name(), "data.yaml"))
			require.NoError(t, err)
			test := &VerifyMsgTest{}
			require.NoError(t, yaml.Unmarshal(file, test))

			pkBytes, err := hex.DecodeString(test.Input.Pubkey[2:])
			require.NoError(t, err)
			pk, err := bls.PublicKeyFromBytes(pkBytes)
			require.NoError(t, err)

			msgBytes, err := hex.DecodeString(test.Input.Message[2:])
			require.NoError(t, err)

			sigBytes, err := hex.DecodeString(test.Input.Signature[2:])
			require.NoError(t, err)
			sig, err := bls.SignatureFromBytes(sigBytes)
			if err != nil {
				if test.Output == false {
					return
				}
				t.Fatalf("Cannot unmarshal input to signature: %v", err)
			}

			verified := sig.Verify(pk, msgBytes)
			if verified != test.Output {
				t.Fatalf("Signature does not match the expected verification output. "+
					"Expected %#v but received %#v for test case %d", test.Output, verified, i)
			}
			t.Log("Success")
		})
	}
}
