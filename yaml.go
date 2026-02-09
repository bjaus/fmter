package fmter

import (
	"io"

	"gopkg.in/yaml.v3"
)

func writeYAML[T any](w io.Writer, items []T) error {
	enc := yaml.NewEncoder(w)
	if len(items) > 0 {
		if ind, ok := any(items[0]).(Indented); ok {
			enc.SetIndent(len(ind.Indent()))
		}
	}
	if len(items) == 1 {
		if err := enc.Encode(items[0]); err != nil {
			return err
		}
	} else {
		if err := enc.Encode(items); err != nil {
			return err
		}
	}
	return enc.Close()
}
