package output

import (
	"bytes"
	"strings"
	"sync"
	"testing"

	"github.com/fatih/color"
)

func init() {
	// Disable color output in tests for easier string matching
	color.NoColor = true
}

func TestPrefixedWriter_SingleLine(t *testing.T) {
	var buf bytes.Buffer
	mu := &sync.Mutex{}
	c := color.New(color.FgCyan)

	w := NewPrefixedWriter(&buf, "task1", c, mu)
	w.Write([]byte("hello world\n"))

	got := buf.String()
	want := "task1 | hello world\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPrefixedWriter_MultipleLines(t *testing.T) {
	var buf bytes.Buffer
	mu := &sync.Mutex{}
	c := color.New(color.FgCyan)

	w := NewPrefixedWriter(&buf, "task1", c, mu)
	w.Write([]byte("line1\nline2\nline3\n"))

	got := buf.String()
	lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
	if len(lines) != 3 {
		t.Errorf("got %d lines, want 3", len(lines))
	}
	for _, line := range lines {
		if !strings.HasPrefix(line, "task1 | ") {
			t.Errorf("line %q missing prefix", line)
		}
	}
}

func TestPrefixedWriter_PartialLines(t *testing.T) {
	var buf bytes.Buffer
	mu := &sync.Mutex{}
	c := color.New(color.FgCyan)

	w := NewPrefixedWriter(&buf, "task1", c, mu)

	// Write partial line
	w.Write([]byte("hel"))
	if buf.Len() != 0 {
		t.Error("partial line should be buffered, not written")
	}

	// Complete the line
	w.Write([]byte("lo\n"))
	got := buf.String()
	want := "task1 | hello\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPrefixedWriter_Flush(t *testing.T) {
	var buf bytes.Buffer
	mu := &sync.Mutex{}
	c := color.New(color.FgCyan)

	w := NewPrefixedWriter(&buf, "task1", c, mu)

	// Write partial line without newline
	w.Write([]byte("incomplete"))
	if buf.Len() != 0 {
		t.Error("partial line should be buffered")
	}

	// Flush should write it with a newline
	w.Flush()
	got := buf.String()
	want := "task1 | incomplete\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPrefixedWriter_FlushEmpty(t *testing.T) {
	var buf bytes.Buffer
	mu := &sync.Mutex{}
	c := color.New(color.FgCyan)

	w := NewPrefixedWriter(&buf, "task1", c, mu)

	// Flush with nothing buffered should be a no-op
	w.Flush()
	if buf.Len() != 0 {
		t.Error("flush of empty buffer should write nothing")
	}
}

func TestWriterGroup_Creation(t *testing.T) {
	var buf bytes.Buffer
	tasks := []string{"frontend", "backend", "database"}

	group := NewWriterGroup(&buf, tasks)

	for _, task := range tasks {
		w := group.Get(task)
		if w == nil {
			t.Errorf("writer for %q is nil", task)
		}
	}
}

func TestWriterGroup_DifferentPrefixes(t *testing.T) {
	var buf bytes.Buffer
	tasks := []string{"fe", "backend"}

	group := NewWriterGroup(&buf, tasks)

	// Write from each task
	group.Get("fe").Write([]byte("frontend line\n"))
	group.Get("backend").Write([]byte("backend line\n"))

	got := buf.String()

	// Check both prefixes appear
	if !strings.Contains(got, "fe") {
		t.Error("output missing 'fe' prefix")
	}
	if !strings.Contains(got, "backend") {
		t.Error("output missing 'backend' prefix")
	}
}

func TestWriterGroup_PrefixAlignment(t *testing.T) {
	var buf bytes.Buffer
	tasks := []string{"a", "longer"}

	group := NewWriterGroup(&buf, tasks)

	// The shorter prefix should be padded
	w := group.Get("a")
	if w.prefix != "a     " {
		t.Errorf("got prefix %q, want %q", w.prefix, "a     ")
	}
	w = group.Get("longer")
	if w.prefix != "longer" {
		t.Errorf("got prefix %q, want %q", w.prefix, "longer")
	}
}

func TestWriterGroup_FlushAll(t *testing.T) {
	var buf bytes.Buffer
	tasks := []string{"task1", "task2"}

	group := NewWriterGroup(&buf, tasks)

	// Write partial lines
	group.Get("task1").Write([]byte("partial1"))
	group.Get("task2").Write([]byte("partial2"))

	// Nothing written yet
	if buf.Len() != 0 {
		t.Error("partial lines should be buffered")
	}

	// FlushAll should write both
	group.FlushAll()

	got := buf.String()
	if !strings.Contains(got, "partial1") {
		t.Error("output missing partial1")
	}
	if !strings.Contains(got, "partial2") {
		t.Error("output missing partial2")
	}
}

func TestPrefixedWriter_Concurrent(t *testing.T) {
	var buf bytes.Buffer
	mu := &sync.Mutex{}

	// Create multiple writers sharing the same mutex
	w1 := NewPrefixedWriter(&buf, "task1", color.New(color.FgCyan), mu)
	w2 := NewPrefixedWriter(&buf, "task2", color.New(color.FgYellow), mu)

	// Concurrent writes
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			w1.Write([]byte("line from task1\n"))
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			w2.Write([]byte("line from task2\n"))
		}
	}()

	wg.Wait()

	// Count lines - should be exactly 200
	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	if len(lines) != 200 {
		t.Errorf("got %d lines, want 200", len(lines))
	}

	// Each line should have exactly one prefix
	for i, line := range lines {
		hasTask1 := strings.HasPrefix(line, "task1 | ")
		hasTask2 := strings.HasPrefix(line, "task2 | ")
		if !hasTask1 && !hasTask2 {
			t.Errorf("line %d missing prefix: %q", i, line)
		}
		if hasTask1 && hasTask2 {
			t.Errorf("line %d has multiple prefixes: %q", i, line)
		}
	}
}
