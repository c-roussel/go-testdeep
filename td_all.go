// Copyright (c) 2018, Maxime Soulé
// All rights reserved.
//
// This source code is licensed under the BSD-style license found in the
// LICENSE file in the root directory of this source tree.

package testdeep

import (
	"fmt"
	"reflect"

	"github.com/maxatome/go-testdeep/internal/ctxerr"
)

type tdAll struct {
	tdList
}

var _ TestDeep = &tdAll{}

//go:noinline

// All operator compares data against several expected values. During
// a match, all of them have to match to succeed.
//
// TypeBehind method can return a non-nil reflect.Type if all items
// known non-interface types are equal, or if only interface types
// are found (mostly issued from Isa()) and they are equal.
func All(expectedValues ...interface{}) TestDeep {
	return &tdAll{
		tdList: newList(expectedValues...),
	}
}

func (a *tdAll) Match(ctx ctxerr.Context, got reflect.Value) (err *ctxerr.Error) {
	var origErr *ctxerr.Error
	for idx, item := range a.items {
		// Use deepValueEqualFinal here instead of deepValueEqual as we
		// want to know whether an error occurred or not, we do not want
		// to accumulate it silently
		origErr = deepValueEqualFinal(
			ctx.ResetErrors().
				AddDepth(fmt.Sprintf("<All#%d/%d>", idx+1, len(a.items))),
			got, item)
		if origErr != nil {
			if ctx.BooleanError {
				return ctxerr.BooleanError
			}
			err := &ctxerr.Error{
				Message:  fmt.Sprintf("compared (part %d of %d)", idx+1, len(a.items)),
				Got:      got,
				Expected: item,
			}
			if item.IsValid() && item.Type().Implements(testDeeper) {
				err.Origin = origErr
			}
			return ctx.CollectError(err)
		}
	}
	return nil
}

func (a *tdAll) TypeBehind() reflect.Type {
	var (
		lastIfType, lastType, curType reflect.Type
		severalIfTypes                bool
	)

	//
	for _, item := range a.items {
		if !item.IsValid() {
			return nil // no need to go further
		}

		if item.Type().Implements(testDeeper) {
			curType = item.Interface().(TestDeep).TypeBehind()

			// Ignore unknown TypeBehind
			if curType == nil {
				continue
			}

			// Ignore interface pointers too (see Isa), but keep them in
			// mind in case we encounter always the same interface pointer
			if curType.Kind() == reflect.Ptr &&
				curType.Elem().Kind() == reflect.Interface {
				if lastIfType == nil {
					lastIfType = curType
				} else if lastIfType != curType {
					severalIfTypes = true
				}
				continue
			}
		} else {
			curType = item.Type()
		}

		if lastType != curType {
			if lastType != nil {
				return nil
			}
			lastType = curType
		}
	}

	// Only one type found
	if lastType != nil {
		return lastType
	}

	// Only one interface type found
	if lastIfType != nil && !severalIfTypes {
		return lastIfType
	}
	return nil
}
