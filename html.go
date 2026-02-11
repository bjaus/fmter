package fmter

import (
	"fmt"
	"html"
	"io"
)

func writeHTML[T any](w io.Writer, items []T) error {
	if len(items) == 0 {
		return nil
	}
	first := any(items[0])
	if _, ok := first.(Rower); !ok {
		return fmt.Errorf("%w: format %q requires Rower, not implemented by %T", ErrMissingInterface, HTML, items[0])
	}

	var aligns []Alignment
	if a, ok := first.(Aligned); ok {
		aligns = a.Alignments()
	}

	if _, err := fmt.Fprintln(w, "<table>"); err != nil {
		return err
	}

	if t, ok := first.(Titled); ok {
		title := t.Title()
		if title != "" {
			if _, err := fmt.Fprintf(w, "  <caption>%s</caption>\n", html.EscapeString(title)); err != nil {
				return err
			}
		}
	}

	if h, ok := first.(Headed); ok {
		header := h.Header()
		if _, err := fmt.Fprintln(w, "  <thead>"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, "    <tr>"); err != nil {
			return err
		}
		for i, col := range header {
			style := alignStyle(aligns, i)
			if _, err := fmt.Fprintf(w, "      <th%s>%s</th>\n", style, html.EscapeString(col)); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w, "    </tr>"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, "  </thead>"); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(w, "  <tbody>"); err != nil {
		return err
	}
	for _, item := range items {
		row := any(item).(Rower).Row()
		if _, err := fmt.Fprintln(w, "    <tr>"); err != nil {
			return err
		}
		for i, cell := range row {
			style := alignStyle(aligns, i)
			if _, err := fmt.Fprintf(w, "      <td%s>%s</td>\n", style, html.EscapeString(cell)); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w, "    </tr>"); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, "  </tbody>"); err != nil {
		return err
	}

	if f, ok := first.(Footered); ok {
		footer := f.Footer()
		if _, err := fmt.Fprintln(w, "  <tfoot>"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, "    <tr>"); err != nil {
			return err
		}
		for i, cell := range footer {
			style := alignStyle(aligns, i)
			if _, err := fmt.Fprintf(w, "      <td%s>%s</td>\n", style, html.EscapeString(cell)); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w, "    </tr>"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, "  </tfoot>"); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintln(w, "</table>")
	return err
}

func alignStyle(aligns []Alignment, col int) string {
	if col >= len(aligns) {
		return ""
	}
	switch aligns[col] {
	case AlignRight:
		return ` style="text-align: right"`
	case AlignCenter:
		return ` style="text-align: center"`
	default:
		return ""
	}
}

