package txresult

import (
	"bytes"
	"sync"

	"github.com/chain/txvm/crypto/ed25519"
	"github.com/chain/txvm/protocol/bc"
	"github.com/chain/txvm/protocol/txbuilder/standard"
	"github.com/chain/txvm/protocol/txvm"
)

// Result is a container for information that can be parsed from the
// transaction log of a completed txvm program. Extra log annotations
// produced by the txvm programs in protocol/txbuilder/standard are
// understood here.
type Result struct {
	Tx          *bc.Tx
	Outputs     []*Output
	Inputs      []*Input
	Issuances   []*Issuance
	Retirements []*Retirement
	Tags        []byte
}

// Value is a value triple (amount, assetID, and anchor) parsed from
// annotations in a transaction log. Objects below use *Value fields,
// with nil signifying a record (input, output, issuance, or
// retirement) whose value couldn't be understood from the log
// (e.g. because of encryption, or because of non-standard
// annotations).
type Value struct {
	AssetID bc.Hash
	Amount  uint64
	Anchor  []byte
}

// Output contains information parsed from output records in a
// transaction log.
type Output struct {
	LogPos    uint64
	OutputID  bc.Hash
	Value     *Value
	Quorum    int
	Pubkeys   []ed25519.PublicKey
	RefData   []byte
	TokenTags []byte
	Version   int
}

// Input contains information parsed from input records in a
// transaction log.
type Input struct {
	OutputID bc.Hash
	Value    *Value
	RefData  []byte
	Quorum   int
	Pubkeys  []ed25519.PublicKey
}

// Issuance contains information parsed from issuance records in a
// transaction log.
type Issuance struct {
	Value           *Value
	RefData         []byte
	NonceCallerSeed []byte
	NonceSelfSeed   []byte
	NonceBlockID    []byte
	NonceExpMS      int64
	TimerangeMinMS  int64
	TimerangeMaxMS  int64
	Quorum          int
	Pubkeys         []ed25519.PublicKey
}

// Retirement contains information parsed from retirement records in a
// transaction log.
type Retirement struct {
	Value   *Value
	RefData []byte
}

// New produces a Result from a bc.Tx by parsing the Tx object's
// Log.
func New(tx *bc.Tx) *Result {
	result := &Result{Tx: tx}

	for _, out := range tx.Outputs {
		rOut := &Output{
			OutputID: out.ID,
			LogPos:   uint64(out.LogPos),
		}
		addOutputMeta(rOut, out, tx, out.LogPos)
		result.Outputs = append(result.Outputs, rOut)
	}

	for _, inp := range tx.Inputs {
		rInp := &Input{
			OutputID: inp.ID,
		}
		addInputMeta(rInp, inp, tx, inp.LogPos)
		result.Inputs = append(result.Inputs, rInp)
	}

	for _, iss := range tx.Issuances {
		rIss := &Issuance{
			Value: &Value{
				AssetID: iss.AssetID,
				Amount:  uint64(iss.Amount),
				Anchor:  iss.Anchor,
			},
		}
		addIssueMeta(rIss, iss, tx, iss.LogPos)
		result.Issuances = append(result.Issuances, rIss)
	}

	for _, ret := range tx.Retirements {
		rRet := &Retirement{
			Value: &Value{
				AssetID: ret.AssetID,
				Amount:  uint64(ret.Amount),
				Anchor:  ret.Anchor,
			},
		}
		addRetireMeta(rRet, tx, ret.LogPos)
		result.Retirements = append(result.Retirements, rRet)
	}

	addFinalizeMeta(result, tx, len(tx.Log)-1)

	return result
}

// Results produces a Result for each of several bc.Tx's in concurrent
// goroutines.
func Results(txs []*bc.Tx) []*Result {
	var wg sync.WaitGroup
	res := make([]*Result, len(txs))
	for i := range txs {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			res[i] = New(txs[i])
		}()
	}
	wg.Wait()
	return res
}

func addOutputMeta(out *Output, txOut bc.Output, tx *bc.Tx, logPos int) {
	switch txOut.Seed.Byte32() {
	case standard.PayToMultisigSeed1:
		out.Version = 1

	case standard.PayToMultisigSeed2:
		out.Version = 2
		tagsTuple, ok := logTuple(tx.Log[logPos-2], nil)
		if !ok {
			return
		}
		out.TokenTags = tagsTuple[2].(txvm.Bytes)

	default:
		return
	}

	refdataTuple, ok := logTuple(tx.Log[logPos-1], nil)
	if !ok {
		return
	}
	out.RefData = refdataTuple[2].(txvm.Bytes)

	quorumTuple := txOut.Stack[0].(txvm.Tuple)
	if len(quorumTuple) != 2 {
		return
	}
	if quorumTuple[0].(txvm.Bytes)[0] != txvm.IntCode {
		return
	}
	out.Quorum = int(quorumTuple[1].(txvm.Int))

	pubkeyTupleTuple := txOut.Stack[1].(txvm.Tuple)
	if len(pubkeyTupleTuple) != 2 {
		return
	}
	if pubkeyTupleTuple[0].(txvm.Bytes)[0] != txvm.TupleCode {
		return
	}
	pubkeyTuple := pubkeyTupleTuple[1].(txvm.Tuple)
	for _, p := range pubkeyTuple {
		if pubkey, ok := p.(txvm.Bytes); ok {
			out.Pubkeys = append(out.Pubkeys, ed25519.PublicKey(pubkey))
		} else {
			return
		}
	}

	val := txOut.Stack[2].(txvm.Tuple)
	out.Value = &Value{
		Amount:  uint64(val[1].(txvm.Int)),
		AssetID: bc.HashFromBytes(val[2].(txvm.Bytes)),
		Anchor:  val[3].(txvm.Bytes),
	}
}

