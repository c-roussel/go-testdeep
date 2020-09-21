// Copyright (c) 2020, Maxime Soulé
// All rights reserved.
//
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree.

package hooks_test

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/maxatome/go-testdeep/internal/hooks"
	"github.com/maxatome/go-testdeep/internal/test"
)

func TestAddCmpHooks(t *testing.T) {
	for _, tst := range []struct {
		name string
		cmp  interface{}
		err  string
	}{
		{
			name: "not a function",
			cmp:  "zip",
			err:  "expects a function, not a string (@1)",
		},
		{
			name: "no variadic",
			cmp:  func(a []byte, b ...byte) bool { return true },
			err:  "expects: func (T, T) bool|error not func([]uint8, ...uint8) bool (@1)",
		},
		{
			name: "in",
			cmp:  func(a, b, c int) bool { return true },
			err:  "expects: func (T, T) bool|error not func(int, int, int) bool (@1)",
		},
		{
			name: "out",
			cmp:  func(a, b int) {},
			err:  "expects: func (T, T) bool|error not func(int, int) (@1)",
		},
		{
			name: "type mismatch",
			cmp:  func(a int, b bool) bool { return true },
			err:  "expects: func (T, T) bool|error not func(int, bool) bool (@1)",
		},
		{
			name: "interface",
			cmp:  func(a, b interface{}) bool { return true },
			err:  "expects: func (T, T) bool|error not func(interface {}, interface {}) bool (@1)",
		},
		{
			name: "bad return",
			cmp:  func(a, b int) int { return 0 },
			err:  "expects: func (T, T) bool|error not func(int, int) int (@1)",
		},
	} {
		i := hooks.NewInfo()

		err := i.AddCmpHooks([]interface{}{
			func(a, b bool) bool { return true },
			tst.cmp,
		})
		if test.Error(t, err, tst.name) {
			if !strings.Contains(err.Error(), tst.err) {
				t.Errorf("<%s> does not contain <%s> for %s", err, tst.err, tst.name)
			}
		}
	}
}

func TestCmp(t *testing.T) {
	t.Run("bool", func(t *testing.T) {
		var i *hooks.Info

		handled, err := i.Cmp(reflect.ValueOf(12), reflect.ValueOf(12))
		test.NoError(t, err)
		test.IsFalse(t, handled)

		i = hooks.NewInfo()

		err = i.AddCmpHooks([]interface{}{func(a, b int) bool { return a == b }})
		test.NoError(t, err)

		handled, err = i.Cmp(reflect.ValueOf(12), reflect.ValueOf(12))
		test.NoError(t, err)
		test.IsTrue(t, handled)

		handled, err = i.Cmp(reflect.ValueOf(12), reflect.ValueOf(34))
		if err != hooks.ErrBoolean {
			test.EqualErrorMessage(t, err, hooks.ErrBoolean)
		}
		test.IsTrue(t, handled)

		handled, err = i.Cmp(reflect.ValueOf(12), reflect.ValueOf("twelve"))
		test.NoError(t, err)
		test.IsFalse(t, handled)

		handled, err = i.Cmp(reflect.ValueOf("twelve"), reflect.ValueOf("twelve"))
		test.NoError(t, err)
		test.IsFalse(t, handled)

		handled, err = (*hooks.Info)(nil).Cmp(reflect.ValueOf(1), reflect.ValueOf(2))
		test.NoError(t, err)
		test.IsFalse(t, handled)
	})

	t.Run("error", func(t *testing.T) {
		i := hooks.NewInfo()

		diffErr := errors.New("a≠b")

		err := i.AddCmpHooks([]interface{}{
			func(a, b int) error {
				if a == b {
					return nil
				}
				return diffErr
			},
		})
		test.NoError(t, err)

		handled, err := i.Cmp(reflect.ValueOf(12), reflect.ValueOf(12))
		test.NoError(t, err)
		test.IsTrue(t, handled)

		handled, err = i.Cmp(reflect.ValueOf(12), reflect.ValueOf(34))
		if err != diffErr {
			test.EqualErrorMessage(t, err, diffErr)
		}
		test.IsTrue(t, handled)
	})
}

