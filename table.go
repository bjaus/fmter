package fmter

import (
	"fmt"
	"io"
	"strings"

	"github.com/mattn/go-runewidth"
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

	var styles []func(string) string
	if s, ok := first.(Styled); ok {
		styles = s.Styles()
	}

	var groups []string
	if _, ok := first.(Grouped); ok {
		groups = make([]string, len(items))
		for i, item := range items {
			groups[i] = any(item).(Grouped).Group()
		}
	}

	var wrapWidths []int
	if wr, ok := first.(Wrapped); ok {
		wrapWidths = wr.WrapWidths()
	}

	pageSize := 0
	if p, ok := first.(Paged); ok {
		pageSize = p.PageSize()
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
		styles = append([]func(string) string{nil}, styles...)
		if len(wrapWidths) > 0 {
			wrapWidths = append([]int{0}, wrapWidths...)
		}
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
	styles = extendStyles(styles, numCols)

	var err error
	if border == BorderNone {
		err = renderPlainTable(w, header, rows, footer, widths, aligns, styles, groups, wrapWidths, pageSize)
	} else {
		err = renderBorderedTable(w, title, header, rows, footer, widths, aligns, border, styles, groups, wrapWidths, pageSize)
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
		if w := runewidth.StringWidth(h); w > widths[i] {
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
	for i, cell := range footer {
		if w := runewidth.StringWidth(cell); i < numCols && w > widths[i] {
			widths[i] = w
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

func extendStyles(styles []func(string) string, numCols int) []func(string) string {
	if len(styles) >= numCols {
		return styles[:numCols]
	}
	extended := make([]func(string) string, numCols)
	copy(extended, styles)
	return extended
}

// --- Cell wrapping ---

func wrapCell(s string, width int) []string {
	if width <= 0 || runewidth.StringWidth(s) <= width {
		return []string{s}
	}
	var lines []string
	for len(s) > 0 {
		line := runewidth.Truncate(s, width, "")
		lineWidth := runewidth.StringWidth(line)
		if lineWidth == 0 && len(s) > 0 {
			// Safety: advance at least one rune to avoid infinite loop.
			r := []rune(s)
			line = string(r[0])
			lineWidth = runewidth.RuneWidth(r[0])
		}
		lines = append(lines, line)
		s = s[len(line):]
	}
	return lines
}

func wrapRow(cells []string, widths []int, wrapWidths []int) [][]string {
	wrapped := make([][]string, len(widths))
	for i, width := range widths {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		ww := 0
		if i < len(wrapWidths) {
			ww = wrapWidths[i]
		}
		if ww > 0 && ww < width {
			// Use wrap width for wrapping but column width for formatting.
			wrapped[i] = wrapCell(cell, ww)
		} else {
			wrapped[i] = []string{cell}
		}
	}
	return wrapped
}

func maxLines(wrapped [][]string) int {
	n := 1
	for _, lines := range wrapped {
		if len(lines) > n {
			n = len(lines)
		}
	}
	return n
}

// --- Plain table (BorderNone) ---

func renderPlainTable(w io.Writer, header []string, rows [][]string, footer []string, widths []int, aligns []Alignment, styles []func(string) string, groups []string, wrapWidths []int, pageSize int) error {
	if len(header) > 0 {
		if err := writePlainRow(w, header, widths, aligns, styles, wrapWidths); err != nil {
			return err
		}
		if err := writePlainSep(w, widths); err != nil {
			return err
		}
	}
	for i, row := range rows {
		if len(groups) > 0 && i > 0 && groups[i] != groups[i-1] {
			if err := writePlainSep(w, widths); err != nil {
				return err
			}
		}
		if pageSize > 0 && len(header) > 0 && i > 0 && i%pageSize == 0 {
			if err := writePlainSep(w, widths); err != nil {
				return err
			}
			if err := writePlainRow(w, header, widths, aligns, styles, wrapWidths); err != nil {
				return err
			}
			if err := writePlainSep(w, widths); err != nil {
				return err
			}
		}
		if err := writePlainRow(w, row, widths, aligns, styles, wrapWidths); err != nil {
			return err
		}
	}
	if len(footer) > 0 {
		if err := writePlainSep(w, widths); err != nil {
			return err
		}
		if err := writePlainRow(w, footer, widths, aligns, styles, wrapWidths); err != nil {
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

func writePlainRow(w io.Writer, cells []string, widths []int, aligns []Alignment, styles []func(string) string, wrapWidths []int) error {
	if len(wrapWidths) > 0 {
		wrapped := wrapRow(cells, widths, wrapWidths)
		nLines := maxLines(wrapped)
		for line := range nLines {
			parts := make([]string, len(widths))
			for i, width := range widths {
				cell := ""
				if line < len(wrapped[i]) {
					cell = wrapped[i][line]
				}
				formatted := formatTableCell(cell, width, aligns[i])
				if styles[i] != nil {
					formatted = styles[i](formatted)
				}
				parts[i] = formatted
			}
			text := strings.TrimRight(strings.Join(parts, "  "), " ")
			if _, err := fmt.Fprintln(w, text); err != nil {
				return err
			}
		}
		return nil
	}
	parts := make([]string, len(widths))
	for i, width := range widths {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		formatted := formatTableCell(cell, width, aligns[i])
		if styles[i] != nil {
			formatted = styles[i](formatted)
		}
		parts[i] = formatted
	}
	line := strings.TrimRight(strings.Join(parts, "  "), " ")
	_, err := fmt.Fprintln(w, line)
	return err
}

// --- Bordered table ---

func renderBorderedTable(w io.Writer, title string, header []string, rows [][]string, footer []string, widths []int, aligns []Alignment, style BorderStyle, styles []func(string) string, groups []string, wrapWidths []int, pageSize int) error {
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
		if err := drawBorderedRow(w, header, widths, aligns, bc.vertical, styles, wrapWidths); err != nil {
			return err
		}
		if err := drawHLine(w, widths, bc.leftTee, bc.horizontal, bc.cross, bc.rightTee); err != nil {
			return err
		}
	}

	for i, row := range rows {
		if len(groups) > 0 && i > 0 && groups[i] != groups[i-1] {
			if err := drawHLine(w, widths, bc.leftTee, bc.horizontal, bc.cross, bc.rightTee); err != nil {
				return err
			}
		}
		if pageSize > 0 && len(header) > 0 && i > 0 && i%pageSize == 0 {
			if err := drawHLine(w, widths, bc.leftTee, bc.horizontal, bc.cross, bc.rightTee); err != nil {
				return err
			}
			if err := drawBorderedRow(w, header, widths, aligns, bc.vertical, styles, wrapWidths); err != nil {
				return err
			}
			if err := drawHLine(w, widths, bc.leftTee, bc.horizontal, bc.cross, bc.rightTee); err != nil {
				return err
			}
		}
		if err := drawBorderedRow(w, row, widths, aligns, bc.vertical, styles, wrapWidths); err != nil {
			return err
		}
	}

	if len(footer) > 0 {
		if err := drawHLine(w, widths, bc.leftTee, bc.horizontal, bc.cross, bc.rightTee); err != nil {
			return err
		}
		if err := drawBorderedRow(w, footer, widths, aligns, bc.vertical, styles, wrapWidths); err != nil {
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

func drawBorderedRow(w io.Writer, cells []string, widths []int, aligns []Alignment, vert string, styles []func(string) string, wrapWidths []int) error {
	if len(wrapWidths) > 0 {
		wrapped := wrapRow(cells, widths, wrapWidths)
		nLines := maxLines(wrapped)
		for line := range nLines {
			var sb strings.Builder
			sb.WriteString(vert)
			for i, width := range widths {
				cell := ""
				if line < len(wrapped[i]) {
					cell = wrapped[i][line]
				}
				sb.WriteString(" ")
				formatted := formatTableCell(cell, width, aligns[i])
				if styles[i] != nil {
					formatted = styles[i](formatted)
				}
				sb.WriteString(formatted)
				sb.WriteString(" ")
				if i < len(widths)-1 {
					sb.WriteString(vert)
				}
			}
			sb.WriteString(vert)
			if _, err := fmt.Fprintln(w, sb.String()); err != nil {
				return err
			}
		}
		return nil
	}
	var sb strings.Builder
	sb.WriteString(vert)
	for i, width := range widths {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		sb.WriteString(" ")
		formatted := formatTableCell(cell, width, aligns[i])
		if styles[i] != nil {
			formatted = styles[i](formatted)
		}
		sb.WriteString(formatted)
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
	if width > 0 && runewidth.StringWidth(s) > width {
		if width <= 3 {
			s = runewidth.Truncate(s, width, "")
		} else {
			s = runewidth.Truncate(s, width, "...")
		}
	}
	return alignCell(s, width, align)
}

func alignCell(s string, width int, align Alignment) string {
	pad := width - runewidth.StringWidth(s)
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
