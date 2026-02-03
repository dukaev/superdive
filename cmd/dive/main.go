package main

// Copyright © 2018 Alex Goodman
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

import (
	"fmt"
	"os"

	runtimepprof "runtime/pprof"

	"github.com/anchore/clio"
	"github.com/wagoodman/dive/cmd/dive/cli"
)

// applicationName is the non-capitalized name of the application (do not change this)
const (
	applicationName = "dive"
	notProvided     = "[not provided]"
)

// TODO: these need to be wired up to the build flags
// all variables here are provided as build-time arguments, with clear default values
var (
	version        = notProvided
	buildDate      = notProvided
	gitCommit      = notProvided
	gitDescription = notProvided
)

func main() {
	cpuProfile := os.Getenv("CPU_PROFILE")
	memProfile := os.Getenv("MEM_PROFILE")
	goroutineProfile := os.Getenv("GOROUTINE_PROFILE")

	// Start CPU profiling if requested
	if cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			panic(fmt.Sprintf("could not create CPU profile: %v", err))
		}
		if err := runtimepprof.StartCPUProfile(f); err != nil {
			panic(fmt.Sprintf("could not start CPU profile: %v", err))
		}
		defer func() {
			runtimepprof.StopCPUProfile()
			f.Close()
		}()
	}

	app := cli.Application(
		clio.Identification{
			Name:           applicationName,
			Version:        version,
			BuildDate:      buildDate,
			GitCommit:      gitCommit,
			GitDescription: gitDescription,
		},
	)

	app.Run()

	// Write memory profile after execution if requested
	if memProfile != "" {
		f, err := os.Create(memProfile)
		if err != nil {
			panic(fmt.Sprintf("could not create memory profile: %v", err))
		}
		defer f.Close()
		runtimepprof.WriteHeapProfile(f)
	}

	// Write goroutine profile after execution if requested
	if goroutineProfile != "" {
		f, err := os.Create(goroutineProfile)
		if err != nil {
			panic(fmt.Sprintf("could not create goroutine profile: %v", err))
		}
		defer f.Close()
		g := runtimepprof.Lookup("goroutine")
		if g != nil {
			g.WriteTo(f, 0)
		}
	}
}
