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

package maxprocs_test

import (
	"log"

	"go.uber.org/automaxprocs/maxprocs"
)

func Example() {
	undo, err := maxprocs.Set()
	defer undo()
	if err != nil {
		log.Fatalf("failed to set GOMAXPROCS: %v", err)
	}
	// Insert your application logic here.
}

func ExampleLogger() {
	// By default, Set doesn't output any logs. You can enable logging by
	// supplying a printf implementation.
	undo, err := maxprocs.Set(maxprocs.Logger(log.Printf))
	defer undo()
	if err != nil {
		log.Fatalf("failed to set GOMAXPROCS: %v", err)
	}
}
