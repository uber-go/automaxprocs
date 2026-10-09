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
	"math"
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

type logEntry struct {
	level  string
	msg    string
	fields map[string]interface{}
}

type testStructuredLogger struct {
	entries []logEntry
}

func (l *testStructuredLogger) Info(msg string, fields map[string]interface{}) {
	l.entries = append(l.entries, logEntry{level: "info", msg: msg, fields: fields})
}

func (l *testStructuredLogger) Error(msg string, fields map[string]interface{}) {
	l.entries = append(l.entries, logEntry{level: "error", msg: msg, fields: fields})
}

func testStructuredLoggerOption() (*testStructuredLogger, Option) {
	sl := &testStructuredLogger{}
	return sl, StructuredLogger(sl)
}

func stubProcs(f func(int, func(v float64) int) (int, iruntime.CPUQuotaStatus, error)) Option {
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

	t.Run("formatted output with previous value", func(t *testing.T) {
		buf, logOpt := testLogger()
		quotaOpt := stubProcs(func(min int, round func(v float64) int) (int, iruntime.CPUQuotaStatus, error) {
			return 16, iruntime.CPUQuotaUsed, nil
		})
		prev := currentMaxProcs()
		undo, err := Set(logOpt, quotaOpt)
		defer undo()
		require.NoError(t, err, "Set failed")
		expected := fmt.Sprintf("maxprocs: Updating GOMAXPROCS=16 (was %d): determined from CPU quota", prev)
		assert.Contains(t, buf.String(), expected)
	})

	t.Run("formatted output min quota with previous value", func(t *testing.T) {
		buf, logOpt := testLogger()
		quotaOpt := stubProcs(func(min int, round func(v float64) int) (int, iruntime.CPUQuotaStatus, error) {
			return min, iruntime.CPUQuotaMinUsed, nil
		})
		prev := currentMaxProcs()
		undo, err := Set(logOpt, quotaOpt, Min(4))
		defer undo()
		require.NoError(t, err, "Set failed")
		expected := fmt.Sprintf("maxprocs: Updating GOMAXPROCS=4 (was %d): using minimum allowed GOMAXPROCS", prev)
		assert.Contains(t, buf.String(), expected)
	})
}

