package perf

import (
	"os/exec"
	"testing"
)

func TestCommand(t *testing.T) {
	requires(t, paranoid(2), hardwarePMU)

	cmd := exec.Command("echo", "hello world")

	fa := &Attr{
		CountFormat: CountFormat{
			Running: true,
			ID:      true,
		},
	}
	Instructions.Configure(fa)
	fa.Options.ExcludeKernel = true
	fa.Options.ExcludeHypervisor = true

	count, err := Command(fa, cmd, AnyCPU, nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("count = %v", count.Value)

	// A primitive test to ensure the counter measured something, since we
	// don't know the "correct" value.
	if count.Value < 1000 {
		t.Fatal("counter read less than 1000 - should be > 1M")
	}
}

func TestCommandGroup(t *testing.T) {
	requires(t, paranoid(2), hardwarePMU)

	cmd := exec.Command("echo", "hello world")

	var g Group
	g.CountFormat = CountFormat{
		Running: true,
		ID:      true,
	}
	g.Options.ExcludeKernel = true
	g.Options.ExcludeHypervisor = true
	g.Add(Instructions, CPUCycles)

	counts, err := g.Command(cmd, AnyCPU)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("counts:", counts)

	// A primitive test to ensure the counter measured something, since we
	// don't know the "correct" value.
	if counts.Values[0].Value < 1000 || counts.Values[1].Value < 1000 {
		t.Fatal("counter read less than 1000 - should be > 1M")
	}
}
