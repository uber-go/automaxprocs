// Copyright (c) 2017 Uber Technologies, Inc.
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

package maxprocs

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"testing"

	iruntime "go.uber.org/automaxprocs/internal/runtime"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withMax(t testing.TB, n int, f func()) {
	prevStr, ok := os.LookupEnv(_maxProcsKey)
	want := strconv.FormatInt(int64(n), 10)
	require.NoError(t, os.Setenv(_maxProcsKey, want), "couldn't set GOMAXPROCS")
	f()
	if ok {
		require.NoError(t, os.Setenv(_maxProcsKey, prevStr), "couldn't restore original GOMAXPROCS value")
		return
	}
	require.NoError(t, os.Unsetenv(_maxProcsKey), "couldn't clear GOMAXPROCS")
}

func testLogger() (*bytes.Buffer, Option) {
	buf := bytes.NewBuffer(nil)
	printf := func(template string, args ...interface{}) {
		fmt.Fprintf(buf, template, args...)
	}
	return buf, Logger(printf)
}

func stubProcs(f func(int) (int, iruntime.CPUQuotaStatus, error)) Option {
	return optionFunc(func(cfg *config) {
		cfg.procs = f
	})
}

func TestLogger(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		// Calling Set without options should be safe.
		undo, err := Set()
		defer undo()
		require.NoError(t, err, "Set failed")
	})

	t.Run("override", func(t *testing.T) {
		buf, opt := testLogger()
		undo, err := Set(opt)
		defer undo()
		require.NoError(t, err, "Set failed")
		assert.True(t, buf.Len() > 0, "didn't capture log output")
	})
}

func TestSet(t *testing.T) {
	// Ensure that we've undone any modifications correctly.
	prev := currentMaxProcs()
	defer func() {
		require.Equal(t, prev, currentMaxProcs(), "didn't undo GOMAXPROCS changes")
	}()

	t.Run("EnvVarPresent", func(t *testing.T) {
		withMax(t, 42, func() {
			prev := currentMaxProcs()
			undo, err := Set()
			defer undo()
			require.NoError(t, err, "Set failed")
			assert.Equal(t, prev, currentMaxProcs(), "shouldn't alter GOMAXPROCS")
		})
	})

	t.Run("ErrorReadingQuota", func(t *testing.T) {
		opt := stubProcs(func(int) (int, iruntime.CPUQuotaStatus, error) {
			return 0, iruntime.CPUQuotaUndefined, errors.New("failed")
		})
		prev := currentMaxProcs()
		undo, err := Set(opt)
		defer undo()
		require.Error(t, err, "Set should have failed")
		assert.Equal(t, "failed", err.Error(), "should pass errors up the stack")
		assert.Equal(t, prev, currentMaxProcs(), "shouldn't alter GOMAXPROCS")
	})

	t.Run("QuotaUndefined", func(t *testing.T) {
		buf, logOpt := testLogger()
		quotaOpt := stubProcs(func(int) (int, iruntime.CPUQuotaStatus, error) {
			return 0, iruntime.CPUQuotaUndefined, nil
		})
		prev := currentMaxProcs()
		undo, err := Set(logOpt, quotaOpt)
		defer undo()
		require.NoError(t, err, "Set failed")
		assert.Equal(t, prev, currentMaxProcs(), "shouldn't alter GOMAXPROCS")
		assert.Contains(t, buf.String(), "quota undefined", "unexpected log output")
	})

	t.Run("QuotaUndefined return maxProcs=7", func(t *testing.T) {
		buf, logOpt := testLogger()
		quotaOpt := stubProcs(func(int) (int, iruntime.CPUQuotaStatus, error) {
			return 7, iruntime.CPUQuotaUndefined, nil
		})
		prev := currentMaxProcs()
		undo, err := Set(logOpt, quotaOpt)
		defer undo()
		require.NoError(t, err, "Set failed")
		assert.Equal(t, prev, currentMaxProcs(), "shouldn't alter GOMAXPROCS")
		assert.Contains(t, buf.String(), "quota undefined", "unexpected log output")
	})

	t.Run("QuotaTooSmall", func(t *testing.T) {
		buf, logOpt := testLogger()
		quotaOpt := stubProcs(func(min int) (int, iruntime.CPUQuotaStatus, error) {
			return min, iruntime.CPUQuotaMinUsed, nil
		})
		undo, err := Set(logOpt, quotaOpt, Min(5))
		defer undo()
		require.NoError(t, err, "Set failed")
		assert.Equal(t, 5, currentMaxProcs(), "should use min allowed GOMAXPROCS")
		assert.Contains(t, buf.String(), "using minimum allowed", "unexpected log output")
	})

	t.Run("Min unused", func(t *testing.T) {
		buf, logOpt := testLogger()
		quotaOpt := stubProcs(func(min int) (int, iruntime.CPUQuotaStatus, error) {
			return min, iruntime.CPUQuotaMinUsed, nil
		})
		// Min(-1) should be ignored.
		undo, err := Set(logOpt, quotaOpt, Min(5), Min(-1))
		defer undo()
		require.NoError(t, err, "Set failed")
		assert.Equal(t, 5, currentMaxProcs(), "should use min allowed GOMAXPROCS")
		assert.Contains(t, buf.String(), "using minimum allowed", "unexpected log output")
	})

	t.Run("QuotaUsed", func(t *testing.T) {
		opt := stubProcs(func(min int) (int, iruntime.CPUQuotaStatus, error) {
			assert.Equal(t, 1, min, "Default minimum value should be 1")
			return 42, iruntime.CPUQuotaUsed, nil
		})
		undo, err := Set(opt)
		defer undo()
		require.NoError(t, err, "Set failed")
		assert.Equal(t, 42, currentMaxProcs(), "should change GOMAXPROCS to match quota")
	})
}

func TestMain(m *testing.M) {
	if err := os.Unsetenv(_maxProcsKey); err != nil {
		log.Fatalf("Couldn't clear %s: %v\n", _maxProcsKey, err)
	}
	os.Exit(m.Run())
}
