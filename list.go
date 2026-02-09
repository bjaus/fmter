package fmter

import (
	"fmt"
	"io"
	"strings"
)

func writeList[T any](w io.Writer, items []T) error {
	if len(items) == 0 {
		return nil
	}
	if _, ok := any(items[0]).(Lister); !ok {
		return fmt.Errorf("%w: format %q requires Lister, not implemented by %T", ErrMissingInterface, List, items[0])
	}
	sep := "\n"
	if s, ok := any(items[0]).(Separator); ok {
		sep = s.Sep()
	}
	var all []string
	for _, item := range items {
		all = append(all, any(item).(Lister).List()...)
	}
	if len(all) == 0 {
		return nil
	}
	_, err := io.WriteString(w, strings.Join(all, sep)+"\n")
	return err
}
