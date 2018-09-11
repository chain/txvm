package standard

import (
	"bytes"
	"encoding/hex"
	"testing"

	"i10r.io/crypto/ed25519"
	"i10r.io/protocol/bc"
	"i10r.io/protocol/txvm/txvmutil"
	"i10r.io/testutil"
)

// TestSeeds ensures we don't change our standard contracts without
// also (a) retaining support for old versions and (b) adding support
// for new versions. When contracts change, the logic for marshaling
// inputs, outputs, issuances, and retirements may all change, as may
// annotation and indexing.
func TestSeeds(t *testing.T) {
	wantPayToMultisigSeed1 := mustDecodeHex("7b4f536b0aee69d8711a05361a7f1e75de68ed1b27271fb9d7e7dd5400af95a7")
	if PayToMultisigSeed1 != wantPayToMultisigSeed1 {
		t.Errorf("PayToMultisigSeed1 is %x, want %x", PayToMultisigSeed1[:], wantPayToMultisigSeed1[:])
	}

	wantPayToMultisigSeed2 := mustDecodeHex("47bb956c9e8844bf5d3cc3ed93d01e275353523142b6fb3999b5d0e11a958ffa")
	if PayToMultisigSeed2 != wantPayToMultisigSeed2 {
		t.Errorf("PayToMultisigSeed2 is %x, want %x", PayToMultisigSeed2[:], wantPayToMultisigSeed2[:])
	}

	wantAssetContractSeed1 := mustDecodeHex("8c729225bfc05c742eecd33d249560f786d54346d853cc50811e3810ac8f1f88")
	if AssetContractSeed[1] != wantAssetContractSeed1 {
		t.Errorf("AssetContractSeed is %x, want %x", AssetContractSeed, wantAssetContractSeed1)
	}

	wantAssetContractSeed2 := mustDecodeHex("e2003a140131ec4f31b328701ef1242ee470ef2f72ad21b4f55135556f70f7de")
	if AssetContractSeed[2] != wantAssetContractSeed2 {
		t.Errorf("AssetContractSeed[2] is %x, want %x", AssetContractSeed, wantAssetContractSeed2)
	}

	wantRetireContractSeed := mustDecodeHex("e318fda528672bce7466151d089c89618d87e904d62e1ced96a05b04f0269d8a")
	if RetireContractSeed != wantRetireContractSeed {
		t.Errorf("RetireContractSeed is %x, want %x", RetireContractSeed[:], wantRetireContractSeed[:])
	}
}

