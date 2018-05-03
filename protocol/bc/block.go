package bc

import (
	"database/sql/driver"
	"encoding/hex"

	"golang.org/x/sync/errgroup"

	"github.com/golang/protobuf/proto"

	"github.com/chain/txvm/errors"
	"github.com/chain/txvm/protocol/txvm"
)

// Block describes a complete block, including its header
// and the transactions it contains.
type Block struct {
	*BlockHeader
	Transactions []*Tx
	Arguments    []interface{}
}

// MarshalText fulfills the json.Marshaler interface.
// This guarantees that blocks will get deserialized correctly
// when being parsed from HTTP requests.
func (b *Block) MarshalText() ([]byte, error) {
	bits, err := b.Bytes()
	if err != nil {
		return nil, err
	}

	enc := make([]byte, hex.EncodedLen(len(bits)))
	hex.Encode(enc, bits)
	return enc, nil
}

// UnmarshalText fulfills the encoding.TextUnmarshaler interface.
func (b *Block) UnmarshalText(text []byte) error {
	decoded := make([]byte, hex.DecodedLen(len(text)))
	_, err := hex.Decode(decoded, text)
	if err != nil {
		return err
	}
	return b.FromBytes(decoded)
}

// Scan fulfills the sql.Scanner interface.
func (b *Block) Scan(val interface{}) error {
	driverBuf, ok := val.([]byte)
	if !ok {
		return errors.New("Scan must receive a byte slice")
	}
	buf := make([]byte, len(driverBuf))
	copy(buf[:], driverBuf)
	return b.FromBytes(buf)
}

// Value fulfills the sql.driver.Valuer interface.
func (b *Block) Value() (driver.Value, error) {
	return b.Bytes()
}

// FromBytes parses a Block from a byte slice, by unmarshaling and
// converting a RawBlock protobuf.
func (b *Block) FromBytes(bits []byte) error {
	var rb RawBlock
	err := proto.Unmarshal(bits, &rb)
	if err != nil {
		return err
	}
	txs := make([]*Tx, len(rb.Transactions))
	var eg errgroup.Group
	for i := range rb.Transactions {
		i := i
		eg.Go(func() error {
			tx, err := NewTx(rb.Transactions[i].Program, rb.Transactions[i].Version, rb.Transactions[i].Runlimit)
			if err != nil {
				return err
			}
			if !tx.Finalized {
				return txvm.ErrUnfinalized
			}
			txs[i] = tx
			return nil
		})
	}
	err = eg.Wait()
	if err != nil {
		return err
	}
	b.BlockHeader = rb.Header
	b.Transactions = txs
	for _, arg := range rb.Arguments {
		switch arg.Type {
		case DataType_BYTES:
			b.Arguments = append(b.Arguments, arg.Bytes)
		case DataType_INT:
			b.Arguments = append(b.Arguments, arg.Int)
		case DataType_TUPLE:
			b.Arguments = append(b.Arguments, arg.Tuple)
		}
	}
	return nil
}

// Bytes encodes the Block as a byte slice, by converting it to a
// RawBlock protobuf and marshaling that.
func (b *Block) Bytes() ([]byte, error) {
	var txs []*RawTx
	for _, tx := range b.Transactions {
		txs = append(txs, &RawTx{
			Version:  tx.Version,
			Runlimit: tx.Runlimit,
			Program:  tx.Program,
		})
	}
	var args []*DataItem
	for _, arg := range b.Arguments {
		switch a := arg.(type) {
		case []byte:
			args = append(args, &DataItem{Type: DataType_BYTES, Bytes: a})
		case int64:
			args = append(args, &DataItem{Type: DataType_INT, Int: a})
		case []*DataItem:
			args = append(args, &DataItem{Type: DataType_TUPLE, Tuple: a})
		}
	}
	rb := &RawBlock{
		Header:       b.BlockHeader,
		Transactions: txs,
		Arguments:    args,
	}
	return proto.Marshal(rb)
}
