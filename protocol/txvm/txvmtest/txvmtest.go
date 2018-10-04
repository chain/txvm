package txvmtest

import (
	"encoding/hex"
	"fmt"

	"github.com/chain/txvm/crypto/ed25519"
	"github.com/chain/txvm/protocol/txvm"
	"github.com/chain/txvm/protocol/txvm/asm"
)

// sample transactions

// Standard p2sp contract
// contract stack must look like [QUORUM PUBKEYn PUBKEYn-1 ... PUBKEY1 N VALUE]
const p2spUnlockSrc = `
put      # arg: [... VALUE]
         # con: [QUORUM PUBKEYn PUBKEYn-1 ... PUBKEY1 N]

# "deferred multisig check" clause:
[
	# arg: [SIGn SIGn-1 ... SIG1 PROGRAM]
	# con: [QUORUM PUBKEYn PUBKEYn-1 ... PUBKEY1 N ANCHOR]

  get        # arg: [SIGn SIGn-1 ... SIG1]
             # con: [QUORUM PUBKEYn PUBKEYn-1 ... PUBKEY1 N ANCHOR PROGRAM]
  0          # con: [QUORUM PUBKEYn PUBKEYn-1 ... PUBKEY1 N PROGRAM 0]
	2 roll     # con: [QUORUM PUBKEYn PUBKEYn-1 ... PUBKEY1 PROGRAM 0 N]

	$sigstart

		dup 0 eq       # con: [QUORUM PUBKEYn PUBKEYn-1 ... PUBKEY1 PROGRAM 0 N N==0]
		jumpif:$sigend # con: [QUORUM PUBKEYn PUBKEYn-1 ... PUBKEY1 PROGRAM 0 N]
    2 peek         # con: [QUORUM PUBKEYn PUBKEYn-1 ... PUBKEY1 PROGRAM 0 N PROGRAM]
    4 roll         # con: [QUORUM PUBKEYn PUBKEYn-1 ... PROGRAM 0 N PROGRAM PUBKEY1]
    get            # arg: [SIGn SIGn-1 ... SIG2]
                   # con: [QUORUM PUBKEYn PUBKEYn-1 ... PUBKEY1 PROGRAM 0 N PUBKEY1 PROGRAM SIG1]
    0 checksig     # con: [QUORUM PUBKEYn PUBKEYn-1 ... PROGRAM 0 N <checksigresult>]
    2 roll add     # con: [QUORUM PUBKEYn PUBKEYn-1 ... PROGRAM N <checksigtotal>]
    swap 1 sub     # con: [QUORUM PUBKEYn PUBKEYn-1 ... PROGRAM <checksigtotal> N-1]
    jump:$sigstart

	$sigend

	# arg: []
	# con: [QUORUM PROGRAM PROGANCHOR <checksigtotal> 0]

  drop         # con: [QUORUM PROGRAM <checksigtotal>]
  2 roll       # con: [PROGRAM <checksigtotal> QUORUM]
  eq verify    # con: [PROGRAM]
  exec
] yield
`

// arg stack: [... VALUE PUBKEY1 PUBKEY2 ... PUBKEYn N QUORUM]
const p2spSrcFmt = `
get get # arg stack: [... VALUE PUBKEY1 PUBKEY2 ... PUBKEYn]
        # con stack: [QUORUM N]
dup     # con: [QUORUM N N]

$pkstart

	dup 0 eq      # con: [QUORUM N N N==0]
	jumpif:$pkend # con: [QUORUM N N]
  get 2 bury    # arg: [... VALUE PUBKEY1 ... PUBKEYn-1]
								# con: [QUORUM PUBKEYn N N]
	1 sub         # con: [QUORUM PUBKEYn N N-1]
	jump:$pkstart

$pkend

# arg: [... VALUE]
# con: [QUORUM PUBKEYn PUBKEYn-1 ... PUBKEY1 N 0]

drop get # con: [QUORUM PUBKEYn PUBKEYn-1 ... PUBKEY1 N VALUE]

# "unlock" clause:
[%s] output
`

var (
	p2spSrc    = fmt.Sprintf(p2spSrcFmt, p2spUnlockSrc)
	p2spUnlock = mustAssemble(p2spUnlockSrc)
)

var (
	// private keys
	dollarPriv = mustDecodeHex("0da08740ed8d9e5f83d53550fe735e5f0becd79b706a53df7a4c2b0a2436a5aec5a4b271de6397f2d28ed428972296f1547636161c9fe93ec86386d987ecaafb")
	alicePriv  = mustDecodeHex("1c9214e1b8cc9663da918d84710351e4c9e0d6b8c37f95e3ca32bd4af63ebffe4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41")
	bobPriv    = mustDecodeHex("7346606d3a39cb97f77763f03ca2c46f12129b3b662056ac361f8bb58d4bad93db09ef920e5eddf41d109b71cdc9aed34d10311e038281273d01c7b18c1413ff")
)

