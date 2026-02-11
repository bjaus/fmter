package fmter

import (
	"fmt"
	"io"
	"strings"
)

func writeTSV[T any](w io.Writer, items []T) error {
	if len(items) == 0 {
		return nil
	}
	if _, ok := any(items[0]).(Rower); !ok {
		return fmt.Errorf("%w: format %q requires Rower, not implemented by %T", ErrMissingInterface, TSV, items[0])
	}
	if h, ok := any(items[0]).(Headed); ok {
		if _, err := fmt.Fprintln(w, strings.Join(h.Header(), "\t")); err != nil {
			return err
		}
	}
	for _, item := range items {
		if _, err := fmt.Fprintln(w, strings.Join(any(item).(Rower).Row(), "\t")); err != nil {
			return err
		}
	}
	return nil
}
