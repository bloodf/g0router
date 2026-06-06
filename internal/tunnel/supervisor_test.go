package tunnel

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func writeFakeBinary(t *testing.T, dir, name, script string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}
	return path
}

func TestSupervisorStartCapturesURL(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	script := "#!/bin/sh\necho \"tunnel available at https://test-abc.trycloudflare.com\"\nsleep 3600\n"
	binPath := writeFakeBinary(t, t.TempDir(), "fake-cloudflared", script)

	s := &Supervisor{}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.Start(ctx, binPath, []string{}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Stop()

	for i := 0; i < 100; i++ {
		if url := s.URL(); url != "" {
			if url != "https://test-abc.trycloudflare.com" {
				t.Fatalf("url = %q, want https://test-abc.trycloudflare.com", url)
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("timeout waiting for URL")
}

func TestSupervisorStopCleansUp(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	script := "#!/bin/sh\necho \"https://test-abc.trycloudflare.com\"\nsleep 3600\n"
	binPath := writeFakeBinary(t, t.TempDir(), "fake-cloudflared", script)

	s := &Supervisor{}
	ctx := context.Background()

	if err := s.Start(ctx, binPath, []string{}); err != nil {
		t.Fatalf("Start: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	if err := s.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	for i := 0; i < 50; i++ {
		if s.cmd != nil && s.cmd.ProcessState != nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("process did not exit after Stop")
}

func TestSupervisorContextCancelKillsProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	script := "#!/bin/sh\necho \"https://test-abc.trycloudflare.com\"\nsleep 3600\n"
	binPath := writeFakeBinary(t, t.TempDir(), "fake-cloudflared", script)

	s := &Supervisor{}
	ctx, cancel := context.WithCancel(context.Background())

	if err := s.Start(ctx, binPath, []string{}); err != nil {
		t.Fatalf("Start: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-s.done:
		// ok
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for process to exit after context cancel")
	}

	for i := 0; i < 50; i++ {
		if s.cmd != nil && s.cmd.ProcessState != nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("process did not exit after context cancel")
}

func TestSupervisorRequiresAbsolutePath(t *testing.T) {
	s := &Supervisor{}
	ctx := context.Background()

	err := s.Start(ctx, "relative/path/cloudflared", []string{})
	if err == nil {
		t.Fatal("expected error for relative path")
	}
}

func TestSupervisorURLBeforeStart(t *testing.T) {
	s := &Supervisor{}
	if s.URL() != "" {
		t.Fatal("URL should be empty before Start")
	}
}

func TestSupervisorStopWhenNotStarted(t *testing.T) {
	s := &Supervisor{}
	if err := s.Stop(); err != nil {
		t.Fatalf("Stop when not started: %v", err)
	}
}
