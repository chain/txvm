package txvm

import (
	"github.com/chain/txvm/errors"
)

type stack []Item

func (s stack) String() string {
	res := "stack{"
	for i, it := range s {
		if i != 0 {
			res += ", "
		}
		res += it.String()
	}
	res += "}"
	return res
}

func (s *stack) peek(n int64) (Item, bool) {
	index := int64(len(*s)) - 1 - n
	if index < 0 || index >= int64(len(*s)) {
		return nil, false
	}
	return (*s)[index], true
}

func (s *stack) push(v Item) {
	*s = append(*s, v)
}

func (s *stack) pop() (Item, bool) {
	res, ok := s.peek(0)
	if ok {
		*s = (*s)[:len(*s)-1]
	}
	return res, ok
}

func (s *stack) popN(n int64) []Item {
	var res []Item
	for n > 0 && len(*s) > 0 {
		res = append(res, (*s)[len(*s)-1])
		*s = (*s)[:len(*s)-1]
		n--
	}
	return res
}

func (s *stack) isEmpty() bool {
	return len(*s) == 0
}

func (s *stack) roll(n int64) error {
	if n < 0 || n >= int64(len(*s)) {
		return errors.Wrapf(errors.WithData(ErrStackRange, "len(stack)", len(*s)), "roll %d", n)
	}
	i := int64(len(*s)) - 1 - n
	item := (*s)[i]
	before := (*s)[:i]
	after := (*s)[i+1:]
	*s = make([]Item, 0, len(before)+len(after)+1)
	*s = append(*s, before...)
	*s = append(*s, after...)
	*s = append(*s, item)
	return nil
}

func (s *stack) bury(n int64) error {
	if n < 0 || n >= int64(len(*s)) {
		return errors.Wrapf(errors.WithData(ErrStackRange, "len(stack)", len(*s)), "bury %d", n)
	}
	item, _ := s.pop()
	i := int64(len(*s)) - n
	before := (*s)[:i]
	after := (*s)[i:]
	*s = make([]Item, 0, len(before)+len(after)+1)
	*s = append([]Item{}, before...)
	*s = append(*s, item)
	*s = append(*s, after...)
	return nil
}

func (s *stack) Len() int {
	return len(*s)
}

func opRoll(vm *VM) {
	n := int64(vm.popInt())
	err := vm.contract.stack.roll(n)
	if err != nil {
		panic(err)
	}
	vm.charge(n)
}

func opBury(vm *VM) {
	n := int64(vm.popInt())
	err := vm.contract.stack.bury(n)
	if err != nil {
		panic(err)
	}
	vm.charge(n)
}

func opReverse(vm *VM) {
	n := int64(vm.popInt())
	vals := vm.contract.stack.popN(n)
	if int64(len(vals)) != n {
		panic(errors.Wrapf(errors.WithData(ErrStackRange, "len(stack)", len(vals)), "reverse %d", n))
	}
	vm.contract.stack = append(vm.contract.stack, vals...)
	vm.charge(n)
}

func opDepth(vm *VM) {
	vm.push(Int(len(vm.argstack)))
}

func opPeek(vm *VM) {
	n := int64(vm.popInt())
	item, ok := vm.peekNth(n).(Data)
	if !ok {
		panic(errors.WithData(ErrType, "want", "Data"))
	}
	vm.chargeCopy(item)
	vm.push(item)
}
