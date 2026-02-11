package fmter

import (
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"strings"
)

// WriteIter formats items from an iterator and writes them to w as they arrive.
// For formats where items are independent (JSONL, CSV, TSV, List, ENV,
// GoTemplate, Plain), each item is written immediately. For formats that need
// all data for layout (Table, Markdown, HTML), items are collected into a slice
// first. For JSON, items are streamed as array elements. For YAML, items are
// collected (the encoder needs a complete document).
func WriteIter[T any](w io.Writer, f Format, seq iter.Seq[T]) error {
	switch f {
	case JSON:
		return streamJSON(w, seq)
	case YAML:
		return streamCollect(w, f, seq)
	case Table, Markdown, HTML:
		return streamCollect(w, f, seq)
	case CSV:
		return streamCSV(w, seq)
	case TSV:
		return streamTSV(w, seq)
	case JSONL:
		return streamJSONL(w, seq)
	case Plain:
		return streamPlain(w, seq)
	case List:
		return streamCollect(w, f, seq)
	case ENV:
		return streamCollect(w, f, seq)
	default:
		if tmpl, ok := strings.CutPrefix(string(f), goTemplatePrefix); ok {
			return streamGoTemplate(w, tmpl, seq)
		}
		return fmt.Errorf("%w: %q", ErrUnsupportedFormat, f)
	}
}

// WriteChan formats items from a channel and writes them to w.
// It is a thin wrapper around [WriteIter].
func WriteChan[T any](w io.Writer, f Format, ch <-chan T) error {
	return WriteIter(w, f, chanToIter(ch))
}

func chanToIter[T any](ch <-chan T) iter.Seq[T] {
	return func(yield func(T) bool) {
		for item := range ch {
			if !yield(item) {
				return
			}
		}
	}
}

func streamCollect[T any](w io.Writer, f Format, seq iter.Seq[T]) error {
	var items []T
	seq(func(item T) bool {
		items = append(items, item)
		return true
	})
	if len(items) == 0 {
		return nil
	}
	return Write(w, f, items...)
}

func streamJSON[T any](w io.Writer, seq iter.Seq[T]) error {
	if _, err := io.WriteString(w, "["); err != nil {
		return err
	}
	first := true
	var encErr error
	seq(func(item T) bool {
		if !first {
			if _, err := io.WriteString(w, ","); err != nil {
				encErr = err
				return false
			}
		}
		first = false
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		if ind, ok := any(item).(Indented); ok {
			enc.SetIndent("", ind.Indent())
		}
		if err := enc.Encode(item); err != nil {
			encErr = err
			return false
		}
		return true
	})
	if encErr != nil {
		return encErr
	}
	_, err := io.WriteString(w, "]\n")
	return err
}

func streamCSV[T any](w io.Writer, seq iter.Seq[T]) error {
	first := true
	var streamErr error
	seq(func(item T) bool {
		if first {
			first = false
			if _, ok := any(item).(Rower); !ok {
				streamErr = fmt.Errorf("%w: format %q requires Rower, not implemented by %T", ErrMissingInterface, CSV, item)
				return false
			}
			if err := Write(w, CSV, item); err != nil {
				streamErr = err
				return false
			}
			return true
		}
		// Subsequent items: write row only (header already written by first Write).
		if err := writeCSVRow(w, any(item).(Rower).Row()); err != nil {
			streamErr = err
			return false
		}
		return true
	})
	return streamErr
}

func streamTSV[T any](w io.Writer, seq iter.Seq[T]) error {
	first := true
	var streamErr error
	seq(func(item T) bool {
		if first {
			first = false
			if _, ok := any(item).(Rower); !ok {
				streamErr = fmt.Errorf("%w: format %q requires Rower, not implemented by %T", ErrMissingInterface, TSV, item)
				return false
			}
			if err := Write(w, TSV, item); err != nil {
				streamErr = err
				return false
			}
			return true
		}
		if _, err := fmt.Fprintln(w, strings.Join(any(item).(Rower).Row(), "\t")); err != nil {
			streamErr = err
			return false
		}
		return true
	})
	return streamErr
}

func streamJSONL[T any](w io.Writer, seq iter.Seq[T]) error {
	var streamErr error
	seq(func(item T) bool {
		enc := json.NewEncoder(w)
		if ind, ok := any(item).(Indented); ok {
			enc.SetIndent("", ind.Indent())
		}
		if err := enc.Encode(item); err != nil {
			streamErr = err
			return false
		}
		return true
	})
	return streamErr
}

func streamPlain[T any](w io.Writer, seq iter.Seq[T]) error {
	var streamErr error
	seq(func(item T) bool {
		var s string
		if str, ok := any(item).(fmt.Stringer); ok {
			s = str.String()
		} else {
			s = fmt.Sprintf("%v", item)
		}
		if _, err := fmt.Fprintln(w, s); err != nil {
			streamErr = err
			return false
		}
		return true
	})
	return streamErr
}

func streamGoTemplate[T any](w io.Writer, tmplStr string, seq iter.Seq[T]) error {
	var items []T
	seq(func(item T) bool {
		items = append(items, item)
		return true
	})
	if len(items) == 0 {
		return nil
	}
	return writeGoTemplate(w, tmplStr, items)
}
