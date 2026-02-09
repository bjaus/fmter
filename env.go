package fmter

import (
	"fmt"
	"io"
)

func writeENV[T any](w io.Writer, items []T) error {
	if len(items) == 0 {
		return nil
	}
	if _, ok := any(items[0]).(Mappable); !ok {
		return fmt.Errorf("%w: format %q requires Mappable, not implemented by %T", ErrMissingInterface, ENV, items[0])
	}
	export := false
	if e, ok := any(items[0]).(Exported); ok {
		export = e.Export()
	}
	quoted := false
	if q, ok := any(items[0]).(Quoted); ok {
		quoted = q.Quote()
	}
	prefix := ""
	if export {
		prefix = "export "
	}
	for i, item := range items {
		if i > 0 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
		for _, kv := range any(item).(Mappable).Pairs() {
			var err error
			if quoted {
				_, err = fmt.Fprintf(w, "%s%s=%q\n", prefix, kv.Key, kv.Value)
			} else {
				_, err = fmt.Fprintf(w, "%s%s=%s\n", prefix, kv.Key, kv.Value)
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}
