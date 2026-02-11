package fmter

import (
	"fmt"
	"io"
)

func writePlain[T any](w io.Writer, items []T) error {
	for _, item := range items {
		var s string
		if str, ok := any(item).(fmt.Stringer); ok {
			s = str.String()
		} else {
			s = fmt.Sprintf("%v", item)
		}
		if _, err := fmt.Fprintln(w, s); err != nil {
			return err
		}
	}
	return nil
}
