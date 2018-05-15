package testutil

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"
	"unsafe"
)

type visit struct {
	a1, a2 unsafe.Pointer
	typ    reflect.Type
}

// DeepEqual is similar to reflect.DeepEqual, but treats nil as equal
// to empty maps and slices. Some of the implementation is cribbed
// from Go's reflect package.
func DeepEqual(x, y interface{}) bool {
	vx := reflect.ValueOf(x)
	vy := reflect.ValueOf(y)
	return deepValueEqual(vx, vy, make(map[visit]bool))
}

func deepValueEqual(x, y reflect.Value, visited map[visit]bool) bool {
	if isEmpty(x) && isEmpty(y) {
		return true
	}
	if !x.IsValid() {
		return !y.IsValid()
	}
	if !y.IsValid() {
		return false
	}

	tx := x.Type()
	ty := y.Type()
	if tx != ty {
		return false
	}

	switch tx.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.Struct:
		if x.CanAddr() && y.CanAddr() {
			a1 := unsafe.Pointer(x.UnsafeAddr())
			a2 := unsafe.Pointer(y.UnsafeAddr())
			if uintptr(a1) > uintptr(a2) {
				// Canonicalize order to reduce number of entries in visited.
				// Assumes non-moving garbage collector.
				a1, a2 = a2, a1
			}
			v := visit{a1, a2, tx}
			if visited[v] {
				return true
			}
			visited[v] = true
		}
	}

	switch tx.Kind() {
	case reflect.Bool:
		return x.Bool() == y.Bool()

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return x.Int() == y.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return x.Uint() == y.Uint()

	case reflect.Float32, reflect.Float64:
		return x.Float() == y.Float()

	case reflect.Complex64, reflect.Complex128:
		return x.Complex() == y.Complex()

	case reflect.String:
		return x.String() == y.String()

	case reflect.Array:
		for i := 0; i < tx.Len(); i++ {
			if !deepValueEqual(x.Index(i), y.Index(i), visited) {
				return false
			}
		}
		return true

	case reflect.Slice:
		ttx := tx.Elem()
		tty := ty.Elem()
		if ttx != tty {
			return false
		}
		if x.Len() != y.Len() {
			return false
		}
		for i := 0; i < x.Len(); i++ {
			if !deepValueEqual(x.Index(i), y.Index(i), visited) {
				return false
			}
		}
		return true

	case reflect.Interface:
		if x.IsNil() {
			return y.IsNil()
		}
		if y.IsNil() {
			return false
		}
		return deepValueEqual(x.Elem(), y.Elem(), visited)

	case reflect.Ptr:
		if x.Pointer() == y.Pointer() {
			return true
		}
		return deepValueEqual(x.Elem(), y.Elem(), visited)

	case reflect.Struct:
		for i := 0; i < tx.NumField(); i++ {
			if !deepValueEqual(x.Field(i), y.Field(i), visited) {
				return false
			}
		}
		return true

	case reflect.Map:
		if x.Pointer() == y.Pointer() {
			return true
		}
		if x.Len() != y.Len() {
			return false
		}
		for _, k := range x.MapKeys() {
			if !deepValueEqual(x.MapIndex(k), y.MapIndex(k), visited) {
				return false
			}
		}
		return true

	case reflect.Func:
		return x.IsNil() && y.IsNil()
	}
	return false
}

func isEmpty(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	switch v.Type().Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Ptr:
		return v.IsNil()

	case reflect.Slice, reflect.Map:
		return v.IsNil() || v.Len() == 0
	}
	return false
}

func WithSort(path []string, less func(a, b interface{}) bool) func(a, b interface{}) {
	return func(a, b interface{}) {
		for _, p := range path {
			mapA, ok := a.(map[string]interface{})
			if !ok {
				return
			}
			a = mapA[p]
			mapB, ok := b.(map[string]interface{})
			if !ok {
				return
			}
			b = mapB[p]
		}
		aSlice, ok := a.([]interface{})
		if !ok {
			return
		}
		bSlice, ok := b.([]interface{})
		if !ok {
			return
		}
		sort.Slice(aSlice, func(i, j int) bool { return less(aSlice[i], aSlice[j]) })
		sort.Slice(bSlice, func(i, j int) bool { return less(bSlice[i], bSlice[j]) })
	}
}

func IsJSONSubset(t testing.TB, a, b []byte, opts ...func(a, b interface{})) bool {
	var aval, bval interface{}
	err := json.Unmarshal(a, &aval)
	if err != nil {
		t.Fatalf("a: %s is not valid json", string(a))
	}
	err = json.Unmarshal(b, &bval)
	if err != nil {
		t.Fatalf("b: %s is not valid json", string(b))
	}
	for _, o := range opts {
		o(aval, bval)
	}
	return IsSubset(aval, bval)
}

func IsSubset(a, b interface{}) bool {
	if reflect.TypeOf(a) != reflect.TypeOf(b) {
		return false
	}
	switch va := a.(type) {
	case map[string]interface{}:
		// don't check length or presence of fields
		// exclusively in a, since this is a subset check
		vb := b.(map[string]interface{})
		for k := range vb {
			if !IsSubset(va[k], vb[k]) {
				return false
			}
		}
		return true
	case []interface{}:
		vb := b.([]interface{})
		if len(va) != len(vb) {
			return false
		}
		for i := range va {
			if !IsSubset(va[i], vb[i]) {
				return false
			}
		}
		return true
	case float64:
		return va == b.(float64)
	case bool:
		return va == b.(bool)
	case string:
		return va == b.(string)
	case nil:
		return true
	default:
		return false
	}
}
