package fmter

import (
	"encoding/json"
	"io"
)

func writeJSON[T any](w io.Writer, items []T) error {
	enc := json.NewEncoder(w)
	if len(items) > 0 {
		if ind, ok := any(items[0]).(Indented); ok {
			enc.SetIndent("", ind.Indent())
		}
	}
	if len(items) == 1 {
		return enc.Encode(items[0])
	}
	return enc.Encode(items)
}