func TestStructuredLogger(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		sl, opt := testStructuredLoggerOption()
		undo, err := Set(opt)
		defer undo()
		require.NoError(t, err, "Set failed")
		assert.NotEmpty(t, sl.entries, "expected structured log entries")
	})

	t.Run("StructuredLogFunc", func(t *testing.T) {
		var entries []logEntry
		fn := StructuredLogFunc(func(msg string, fields map[string]interface{}) {
			entries = append(entries, logEntry{msg: msg, fields: fields})
		})
		undo, err := Set(StructuredLogger(fn))
		defer undo()
		require.NoError(t, err, "Set failed")
		assert.NotEmpty(t, entries, "expected structured log entries from StructuredLogFunc")
	})

	t.Run("EnvVarPresent", func(t *testing.T) {
		withMax(t, 42, func() {
			sl, opt := testStructuredLoggerOption()
			undo, err := Set(opt)
			defer undo()
			require.NoError(t, err, "Set failed")
			require.NotEmpty(t, sl.entries)
			assert.Equal(t, "info", sl.entries[0].level)
			assert.Equal(t, "maxprocs: Honoring GOMAXPROCS as set in environment", sl.entries[0].msg)
			assert.Equal(t, "42", sl.entries[0].fields["gomaxprocs"])
			assert.Equal(t, "environment", sl.entries[0].fields["source"])
		})
	})

	t.Run("ErrorReadingQuota", func(t *testing.T) {
		sl, logOpt := testStructuredLoggerOption()
		expectedErr := errors.New("read error")
		quotaOpt := stubProcs(func(int, func(v float64) int) (int, iruntime.CPUQuotaStatus, error) {
			return 0, iruntime.CPUQuotaUndefined, expectedErr
		})
		undo, err := Set(logOpt, quotaOpt)
		defer undo()
		require.Error(t, err, "Set should fail on error")
		require.NotEmpty(t, sl.entries)
		assert.Equal(t, "error", sl.entries[0].level)
		assert.Equal(t, "maxprocs: Failed to check CPU quota", sl.entries[0].msg)
		assert.Equal(t, expectedErr, sl.entries[0].fields["error"])
	})

	t.Run("QuotaUndefined", func(t *testing.T) {
		sl, logOpt := testStructuredLoggerOption()
		quotaOpt := stubProcs(func(int, func(v float64) int) (int, iruntime.CPUQuotaStatus, error) {
			return 0, iruntime.CPUQuotaUndefined, nil
		})
		undo, err := Set(logOpt, quotaOpt)
		defer undo()
		require.NoError(t, err, "Set failed")
		require.NotEmpty(t, sl.entries)
		assert.Equal(t, "info", sl.entries[0].level)
		assert.Equal(t, "maxprocs: Leaving GOMAXPROCS unchanged: CPU quota undefined", sl.entries[0].msg)
		assert.Equal(t, currentMaxProcs(), sl.entries[0].fields["gomaxprocs"])
		assert.Equal(t, "CPU quota undefined", sl.entries[0].fields["reason"])
	})

	t.Run("QuotaMinUsed", func(t *testing.T) {
		sl, logOpt := testStructuredLoggerOption()
		quotaOpt := stubProcs(func(min int, round func(v float64) int) (int, iruntime.CPUQuotaStatus, error) {
			return min, iruntime.CPUQuotaMinUsed, nil
		})
		prev := currentMaxProcs()
		undo, err := Set(logOpt, quotaOpt, Min(4))
		defer undo()
		require.NoError(t, err, "Set failed")
		require.NotEmpty(t, sl.entries)
		assert.Equal(t, "info", sl.entries[0].level)
		assert.Equal(t, "maxprocs: Updating GOMAXPROCS: using minimum allowed GOMAXPROCS", sl.entries[0].msg)
		assert.Equal(t, 4, sl.entries[0].fields["gomaxprocs"])
		assert.Equal(t, prev, sl.entries[0].fields["previous"])
		assert.Equal(t, "using minimum allowed GOMAXPROCS", sl.entries[0].fields["reason"])
	})

	t.Run("QuotaUsed", func(t *testing.T) {
		sl, logOpt := testStructuredLoggerOption()
		quotaOpt := stubProcs(func(min int, round func(v float64) int) (int, iruntime.CPUQuotaStatus, error) {
			return 8, iruntime.CPUQuotaUsed, nil
		})
		prev := currentMaxProcs()
		undo, err := Set(logOpt, quotaOpt)
		defer undo()
		require.NoError(t, err, "Set failed")
		require.NotEmpty(t, sl.entries)
		assert.Equal(t, "info", sl.entries[0].level)
		assert.Equal(t, "maxprocs: Updating GOMAXPROCS: determined from CPU quota", sl.entries[0].msg)
		assert.Equal(t, 8, sl.entries[0].fields["gomaxprocs"])
		assert.Equal(t, prev, sl.entries[0].fields["previous"])
		assert.Equal(t, "determined from CPU quota", sl.entries[0].fields["reason"])
	})

	t.Run("Undo", func(t *testing.T) {
		sl, logOpt := testStructuredLoggerOption()
		quotaOpt := stubProcs(func(min int, round func(v float64) int) (int, iruntime.CPUQuotaStatus, error) {
			return 8, iruntime.CPUQuotaUsed, nil
		})
		prev := currentMaxProcs()
		undo, err := Set(logOpt, quotaOpt)
		require.NoError(t, err, "Set failed")
		sl.entries = nil
		undo()
		require.NotEmpty(t, sl.entries)
		assert.Equal(t, "info", sl.entries[0].level)
		assert.Equal(t, "maxprocs: Resetting GOMAXPROCS", sl.entries[0].msg)
		assert.Equal(t, prev, sl.entries[0].fields["gomaxprocs"])
	})

	t.Run("UndoNoop", func(t *testing.T) {
		sl, logOpt := testStructuredLoggerOption()
		quotaOpt := stubProcs(func(min int, round func(v float64) int) (int, iruntime.CPUQuotaStatus, error) {
			return 0, iruntime.CPUQuotaUndefined, nil
		})
		undo, err := Set(logOpt, quotaOpt)
		require.NoError(t, err, "Set failed")
		sl.entries = nil
		undo()
		require.NotEmpty(t, sl.entries)
		assert.Equal(t, "info", sl.entries[0].level)
		assert.Equal(t, "maxprocs: No GOMAXPROCS change to reset", sl.entries[0].msg)
	})

	t.Run("BothLoggers", func(t *testing.T) {
		buf, printfOpt := testLogger()
		sl, structOpt := testStructuredLoggerOption()
		quotaOpt := stubProcs(func(min int, round func(v float64) int) (int, iruntime.CPUQuotaStatus, error) {
			return 6, iruntime.CPUQuotaUsed, nil
		})
		undo, err := Set(printfOpt, structOpt, quotaOpt)
		defer undo()
		require.NoError(t, err, "Set failed")
		assert.Contains(t, buf.String(), "Updating GOMAXPROCS=6")
		require.NotEmpty(t, sl.entries)
		assert.Equal(t, 6, sl.entries[0].fields["gomaxprocs"])
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
		opt := stubProcs(func(int, func(v float64) int) (int, iruntime.CPUQuotaStatus, error) {
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
		quotaOpt := stubProcs(func(int, func(v float64) int) (int, iruntime.CPUQuotaStatus, error) {
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
		quotaOpt := stubProcs(func(int, func(v float64) int) (int, iruntime.CPUQuotaStatus, error) {
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
		quotaOpt := stubProcs(func(min int, round func(v float64) int) (int, iruntime.CPUQuotaStatus, error) {
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
		quotaOpt := stubProcs(func(min int, round func(v float64) int) (int, iruntime.CPUQuotaStatus, error) {
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
		opt := stubProcs(func(min int, round func(v float64) int) (int, iruntime.CPUQuotaStatus, error) {
			assert.Equal(t, 1, min, "Default minimum value should be 1")
			return 42, iruntime.CPUQuotaUsed, nil
		})
		undo, err := Set(opt)
		defer undo()
		require.NoError(t, err, "Set failed")
		assert.Equal(t, 42, currentMaxProcs(), "should change GOMAXPROCS to match quota")
	})

	t.Run("RoundQuotaSetToCeil", func(t *testing.T) {
		opt := stubProcs(func(min int, round func(v float64) int) (int, iruntime.CPUQuotaStatus, error) {
			assert.Equal(t, round(2.4), 3, "round should be math.Ceil")
			return 43, iruntime.CPUQuotaUsed, nil
		})
		undo, err := Set(opt, RoundQuotaFunc(func(v float64) int { return int(math.Ceil(v)) }))
		defer undo()
		require.NoError(t, err, "Set failed")
		assert.Equal(t, 43, currentMaxProcs(), "should change GOMAXPROCS to match rounded up quota")
	})

	t.Run("RoundQuotaSetToFloor", func(t *testing.T) {
		opt := stubProcs(func(min int, round func(v float64) int) (int, iruntime.CPUQuotaStatus, error) {
			assert.Equal(t, round(2.6), 2, "round should be math.Floor")
			return 42, iruntime.CPUQuotaUsed, nil
		})
		undo, err := Set(opt, RoundQuotaFunc(func(v float64) int { return int(math.Floor(v)) }))
		defer undo()
		require.NoError(t, err, "Set failed")
		assert.Equal(t, 42, currentMaxProcs(), "should change GOMAXPROCS to match rounded up quota")
	})
}

func TestMain(m *testing.M) {
	if err := os.Unsetenv(_maxProcsKey); err != nil {
		log.Fatalf("Couldn't clear %s: %v\n", _maxProcsKey, err)
	}
	os.Exit(m.Run())
}
