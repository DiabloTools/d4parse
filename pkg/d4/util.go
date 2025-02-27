package d4

import (
	"reflect"
	"unicode"
)

func newElem[T Object](t T) T {
	elemType := reflect.TypeOf(t).Elem()
	elemPtr := reflect.New(elemType)
	return elemPtr.Interface().(T)
}

func TrimNullTerminated(x []rune) string {
	for i, c := range x {
		if c == 0 {
			return string(x[:i])
		}
	}
	return string(x)
}

func IsIndex(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}
