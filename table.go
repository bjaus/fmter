package fmter

import (
	"fmt"
	"io"
	"strings"
)

type borderChars struct {
	topLeft, topRight, bottomLeft, bottomRight string
	horizontal, vertical                       string
	topTee, bottomTee, leftTee, rightTee       string
	cross                                      string
}

var borderSets = map[BorderStyle]borderChars{
	BorderRounded: {
		topLeft: "╭", topRight: "╮", bottomLeft: "╰", bottomRight: "╯",
		horizontal: "─", vertical: "│",
		topTee: "┬", bottomTee: "┴", leftTee: "├", rightTee: "┤",
		cross: "┼",
	},
	BorderASCII: {
		topLeft: "+", topRight: "+", bottomLeft: "+", bottomRight: "+",
		horizontal: "-", vertical: "|",
		topTee: "+", bottomTee: "+", leftTee: "+", rightTee: "+",
		cross: "+",
	},
	BorderHeavy: {
		topLeft: "┏", topRight: "┓", bottomLeft: "┗", bottomRight: "┛",
		horizontal: "━", vertical: "┃",
		topTee: "┳", bottomTee: "┻", leftTee: "┣", rightTee: "┫",
		cross: "╋",
	},
	BorderDouble: {
		topLeft: "╔", topRight: "╗", bottomLeft: "╚", bottomRight: "╝",
		horizontal: "═", vertical: "║",
		topTee: "╦", bottomTee: "╩", leftTee: "╠", rightTee: "╣",
		cross: "╬",
	},
}

func writeTable[T any](w io.Writer, items []T) error {
	if len(items) == 0 {
		return nil
	}
	first := any(items[0])
	if _, ok := first.(Rower); !ok {
		return fmt.Errorf("%w: format %q requires Rower, not implemented by %T", ErrMissingInterface, Table, items[0])
	}

	rows := make([][]string, len(items))
	for i, item := range items {
		rows[i] = any(item).(Rower).Row()
	}

	var header []string
	if h, ok := first.(Headed); ok {
		header = h.Header()
	}

	var title string
	if t, ok := first.(Titled); ok {
		title = t.Title()
	}

	border := BorderRounded
	if b, ok := first.(Bordered); ok {
		border = b.Border()
	}

	var aligns []Alignment
	if a, ok := first.(Aligned); ok {
		aligns = a.Alignments()
	}

	var footer []string
	if f, ok := first.(Footered); ok {
		footer = f.Footer()
	}

	var numHdr string
	numbered := false
	if n, ok := first.(Numbered); ok {
		numbered = true
		numHdr = n.NumberHeader()
	}

	var caption string
	if c, ok := first.(Captioned); ok {
		caption = c.Caption()
	}

	// Apply row numbering by prepending a column.
	if numbered {
		if len(header) > 0 {
			header = append([]string{numHdr}, header...)
		}
		for i, row := range rows {
			rows[i] = append([]string{fmt.Sprintf("%d", i+1)}, row...)
		}
		if len(footer) > 0 {
			footer = append([]string{""}, footer...)
		}
		aligns = append([]Alignment{AlignRight}, aligns...)
	}

	numCols := colCount(header, rows, footer)
	widths := computeWidths(numCols, header, rows, footer)

	// Apply max column widths for truncation.
	if tr, ok := first.(Truncated); ok {
		for i, max := range tr.MaxWidths() {
			if i < numCols && max > 0 && widths[i] > max {
				widths[i] = max
			}
		}
	}

	aligns = extendAligns(aligns, numCols)

	var err error
	if border == BorderNone {
		err = renderPlainTable(w, header, rows, footer, widths, aligns)
	} else {
		err = renderBorderedTable(w, title, header, rows, footer, widths, aligns, border)
	}
	if err != nil {
		return err
	}

	if caption != "" {
		if _, err := fmt.Fprintln(w, caption); err != nil {
			return err
		}
	}
	return nil
}

func colCount(header []string, rows [][]string, footer []string) int {
	n := len(header)
	for _, row := range rows {
		if len(row) > n {
			n = len(row)
		}
	}
	if len(footer) > n {
		n = len(footer)
	}
	return n
}

func computeWidths(numCols int, header []string, rows [][]string, footer []string) []int {
	widths := make([]int, numCols)
	for i, h := range header {
		if len(h) > widths[i] {
			widths[i] = len(h)
		}
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < numCols && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}
	for i, cell := range footer {
		if i < numCols && len(cell) > widths[i] {
			widths[i] = len(cell)
		}
	}
	return widths
}

func extendAligns(aligns []Alignment, numCols int) []Alignment {
	if len(aligns) >= numCols {
		return aligns[:numCols]
	}
	extended := make([]Alignment, numCols)
	copy(extended, aligns)
	return extended
}

// --- Plain table (BorderNone) ---

