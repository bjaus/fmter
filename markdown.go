package fmter

import (
	"fmt"
	"io"
	"strings"

	"github.com/mattn/go-runewidth"
)

func writeMarkdown[T any](w io.Writer, items []T) error {
	if len(items) == 0 {
		return nil
	}
	first := any(items[0])
	if _, ok := first.(Rower); !ok {
		return fmt.Errorf("%w: format %q requires Rower, not implemented by %T", ErrMissingInterface, Markdown, items[0])
	}
	h, ok := first.(Headed)
	if !ok {
		return fmt.Errorf("%w: format %q requires Headed, not implemented by %T", ErrMissingInterface, Markdown, items[0])
	}

	header := h.Header()
	numCols := len(header)

	rows := make([][]string, len(items))
	for i, item := range items {
		rows[i] = any(item).(Rower).Row()
	}

	// Calculate column widths (minimum 3 for alignment markers).
	widths := make([]int, numCols)
	for i, col := range header {
		if w := runewidth.StringWidth(col); w > widths[i] {
			widths[i] = w
		}
	}
	for _, row := range rows {
		for i, cell := range row {
			if w := runewidth.StringWidth(cell); i < numCols && w > widths[i] {
				widths[i] = w
			}
		}
	}
	for i := range widths {
		if widths[i] < 3 {
			widths[i] = 3
		}
	}

	var aligns []Alignment
	if a, ok := first.(Aligned); ok {
		aligns = a.Alignments()
	}
	aligns = extendAligns(aligns, numCols)

	if err := writeMarkdownRow(w, header, widths, aligns); err != nil {
		return err
	}

	sep := make([]string, numCols)
	for i, width := range widths {
		switch aligns[i] {
		case AlignRight:
			sep[i] = strings.Repeat("-", width-1) + ":"
		case AlignCenter:
			sep[i] = ":" + strings.Repeat("-", width-2) + ":"
		default:
			sep[i] = strings.Repeat("-", width)
		}
	}
	if _, err := fmt.Fprintf(w, "| %s |\n", strings.Join(sep, " | ")); err != nil {
		return err
	}

	for _, row := range rows {
		if err := writeMarkdownRow(w, row, widths, aligns); err != nil {
			return err
		}
	}
	return nil
}

func writeMarkdownRow(w io.Writer, cells []string, widths []int, aligns []Alignment) error {
	padded := make([]string, len(widths))
	for i, width := range widths {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		padded[i] = alignCell(cell, width, aligns[i])
	}
	_, err := fmt.Fprintf(w, "| %s |\n", strings.Join(padded, " | "))
	return err
}