// "dollar" quorum 1 pubkey c5a4b271de6397f2d28ed428972296f1547636161c9fe93ec86386d987ecaafb
// "dollar" assetid a9edd0fe349fac6ad2301dea61b99b201b0c25de8367cd3c1e8a8e6698fa8e43

// Programs
var (
	// SimplePayment is the txvm source for a simple transfer of a single UTXO.
	SimplePayment = `
	{'C',
	  'contractseed',
		[put [txid 1 roll get 0 checksig verify] yield],
	  {'S', x'4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41'},
	  {'V', 10, x'd073785d7dffc98c69ef62bbc6c8efde78a3286a848f570f8028695048a8f62d', 'anchor'}
	} input call

	# argstack: [$10-value signature-check-contract]

	get get        # con stack: [sig-check-contract $10-value]
	splitzero swap # con stack: [sig-check-contract $0-value $10-value]
	put            # arg stack: [$10-value]
				   # con stack: [sig-check-contract $0-value]
	x'1111111111111111111111111111111111111111111111111111111111111111' put
	[get get       # con stack: [pubkey $10-value]
	 [put [txid 1 roll get 0 checksig verify] yield] output
	] contract call

	# stack: [signature-check-contract $0-value]

	finalize
`

	// Issuance is the txvm source for a simple issuance.
	Issuance = `
	'' put
	10 put
	'blockchainidblockchainidblockcha' 10 nonce put
	[get get get issue put x'c5a4b271de6397f2d28ed428972296f1547636161c9fe93ec86386d987ecaafb' [txid 1 roll get 0 checksig verify] yield] contract call

	get get
	splitzero swap put
	x'1111111111111111111111111111111111111111111111111111111111111111' put
	[get get [put [txid 1 roll get 0 checksig verify] yield] output] contract call

	finalize
`

	StackLimitTest = `
	{'C',
	  'contractseed',
		[put [txid 1 roll get 0 checksig verify] yield],
	  {'S', x'4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41'},
	  {'V', 10, x'd073785d7dffc98c69ef62bbc6c8efde78a3286a848f570f8028695048a8f62d', 'anchor'}
	} input call

	# argstack: [$10-value signature-check-contract]

	get get        # con stack: [sig-check-contract $10-value]
	splitzero swap # con stack: [sig-check-contract $0-value $10-value]
	put            # arg stack: [$10-value]
				   # con stack: [sig-check-contract $0-value]
	x'1111111111111111111111111111111111111111111111111111111111111111' put
	[get get       # con stack: [pubkey $10-value]
	 [put [txid 1 roll get 0 checksig verify] yield] 4 roll output
	] contract call

	finalize
	x'21e58486696d3c66f08b602e6d102ff4efd5c9b45331564ca06882f4468ed33d036337e0e1e3999e7d872d45df139fc6ac3b76874d113616f22a2156e6d8710b' put call
`

	// SimplePayment2 is the txvm source for a simple transfer of a
	// single utxo using a (1-of-1) multisig check.
	SimplePayment2 = fmt.Sprintf(`
	{'C',
	  'contractseed',
		x'%x',
	  {'Z', 1},
	  {'S', x'4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41'},
	  {'Z', 1},
	  {'V', 10, x'd073785d7dffc98c69ef62bbc6c8efde78a3286a848f570f8028695048a8f62d', 'anchor'}
	} input call   # arg stack now: [... VALUE SIGCHECKCONTRACT]
	get get        # con stack now: [... SIGCHECKCONTRACT VALUE]
	splitzero swap # con stack now: [... SIGCHECKCONTRACT ZEROVAL VALUE]
	put x'1111111111111111111111111111111111111111111111111111111111111111' put 1 put 1 put
	[%s] contract call
	finalize
`, p2spUnlock, p2spSrc)

	SplitPayment = fmt.Sprintf(`
	{'C',
	  'contractseed',
		x'%x',
	  {'Z', 1},
	  {'S', x'4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41'},
	  {'Z', 1},
	  {'V', 10, x'd073785d7dffc98c69ef62bbc6c8efde78a3286a848f570f8028695048a8f62d', 'anchor'}
	} input call                  # arg stack now: [... 10VAL SIGCHECKCONTRACT]
	get get                       # con stack now: [... SIGCHECKCONTRACT 10VAL]
	splitzero swap                # to get zeroval for finalize. con stack now: [... SIGCHECKCONTRACT ZEROVAL 10VAL]
	splitzero swap splitzero swap # to get zerovals for proving values. [... ZEROVAL ZEROVAL ZEROVAL 10VAL]
	4 split                       # [... SIGCHECKCONTRACT ZEROVAL ZEROVAL ZEROVAL 6VAL 4VAL]
	2 roll                        # move ZEROVAL to be on top of stack: [... ZEROVAL 6VAL 4VAL ZEROVAL]
	drop                          # move ZEROVAL to be on top of stack: [... ZEROVAL 6VAL 4VAL]
	2 roll
	drop
	put x'1111111111111111111111111111111111111111111111111111111111111111' put 1 put 1 put
	[%s] contract call
	put x'2222222222222222222222222222222222222222222222222222222222222222' put 1 put 1 put
	[%s] contract call
	finalize
`, p2spUnlock, p2spSrc, p2spSrc)  // arg stack now: [... 10VAL SIGCHECKCONTRACT]

	MergePayment = fmt.Sprintf(`
	{'C',
	  'contractseed',
		x'%x',
	  {'Z', 1},
	  {'S', x'4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41'},
	  {'Z', 1},
	  {'V', 10, x'd073785d7dffc98c69ef62bbc6c8efde78a3286a848f570f8028695048a8f62d', 'anchor'}
	} input call
	{'C',
	  'contractseed',
		x'%x',
	  {'Z', 1},
	  {'S', x'db09ef920e5eddf41d109b71cdc9aed34d10311e038281273d01c7b18c1413ff'},
	  {'Z', 1},
	  {'V', 15, x'd073785d7dffc98c69ef62bbc6c8efde78a3286a848f570f8028695048a8f62d', 'anchor'}
	} input call # arg stack: [10VAL 10SIGCHECKCONTRACT 15VAL 15SIGCHECKCONTRACT]
	get get get get     # con stack: [15SIGCHECKCONTRACT 15VAL 10SIGCHECKCONTRACT 10VAL]
	splitzero splitzero # [... ZEROVAL ZEROVAL]
	2 roll              # [15SIGCHECKCONTRACT 15VAL 10SIGCHECKCONTRACT ZEROVAL ZEROVAL 10VAL]
	4 roll              # [15SIGCHECKCONTRACT 10SIGCHECKCONTRACT ZEROVAL ZEROVAL 10VAL 15VAL]
	merge swap drop     # [... ZEROVAL 25VAL]
	put x'1111111111111111111111111111111111111111111111111111111111111111' put 1 put 1 put
	[%s] contract call
	finalize
`, p2spUnlock, p2spUnlock, p2spSrc)

	Retirement = fmt.Sprintf(`
	{'C',
	  'contractseed',
		x'%x',
	  {'Z', 1},
	  {'S', x'4a771e03af3f5705ec280ac8761d568776fb2b650da9067d3f3ef7010b588d41'},
	  {'Z', 1},
	  {'V', 10, x'd073785d7dffc98c69ef62bbc6c8efde78a3286a848f570f8028695048a8f62d', 'anchor'}
	} input call
	get get splitzero swap retire finalize
`, p2spUnlock)
)

