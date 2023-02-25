package saver_test

import (
	"testing"
)

func assertEqNew[T any](comp func(a, b T) (same bool)) func(a, b T) func(*testing.T) {
	return func(a, b T) func(*testing.T) {
		return func(t *testing.T) {
			var same bool = comp(a, b)
			if !same {
				t.Errorf("Unexpected value got\n")
				t.Errorf("Expected: %v\n", b)
				t.Fatalf("Got:      %v\n", a)
			}
		}
	}
}

func assertEq[T comparable](a, b T) func(*testing.T) {
	var comp func(a, b T) (same bool) = func(a, b T) (same bool) { return a == b }
	return assertEqNew(comp)(a, b)
}

func assertTrue(a bool) func(*testing.T) { return assertEq(a, true) }

func assertNil(e error) func(*testing.T) { return assertEq(nil == e, true) }
