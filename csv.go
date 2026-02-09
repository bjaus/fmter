package fmter

import (
	"encoding/csv"
	"fmt"
	"io"
)

func writeCSV[T any](w io.Writer, items []T) error {
	if len(items) == 0 {
		return nil
	}
	if _, ok := any(items[0]).(Rower); !ok {
		return fmt.Errorf("%w: format %q requires Rower, not implemented by %T", ErrMissingInterface, CSV, items[0])
	}
	cw := csv.NewWriter(w)
	if d, ok := any(items[0]).(Delimited); ok {
		cw.Comma = d.Delimiter()
	}
	if h, ok := any(items[0]).(Headed); ok {
		if err := cw.Write(h.Header()); err != nil {
			return err
		}
	}
	for _, item := range items {
		if err := cw.Write(any(item).(Rower).Row()); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
