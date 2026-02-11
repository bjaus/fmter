package fmter

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var errInternalWrite = errors.New("write failed")

func TestWrapCellWideCharSafety(t *testing.T) {
	t.Parallel()
	// "你" is a full-width character (2 columns). With width=1, Truncate
	// returns "" because the char doesn't fit. The safety branch advances
	// one rune to avoid an infinite loop.
	lines := wrapCell("你好", 1)
	assert.Equal(t, []string{"你", "好"}, lines)
}

func TestWrapCellNoWrap(t *testing.T) {
	t.Parallel()
	lines := wrapCell("hi", 0)
	assert.Equal(t, []string{"hi"}, lines)
}

func TestWrapCellFits(t *testing.T) {
	t.Parallel()
	lines := wrapCell("hi", 5)
	assert.Equal(t, []string{"hi"}, lines)
}

func TestWrapCellBasic(t *testing.T) {
	t.Parallel()
	lines := wrapCell("Hello", 3)
	assert.Equal(t, []string{"Hel", "lo"}, lines)
}

func TestExtendStylesNoop(t *testing.T) {
	t.Parallel()
	fn := func(s string) string { return s }
	styles := extendStyles([]func(string) string{fn, fn, fn}, 2)
	assert.Len(t, styles, 2)
}

func TestWriteCSVRowSuccess(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := writeCSVRow(&buf, []string{"a", "b"})
	assert.NoError(t, err)
	assert.Equal(t, "a,b\n", buf.String())
}

func TestWriteCSVRowError(t *testing.T) {
	t.Parallel()
	w := &errWriterInternal{}
	// Small data: flush error hit via cw.Error().
	err := writeCSVRow(w, []string{"a", "b"})
	assert.Error(t, err)
}

func TestWriteCSVRowLargeDataError(t *testing.T) {
	t.Parallel()
	w := &errWriterInternal{}
	// Large data exceeds bufio buffer (4096 bytes), causing cw.Write to fail.
	big := strings.Repeat("x", 5000)
	err := writeCSVRow(w, []string{big})
	assert.Error(t, err)
}

type errWriterInternal struct{}

func (e *errWriterInternal) Write([]byte) (int, error) {
	return 0, errInternalWrite
}
