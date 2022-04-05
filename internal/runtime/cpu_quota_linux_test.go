// Copyright (c) 2022 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

//go:build linux
// +build linux

package runtime

import (
	"errors"
	"fmt"
	"testing"

	"github.com/prashantv/gostub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/automaxprocs/internal/cgroups"
)

func TestNewQueryer(t *testing.T) {
	t.Run("use v2", func(t *testing.T) {
		stubs := newStubs(t)

		c2 := new(cgroups.CGroups2)
		stubs.StubFunc(&_newCgroups2, c2, nil)

		got, err := newQueryer()
		require.NoError(t, err)
		assert.Same(t, c2, got)
	})

	t.Run("v2 error", func(t *testing.T) {
		stubs := newStubs(t)

		giveErr := errors.New("great sadness")
		stubs.StubFunc(&_newCgroups2, nil, giveErr)

		_, err := newQueryer()
		assert.ErrorIs(t, err, giveErr)
	})

	t.Run("use v1", func(t *testing.T) {
		stubs := newStubs(t)

		stubs.StubFunc(&_newCgroups2, nil,
			fmt.Errorf("not v2: %w", cgroups.ErrNotV2))

		c1 := make(cgroups.CGroups)
		stubs.StubFunc(&_newCgroups, c1, nil)

		got, err := newQueryer()
		require.NoError(t, err)
		assert.IsType(t, c1, got, "must be a v1 cgroup")
	})

	t.Run("v1 error", func(t *testing.T) {
		stubs := newStubs(t)

		stubs.StubFunc(&_newCgroups2, nil, cgroups.ErrNotV2)

		giveErr := errors.New("great sadness")
		stubs.StubFunc(&_newCgroups, nil, giveErr)

		_, err := newQueryer()
		assert.ErrorIs(t, err, giveErr)
	})
}

func newStubs(t *testing.T) *gostub.Stubs {
	stubs := gostub.New()
	t.Cleanup(stubs.Reset)
	return stubs
}
