package errors_test

import "github.com/chain/txvm/errors"

var ErrInvalidKey = errors.New("invalid key")

func ExampleSub() {
	err := sign()
	if err != nil {
		err = errors.Sub(ErrInvalidKey, err)
		return
	}
}

func ExampleSub_return() {
	err := sign()
	err = errors.Sub(ErrInvalidKey, err)
	return
}

func sign() error { return nil }
