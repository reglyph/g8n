package printer

import (
	"bytes"
	"fmt"
	"strings"
)

// Printer buffers generated Go source lines with indentation support.
type Printer struct {
	buf   bytes.Buffer
	depth int
}

// Indent increases the indentation level.
func (p *Printer) Indent() { p.depth++ }

// Dedent decreases the indentation level.
func (p *Printer) Dedent() { p.depth-- }

// Line writes one line at the current indentation level.
func (p *Printer) Line(s string) {
	p.buf.WriteString(strings.Repeat("\t", p.depth))
	p.buf.WriteString(s)
	p.buf.WriteByte('\n')
}

// Linef writes one formatted line at the current indentation level.
func (p *Printer) Linef(f string, args ...any) {
	p.Line(fmt.Sprintf(f, args...))
}

// Blank writes an empty line.
func (p *Printer) Blank() {
	p.buf.WriteByte('\n')
}

// WriteRaw writes text verbatim without indentation.
func (p *Printer) WriteRaw(s string) {
	p.buf.WriteString(s)
}

// Bytes returns the buffered source.
func (p *Printer) Bytes() []byte {
	return p.buf.Bytes()
}