func addInputMeta(input *Input, txIn bc.Input, tx *bc.Tx, logPos int) {
	// expect refdata log after an account-spending input:
	if logPos+1 >= len(tx.Log) {
		return
	}
	spendRefTuple := tx.Log[logPos+1]
	seed := spendRefTuple[1].(txvm.Bytes)
	if !bytes.Equal(seed, standard.PayToMultisigSeed1[:]) && !bytes.Equal(seed, standard.PayToMultisigSeed2[:]) {
		return
	}
	spendRefdata := []byte(spendRefTuple[2].(txvm.Bytes))

	quorumTuple := txIn.Stack[0].(txvm.Tuple)
	if len(quorumTuple) != 2 {
		return
	}
	if quorumTuple[0].(txvm.Bytes)[0] != txvm.IntCode {
		return
	}
	input.Quorum = int(quorumTuple[1].(txvm.Int))

	pubkeyTupleTuple := txIn.Stack[1].(txvm.Tuple)
	if len(pubkeyTupleTuple) != 2 {
		return
	}
	if pubkeyTupleTuple[0].(txvm.Bytes)[0] != txvm.TupleCode {
		return
	}
	pubkeyTuple := pubkeyTupleTuple[1].(txvm.Tuple)
	for _, p := range pubkeyTuple {
		if pubkey, ok := p.(txvm.Bytes); ok {
			input.Pubkeys = append(input.Pubkeys, ed25519.PublicKey(pubkey))
		} else {
			return
		}
	}

	val := txIn.Stack[2].(txvm.Tuple)
	input.Value = &Value{
		Amount:  uint64(val[1].(txvm.Int)),
		AssetID: bc.HashFromBytes(val[2].(txvm.Bytes)),
		Anchor:  val[3].(txvm.Bytes),
	}
	input.RefData = spendRefdata
}

func addIssueMeta(issuance *Issuance, txIss bc.Issuance, tx *bc.Tx, logPos int) {
	if logPos < 2 || logPos+1 >= len(tx.Log) {
		return
	}

	nonceTuple := tx.Log[logPos-2]
	if len(nonceTuple) != 5 {
		return
	}
	if nonceTuple[0].(txvm.Bytes)[0] != txvm.NonceCode {
		return
	}

	timerangeTuple := tx.Log[logPos-1]
	if len(timerangeTuple) != 4 {
		return
	}
	if timerangeTuple[0].(txvm.Bytes)[0] != txvm.TimerangeCode {
		return
	}

	v1Seed := standard.AssetContractSeed[1]
	v2Seed := standard.AssetContractSeed[2]
	refdataTuple, ok := logTuple(tx.Log[logPos+1], &v2Seed)
	if !ok {
		refdataTuple, ok = logTuple(tx.Log[logPos+1], &v1Seed)
		if !ok {
			return
		}
	}

	issuance.Quorum = int(txIss.Stack[0].(txvm.Tuple)[1].(txvm.Int))
	pubkeyTupleTuple := txIss.Stack[1].(txvm.Tuple)
	pubkeyTuple := pubkeyTupleTuple[1].(txvm.Tuple)
	for _, p := range pubkeyTuple {
		if pubkey, ok := p.(txvm.Bytes); ok {
			issuance.Pubkeys = append(issuance.Pubkeys, ed25519.PublicKey(pubkey))
		}
	}

	issuance.NonceCallerSeed = nonceTuple[1].(txvm.Bytes)
	issuance.NonceSelfSeed = nonceTuple[2].(txvm.Bytes)
	issuance.NonceBlockID = nonceTuple[3].(txvm.Bytes)
	issuance.NonceExpMS = int64(nonceTuple[4].(txvm.Int))

	issuance.TimerangeMinMS = int64(timerangeTuple[2].(txvm.Int))
	issuance.TimerangeMaxMS = int64(timerangeTuple[3].(txvm.Int))

	issuance.RefData = []byte(refdataTuple[2].(txvm.Bytes))
}

func addRetireMeta(retirement *Retirement, tx *bc.Tx, logPos int) {
	if logPos+1 >= len(tx.Log) {
		return
	}

	refdataTuple, ok := logTuple(tx.Log[logPos+1], &standard.RetireContractSeed)
	if !ok {
		return
	}
	refdata := []byte(refdataTuple[2].(txvm.Bytes))

	retirement.RefData = refdata
}

func addFinalizeMeta(res *Result, tx *bc.Tx, logPos int) {
	// expect 1 log entry before the finalize entry. it contains the txtags

	if logPos < 1 {
		return
	}

	tuple, ok := logTuple(tx.Log[logPos-1], nil)
	if !ok {
		return
	}
	txTags, ok := tuple[2].(txvm.Bytes)
	if !ok {
		return
	}
	res.Tags = []byte(txTags)
}

func logTuple(t txvm.Tuple, seed *[32]byte) (txvm.Tuple, bool) {
	if t[0].(txvm.Bytes)[0] != txvm.LogCode {
		return nil, false
	}
	return t, seed == nil || bytes.Equal((*seed)[:], []byte(t[1].(txvm.Bytes)))
}
