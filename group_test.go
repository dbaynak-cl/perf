// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package perf

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func TestGroup(t *testing.T) {
	t.Run("Count", testGroupCount)
	t.Run("Record", testGroupRecord)
}

func testGroupCount(t *testing.T) {
	requires(t, paranoid(1), hardwarePMU, softwarePMU)

	da := new(Attr)
	Dummy.Configure(da)

	g := Group{
		CountFormat: CountFormat{
			Enabled: true,
			Running: true,
		},
	}
	g.Add(CPUCycles, Instructions, da)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	ev, err := g.Open(CallingThread, AnyCPU)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	sum := int64(0)
	gc, err := ev.MeasureGroup(func() {
		for i := int64(0); i < 50000; i++ {
			sum += i
		}
	})
	if err != nil {
		t.Fatalf("MeasureGroup: %v", err)
	}

	t.Logf("got sum %d in %d %s and %d %s", sum, gc.Values[0].Value, gc.Values[0].Label, gc.Values[1].Value, gc.Values[1].Label)
}

func testGroupRecord(t *testing.T) {
	requires(t, tracepointPMU, debugfs) // TODO(acln): paranoid

	ga := &Attr{
		Options: Options{
			Disabled: true,
		},
		SampleFormat: SampleFormat{
			Tid:      true,
			Time:     true,
			CPU:      true,
			IP:       true,
			StreamID: true,
		},
	}
	ga.SetSamplePeriod(1)
	ga.SetWakeupEvents(1)
	gtp := Tracepoint("syscalls", "sys_enter_getpid")
	if err := gtp.Configure(ga); err != nil {
		t.Fatal(err)
	}

	wa := &Attr{
		SampleFormat: SampleFormat{
			Tid:      true,
			Time:     true,
			CPU:      true,
			IP:       true,
			StreamID: true,
		},
	}
	wa.SetSamplePeriod(1)
	wa.SetWakeupEvents(1)
	wtp := Tracepoint("syscalls", "sys_enter_write")
	if err := wtp.Configure(wa); err != nil {
		t.Fatal(err)
	}

	g := Group{
		CountFormat: CountFormat{
			Enabled: true,
			Running: true,
		},
	}
	g.Add(ga, wa)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	ev, err := g.Open(CallingThread, AnyCPU)
	if err != nil {
		t.Fatal(err)
	}
	defer ev.Close()

	gc, err := ev.MeasureGroup(func() {
		getpidTrigger()
		writeTrigger()
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, got := range gc.Values {
		if got.Value != 1 {
			t.Fatalf("want 1 hit for %q, got %d", got.Label, got.Value)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	grec, err := ev.ReadRecord(ctx)
	if err != nil {
		t.Fatal(err)
	}
	gsr, ok := grec.(*SampleGroupRecord)
	if !ok {
		t.Fatalf("got %T, want *SampleGroupRecord", grec)
	}

	wrec, err := ev.ReadRecord(ctx)
	if err != nil {
		t.Fatal(err)
	}
	wsr, ok := wrec.(*SampleGroupRecord)
	if !ok {
		t.Fatalf("got %T, want *SampleGroupRecord", wrec)
	}

	if gip, wip := gsr.IP, wsr.IP; gip == wip {
		t.Fatalf("equal IP 0x%x for samples of different events", wip)
	}
}
