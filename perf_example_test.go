// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package perf

import (
	"context"
	"fmt"
	"log"
	"runtime"

	"golang.org/x/sys/unix"
)

func ExampleHardwareCounter_iPC() {
	g := Group{
		CountFormat: CountFormat{
			Running: true,
		},
	}
	g.Add(Instructions, CPUCycles)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	ipc, err := g.Open(CallingThread, AnyCPU)
	if err != nil {
		log.Fatal(err)
	}
	defer ipc.Close()

	sum := 0
	gc, err := ipc.MeasureGroup(func() {
		for i := 0; i < 100000; i++ {
			sum += i
		}
	})
	if err != nil {
		log.Fatal(err)
	}

	insns, cycles := gc.Values[0].Value, gc.Values[1].Value

	fmt.Printf("got sum = %d in %v: %d instructions, %d CPU cycles: %f IPC",
		sum, gc.Running, insns, cycles, float64(insns)/float64(cycles))
}

func ExampleSoftwareCounter_pageFaults() {
	pfa := new(Attr)
	PageFaults.Configure(pfa)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	faults, err := Open(pfa, CallingThread, AnyCPU, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer faults.Close()

	var mem []byte
	const (
		size = 64 * 1024 * 1024
		pos  = 63 * 1024 * 1024
	)
	c, err := faults.Measure(func() {
		mem = make([]byte, size)
		mem[pos] = 42
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("saw %d page faults, wrote value %d", c.Value, mem[pos])
}

func ExampleTracepoint_getpid() {
	ga := new(Attr)
	gtp := Tracepoint("syscalls", "sys_enter_getpid")
	if err := gtp.Configure(ga); err != nil {
		log.Fatal(err)
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	getpid, err := Open(ga, CallingThread, AnyCPU, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer getpid.Close()

	unix.Getpid() // does not count towards the measurement

	c, err := getpid.Measure(func() {
		unix.Getpid()
		unix.Getpid()
		unix.Getpid()
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("saw getpid %d times\n", c.Value) // should print 3
}

func ExampleMmapRecord_plugin() {
	var targetpid int // pid of the monitored process

	da := &Attr{
		Options: Options{
			Mmap: true,
		},
	}
	da.SetSamplePeriod(1)
	da.SetWakeupEvents(1)
	Dummy.Configure(da) // configure a dummy event, so we can Open it

	mmap, err := Open(da, targetpid, AnyCPU, nil)
	if err != nil {
		log.Fatal(err)
	}
	if err := mmap.MapRing(); err != nil {
		log.Fatal(err)
	}

	// Monitor the target process, wait for it to load something like
	// a plugin, or a shared library, which requires a PROT_EXEC mapping.

	for {
		rec, err := mmap.ReadRecord(context.Background())
		if err != nil {
			log.Fatal(err)
		}
		mr, ok := rec.(*MmapRecord)
		if !ok {
			continue
		}
		fmt.Printf("pid %d created a PROT_EXEC mapping at %#x: %s",
			mr.Pid, mr.Addr, mr.Filename)
	}
}