func TestSmuggle(t *testing.T) {
	var i *hooks.Info

	got := reflect.ValueOf(123)
	handled, err := i.Smuggle(&got)
	test.NoError(t, err)
	test.IsFalse(t, handled)

	i = hooks.NewInfo()

	err = i.AddSmuggleHooks([]interface{}{func(a int) bool { return a != 0 }})
	test.NoError(t, err)

	got = reflect.ValueOf(123)
	handled, err = i.Smuggle(&got)
	test.NoError(t, err)
	test.IsTrue(t, handled)
	if test.EqualInt(t, int(got.Kind()), int(reflect.Bool)) {
		test.IsTrue(t, got.Bool())
	}

	got = reflect.ValueOf("biz")
	handled, err = i.Smuggle(&got)
	test.NoError(t, err)
	test.IsFalse(t, handled)
	test.EqualStr(t, got.String(), "biz")

	err = i.AddSmuggleHooks([]interface{}{strconv.Atoi})
	test.NoError(t, err)

	got = reflect.ValueOf("123")
	handled, err = i.Smuggle(&got)
	test.NoError(t, err)
	test.IsTrue(t, handled)
	if test.EqualInt(t, int(got.Kind()), int(reflect.Int)) {
		test.EqualInt(t, int(got.Int()), 123)
	}

	got = reflect.ValueOf("NotANumber")
	handled, err = i.Smuggle(&got)
	test.Error(t, err)
	test.IsTrue(t, handled)
}

func TestAddSmuggleHooks(t *testing.T) {
	for _, tst := range []struct {
		name    string
		smuggle interface{}
		err     string
	}{
		{
			name:    "not a function",
			smuggle: "zip",
			err:     "expects a function, not a string (@1)",
		},
		{
			name:    "no variadic",
			smuggle: func(a ...byte) bool { return true },
			err:     "expects: func (A) (B[, error]) not func(...uint8) bool (@1)",
		},
		{
			name:    "in",
			smuggle: func(a, b int) bool { return true },
			err:     "expects: func (A) (B[, error]) not func(int, int) bool (@1)",
		},
		{
			name:    "interface",
			smuggle: func(a interface{}) bool { return true },
			err:     "expects: func (A) (B[, error]) not func(interface {}) bool (@1)",
		},
		{
			name:    "out",
			smuggle: func(a int) {},
			err:     "expects: func (A) (B[, error]) not func(int) (@1)",
		},
		{
			name:    "bad return",
			smuggle: func(a int) (int, int) { return 0, 0 },
			err:     "expects: func (A) (B[, error]) not func(int) (int, int) (@1)",
		},
		{
			name:    "return interface",
			smuggle: func(a int) interface{} { return 0 },
			err:     "expects: func (A) (B[, error]) not func(int) interface {} (@1)",
		},
		{
			name:    "return interface, error",
			smuggle: func(a int) (interface{}, error) { return 0, nil },
			err:     "expects: func (A) (B[, error]) not func(int) (interface {}, error) (@1)",
		},
	} {
		i := hooks.NewInfo()

		err := i.AddSmuggleHooks([]interface{}{
			func(a int) bool { return true },
			tst.smuggle,
		})
		if test.Error(t, err, tst.name) {
			if !strings.Contains(err.Error(), tst.err) {
				t.Errorf("<%s> does not contain <%s> for %s", err, tst.err, tst.name)
			}
		}
	}
}

func TestCopy(t *testing.T) {
	var orig *hooks.Info

	ni := orig.Copy()
	if ni == nil {
		t.Errorf("Copy should never return nil, even for a nil instance")
	}

	orig = hooks.NewInfo()
	copy1 := orig.Copy()
	if copy1 == nil {
		t.Errorf("Copy should never return nil")
	}
	hookedBool := false
	test.NoError(t, copy1.AddSmuggleHooks([]interface{}{
		func(in bool) bool { hookedBool = true; return in },
	}))

	gotBool := reflect.ValueOf(true)

	// orig instance does not have any hook
	handled, _ := orig.Smuggle(&gotBool)
	test.IsFalse(t, hookedBool)
	test.IsFalse(t, handled)

	// new bool smuggle hook OK
	hookedBool = false
	handled, _ = copy1.Smuggle(&gotBool)
	test.IsTrue(t, hookedBool)
	test.IsTrue(t, handled)

	copy2 := copy1.Copy()
	if copy2 == nil {
		t.Errorf("Copy should never return nil")
	}
	hookedInt := false
	test.NoError(t, copy2.AddSmuggleHooks([]interface{}{
		func(in int) int { hookedInt = true; return in },
	}))

	// bool smuggle hook inherited from copy1
	hookedBool = false
	handled, _ = copy2.Smuggle(&gotBool)
	test.IsTrue(t, hookedBool)
	test.IsTrue(t, handled)

	gotInt := reflect.ValueOf(123)

	// new int smuggle hook not available in copy1 instance
	hookedInt = false
	handled, _ = copy1.Smuggle(&gotInt)
	test.IsFalse(t, hookedInt)
	test.IsFalse(t, handled)

	// new int smuggle hook OK
	hookedInt = false
	handled, _ = copy2.Smuggle(&gotInt)
	test.IsTrue(t, hookedInt)
	test.IsTrue(t, handled)
	test.IsTrue(t, handled)
}
