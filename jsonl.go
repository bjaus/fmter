package fmter

import (
	"encoding/json"
	"io"
)

func writeJSONL[T any](w io.Writer, items []T) error {
	for _, item := range items {
		enc := json.NewEncoder(w)
		if ind, ok := any(item).(Indented); ok {
			enc.SetIndent("", ind.Indent())
		}
		if err := enc.Encode(item); err != nil {
			return err
		}
	}
	return nil
}