func init() {
	SimplePayment = addSig(SimplePayment, alicePriv)
	Issuance = addSig(Issuance, dollarPriv)
	SimplePayment2 = addPayToSig(SimplePayment2, alicePriv)
	SplitPayment = addPayToSig(SplitPayment, alicePriv)
	MergePayment = addPayToSig(addPayToSig(MergePayment, alicePriv), bobPriv)
	Retirement = addPayToSig(Retirement, alicePriv)
}

// payToTxid produces a program (suitable for use as the argument to
// MultisigProgram) that verifies the tx has a specific id.
func payToTxid(txid [32]byte) string {
	return fmt.Sprintf("txid x'%x' eq verify", txid[:])
}

func addSig(src string, priv []byte) string {
	prog := mustAssemble(src)
	vm, err := txvm.Validate(prog, 3, 100000, txvm.StopAfterFinalize)
	must(err)
	if !vm.Finalized {
		panic(txvm.ErrUnfinalized)
	}
	return src + fmt.Sprintf("x'%x' put call", ed25519.Sign(priv, vm.TxID[:]))
}

func addPayToSig(src string, priv []byte) string {
	prog := mustAssemble(src)
	vm, err := txvm.Validate(prog, 3, 100000, txvm.StopAfterFinalize)
	must(err)
	if !vm.Finalized {
		panic(txvm.ErrUnfinalized)
	}

	txidProgSrc := payToTxid(vm.TxID)
	txidProg := mustAssemble(txidProgSrc)
	sig := ed25519.Sign(priv, txidProg)

	return src + fmt.Sprintf(" x'%x' put [%s] put call", sig, txidProgSrc)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func mustDecodeHex(hexstr string) []byte {
	decoded, err := hex.DecodeString(hexstr)
	must(err)
	return decoded
}

func mustAssemble(src string) []byte {
	res, err := asm.Assemble(src)
	must(err)
	return res
}
