package output

import (
	"bytes"
	"io"
	"sync"

	"github.com/fatih/color"
)

// Color palette for task prefixes - distinct, easy to read colors
var colorPalette = []*color.Color{
	color.New(color.FgCyan),
	color.New(color.FgYellow),
	color.New(color.FgGreen),
	color.New(color.FgMagenta),
	color.New(color.FgBlue),
	color.New(color.FgRed),
	color.New(color.FgHiCyan),
	color.New(color.FgHiYellow),
	color.New(color.FgHiGreen),
	color.New(color.FgHiMagenta),
}

// PrefixedWriter wraps an io.Writer and prefixes each line with a colored task identifier.
// It buffers partial lines and only writes complete lines to prevent interleaving.
type PrefixedWriter struct {
	out    io.Writer
	prefix string
	color  *color.Color
	mu     *sync.Mutex // shared mutex for synchronized writes
	buf    bytes.Buffer
}

// NewPrefixedWriter creates a new PrefixedWriter with the given prefix and color.
// The mutex should be shared across all writers in a group to prevent line clobbering.
func NewPrefixedWriter(out io.Writer, prefix string, c *color.Color, mu *sync.Mutex) *PrefixedWriter {
	return &PrefixedWriter{
		out:    out,
		prefix: prefix,
		color:  c,
		mu:     mu,
	}
}

// Write implements io.Writer. It buffers input and writes complete lines with prefix.
func (w *PrefixedWriter) Write(p []byte) (n int, err error) {
	n = len(p) // We always "consume" all bytes from caller's perspective

	w.buf.Write(p)

	// Process complete lines
	for {
		line, err := w.buf.ReadBytes('\n')
		if err != nil {
			// No complete line yet - put back what we read
			w.buf.Write(line)
			break
		}
		// Write the complete line with prefix
		w.writeLine(line)
	}

	return n, nil
}

// writeLine writes a single line with the colored prefix, holding the mutex.
func (w *PrefixedWriter) writeLine(line []byte) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Write colored prefix
	w.color.Fprintf(w.out, "%s | ", w.prefix)
	// Write the line content (includes newline)
	w.out.Write(line)
}

// Flush writes any remaining buffered content (partial line without newline).
func (w *PrefixedWriter) Flush() {
	if w.buf.Len() == 0 {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// Write remaining content with prefix and add newline
	w.color.Fprintf(w.out, "%s | ", w.prefix)
	w.out.Write(w.buf.Bytes())
	w.out.Write([]byte("\n"))
	w.buf.Reset()
}

// WriterGroup manages a set of PrefixedWriters for multiple tasks.
type WriterGroup struct {
	writers map[string]*PrefixedWriter
	mu      *sync.Mutex
}

// NewWriterGroup creates a WriterGroup with PrefixedWriters for each task.
// Colors are assigned from a rotating palette based on task order.
func NewWriterGroup(out io.Writer, taskNames []string) *WriterGroup {
	mu := &sync.Mutex{}
	writers := make(map[string]*PrefixedWriter, len(taskNames))

	// Find max prefix length for alignment
	maxLen := 0
	for _, name := range taskNames {
		if len(name) > maxLen {
			maxLen = len(name)
		}
	}

	for i, name := range taskNames {
		c := colorPalette[i%len(colorPalette)]
		// Pad prefix to align output
		paddedName := padRight(name, maxLen)
		writers[name] = NewPrefixedWriter(out, paddedName, c, mu)
	}

	return &WriterGroup{
		writers: writers,
		mu:      mu,
	}
}

// Get returns the PrefixedWriter for the given task name.
func (g *WriterGroup) Get(name string) *PrefixedWriter {
	return g.writers[name]
}

// FlushAll flushes all writers in the group.
func (g *WriterGroup) FlushAll() {
	for _, w := range g.writers {
		w.Flush()
	}
}

// padRight pads a string with spaces to the specified length.
func padRight(s string, length int) string {
	if len(s) >= length {
		return s
	}
	padding := make([]byte, length-len(s))
	for i := range padding {
		padding[i] = ' '
	}
	return s + string(padding)
}