func TestProgCreation(t *testing.T) {
	var (
		quorum   = 1
		pubkeys  = []ed25519.PublicKey{testutil.TestPub}
		amount   = int64(101)
		assetID  = bc.HashFromBytes([]byte("assetID"))
		anchor   = []byte("anchor")
		refdata  = []byte("refdata")
		assetTag = []byte("assettag")
		blockID  = []byte("blockid")
		expMS    = uint64(1000)
		txid     = mustDecodeHex("e318fda528672bce7466151d089c89618d87e904d62e1ced96a05b04f0269d8a")
	)
	cases := []struct {
		name     string
		pre      func(*testing.T) []byte
		bytecode []byte
	}{
		{
			name: "spend multisig",
			pre: func(t *testing.T) []byte {
				var b txvmutil.Builder
				SpendMultisig(&b, quorum, pubkeys, amount, assetID, anchor, PayToMultisigSeed1[:])
				return b.Build()
			},
			bytecode: mustDecodeHexProg("60437f7b4f536b0aee69d8711a05361a7f1e75de68ed1b27271fb9d7e7dd5400af95a795012d3c37012a2e8c01012a552d51025303212a5900032a51005013410253052a2d003b022a21012a0122210118224152032a5040524244605a01025460547fda5ddf057de3cf8db186e619915b88f37f797a9be1a5a79195533741f8bec4a20154025460566065207f617373657449440000000000000000000000000000000000000000000000000065616e63686f72045406544643"),
		},
		{
			name: "issue with anchor version 2",
			pre: func(t *testing.T) []byte {
				return IssueWithAnchorContract(2, quorum, pubkeys, assetTag, amount, refdata)
			},
			bytecode: mustDecodeHexProg("2d66726566646174612e7fda5ddf057de3cf8db186e619915b88f37f797a9be1a5a79195533741f8bec4a201542e012e6761737365747461672e6065202e2e002eb3012d512709412d522d012a30010241522d2d2d2d51042b2d51052b035458332d3c37012a2e8c01012a552d51025303212a5900032a51005013410253052a2d003b022a21012a0122210118224152032a50405242444843"),
		},
		{
			name: "issue without anchor version 2",
			pre: func(t *testing.T) []byte {
				return IssueWithoutAnchorContract(2, quorum, pubkeys, assetTag, amount, refdata, blockID, expMS, nil)
			},
			bytecode: mustDecodeHexProg("66726566646174612e7fda5ddf057de3cf8db186e619915b88f37f797a9be1a5a79195533741f8bec4a201542e012e6761737365747461672e6065202e66626c6f636b69642e5f2e61e807202eb3012d512709412d522d012a30010241522d2d2d2d51042b2d51052b035458332d3c37012a2e8c01012a552d51025303212a5900032a51005013410253052a2d003b022a21012a0122210118224152032a50405242444843"),
		},
		{
			name: "compute asset id version 2",
			pre: func(t *testing.T) []byte {
				assetID := AssetID(2, quorum, pubkeys, assetTag)
				return assetID[:]
			},
			bytecode: mustDecodeHexProg("38bf33f1e39123f0202bacae7e9ccb65b427cad93a747b59c5c23cd9972fcc5d"),
		},
		{
			name: "issue with anchor version 1",
			pre: func(t *testing.T) []byte {
				return IssueWithAnchorContract(1, quorum, pubkeys, assetTag, amount, refdata)
			},
			bytecode: mustDecodeHexProg("2d66726566646174612e7fda5ddf057de3cf8db186e619915b88f37f797a9be1a5a79195533741f8bec4a201542e012e6761737365747461672e6065202e2e002eb1012d512707412d012a30010241522d2d2d2d51042b2d51052b035458332d3c37012a2e8c01012a552d51025303212a5900032a51005013410253052a2d003b022a21012a0122210118224152032a50405242444843"),
		},
		{
			name: "issue without anchor version 1",
			pre: func(t *testing.T) []byte {
				return IssueWithoutAnchorContract(1, quorum, pubkeys, assetTag, amount, refdata, blockID, expMS, nil)
			},
			bytecode: mustDecodeHexProg("66726566646174612e7fda5ddf057de3cf8db186e619915b88f37f797a9be1a5a79195533741f8bec4a201542e012e6761737365747461672e6065202e66626c6f636b69642e61e807202eb1012d512707412d012a30010241522d2d2d2d51042b2d51052b035458332d3c37012a2e8c01012a552d51025303212a5900032a51005013410253052a2d003b022a21012a0122210118224152032a50405242444843"),
		},
		{
			name: "compute asset id version 1",
			pre: func(t *testing.T) []byte {
				assetID := AssetID(1, quorum, pubkeys, assetTag)
				return assetID[:]
			},
			bytecode: mustDecodeHexProg("c893aaae2314f125cf380c273c5a8584b21e460aa276396ef51f2cbae44aa965"),
		},
		{
			name: "verify txid",
			pre: func(t *testing.T) []byte {
				return VerifyTxID(txid)
			},
			bytecode: mustDecodeHexProg("3e7fe318fda528672bce7466151d089c89618d87e904d62e1ced96a05b04f0269d8a5040"),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.pre(t)
			if !bytes.Equal(got, c.bytecode) {
				t.Fatalf("Program bytecodes do not match. \nExpected: \n%x\nGot: \n%x\n", c.bytecode, got)
			}
		})
	}
}

func mustDecodeHex(s string) [32]byte {
	var result [32]byte
	_, err := hex.Decode(result[:], []byte(s))
	if err != nil {
		panic(err)
	}
	return result
}

func mustDecodeHexProg(s string) []byte {
	out, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return out
}