func renderPlainTable(w io.Writer, header []string, rows [][]string, footer []string, widths []int, aligns []Alignment) error {
	if len(header) > 0 {
		if err := writePlainRow(w, header, widths, aligns); err != nil {
			return err
		}
		if err := writePlainSep(w, widths); err != nil {
			return err
		}
	}
	for _, row := range rows {
		if err := writePlainRow(w, row, widths, aligns); err != nil {
			return err
		}
	}
	if len(footer) > 0 {
		if err := writePlainSep(w, widths); err != nil {
			return err
		}
		if err := writePlainRow(w, footer, widths, aligns); err != nil {
			return err
		}
	}
	return nil
}

func writePlainSep(w io.Writer, widths []int) error {
	sep := make([]string, len(widths))
	for i, width := range widths {
		sep[i] = strings.Repeat("-", width)
	}
	_, err := fmt.Fprintln(w, strings.Join(sep, "  "))
	return err
}

func writePlainRow(w io.Writer, cells []string, widths []int, aligns []Alignment) error {
	parts := make([]string, len(widths))
	for i, width := range widths {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		parts[i] = formatTableCell(cell, width, aligns[i])
	}
	line := strings.TrimRight(strings.Join(parts, "  "), " ")
	_, err := fmt.Fprintln(w, line)
	return err
}

// --- Bordered table ---

func renderBorderedTable(w io.Writer, title string, header []string, rows [][]string, footer []string, widths []int, aligns []Alignment, style BorderStyle) error {
	bc := borderSets[style]

	if title != "" {
		// Full-width top border (no column separators).
		if err := drawHLine(w, widths, bc.topLeft, bc.horizontal, bc.horizontal, bc.topRight); err != nil {
			return err
		}
		inner := tableInnerWidth(widths) - 2 // subtract 1-space padding on each side
		padded := alignCell(title, inner, AlignCenter)
		if _, err := fmt.Fprintf(w, "%s %s %s\n", bc.vertical, padded, bc.vertical); err != nil {
			return err
		}
		// Transition to columns.
		if err := drawHLine(w, widths, bc.leftTee, bc.horizontal, bc.topTee, bc.rightTee); err != nil {
			return err
		}
	} else {
		if err := drawHLine(w, widths, bc.topLeft, bc.horizontal, bc.topTee, bc.topRight); err != nil {
			return err
		}
	}

	if len(header) > 0 {
		if err := drawBorderedRow(w, header, widths, aligns, bc.vertical); err != nil {
			return err
		}
		if err := drawHLine(w, widths, bc.leftTee, bc.horizontal, bc.cross, bc.rightTee); err != nil {
			return err
		}
	}

	for _, row := range rows {
		if err := drawBorderedRow(w, row, widths, aligns, bc.vertical); err != nil {
			return err
		}
	}

	if len(footer) > 0 {
		if err := drawHLine(w, widths, bc.leftTee, bc.horizontal, bc.cross, bc.rightTee); err != nil {
			return err
		}
		if err := drawBorderedRow(w, footer, widths, aligns, bc.vertical); err != nil {
			return err
		}
	}

	return drawHLine(w, widths, bc.bottomLeft, bc.horizontal, bc.bottomTee, bc.bottomRight)
}

// tableInnerWidth returns the total character width between the outer vertical
// borders of a bordered table. Each cell contributes its width plus 2 (one
// space of padding on each side), and cells are separated by a single vertical
// border character.
func tableInnerWidth(widths []int) int {
	n := 0
	for _, w := range widths {
		n += w + 2
	}
	if len(widths) > 1 {
		n += len(widths) - 1
	}
	return n
}

func drawHLine(w io.Writer, widths []int, left, fill, mid, right string) error {
	var sb strings.Builder
	sb.WriteString(left)
	for i, width := range widths {
		sb.WriteString(strings.Repeat(fill, width+2))
		if i < len(widths)-1 {
			sb.WriteString(mid)
		}
	}
	sb.WriteString(right)
	_, err := fmt.Fprintln(w, sb.String())
	return err
}

func drawBorderedRow(w io.Writer, cells []string, widths []int, aligns []Alignment, vert string) error {
	var sb strings.Builder
	sb.WriteString(vert)
	for i, width := range widths {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		sb.WriteString(" ")
		sb.WriteString(formatTableCell(cell, width, aligns[i]))
		sb.WriteString(" ")
		if i < len(widths)-1 {
			sb.WriteString(vert)
		}
	}
	sb.WriteString(vert)
	_, err := fmt.Fprintln(w, sb.String())
	return err
}

func formatTableCell(s string, width int, align Alignment) string {
	if width > 0 && len(s) > width {
		if width <= 3 {
			s = s[:width]
		} else {
			s = s[:width-3] + "..."
		}
	}
	return alignCell(s, width, align)
}

func alignCell(s string, width int, align Alignment) string {
	pad := width - len(s)
	if pad <= 0 {
		return s
	}
	switch align {
	case AlignRight:
		return strings.Repeat(" ", pad) + s
	case AlignCenter:
		left := pad / 2
		right := pad - left
		return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
	default:
		return s + strings.Repeat(" ", pad)
	}
}
