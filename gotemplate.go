package fmter

import (
	"fmt"
	"io"
	"text/template"
)

func writeGoTemplate[T any](w io.Writer, tmplStr string, items []T) error {
	tmpl, err := template.New("").Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidTemplate, err)
	}
	for _, item := range items {
		if err := tmpl.Execute(w, item); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	return nil
}
