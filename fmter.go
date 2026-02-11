package fmter

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Sentinel errors for programmatic error handling.
var (
	ErrUnsupportedFormat = errors.New("unsupported format")
	ErrMissingInterface  = errors.New("missing required interface")
	ErrInvalidTemplate   = errors.New("invalid template")
)

// Format represents an output format.
type Format string

const (
	JSON     Format = "json"
	YAML     Format = "yaml"
	CSV      Format = "csv"
	Table    Format = "table"
	Markdown Format = "markdown"
	List     Format = "list"
	ENV      Format = "env"
	Plain    Format = "plain"
	TSV      Format = "tsv"
	JSONL    Format = "jsonl"
	HTML     Format = "html"
)

const goTemplatePrefix = "go-template="

var formats = []Format{JSON, YAML, CSV, Table, Markdown, List, ENV, Plain, TSV, JSONL, HTML}

// String returns the format name.
func (f Format) String() string { return string(f) }

// Formats returns all supported static format names.
// GoTemplate is not included because it is parameterized.
func Formats() []Format {
	out := make([]Format, len(formats))
	copy(out, formats)
	return out
}

// GoTemplate returns a Format that renders items using a Go text/template.
// Each item is executed against the template and written on its own line.
func GoTemplate(tmpl string) Format {
	return Format(goTemplatePrefix + tmpl)
}

// ParseFormat parses a format string. Recognizes all static formats and
// go-template=<tmpl> strings.
func ParseFormat(s string) (Format, error) {
	if strings.HasPrefix(s, goTemplatePrefix) {
		return Format(s), nil
	}
	for _, f := range formats {
		if string(f) == s {
			return f, nil
		}
	}
	return "", fmt.Errorf("%w: %q", ErrUnsupportedFormat, s)
}

// IsSupported reports whether type T implements the interfaces required by
// format f. JSON, YAML, and GoTemplate always return true.
func IsSupported[T any](f Format) bool {
	if strings.HasPrefix(string(f), goTemplatePrefix) {
		return true
	}
	var zero T
	v := any(zero)
	switch f {
	case JSON, YAML, Plain, JSONL:
		return true
	case CSV, Table, TSV, HTML:
		_, ok := v.(Rower)
		return ok
	case Markdown:
		_, rower := v.(Rower)
		_, headed := v.(Headed)
		return rower && headed
	case List:
		_, ok := v.(Lister)
		return ok
	case ENV:
		_, ok := v.(Mappable)
		return ok
	default:
		return false
	}
}

// --- Core Format Interfaces ---

// Rower provides row data. Required for CSV, Table, and Markdown formats.
type Rower interface {
	Row() []string
}

// Lister provides a flat list of strings. Required for List format.
type Lister interface {
	List() []string
}

// Mappable provides key-value pairs. Required for ENV format.
type Mappable interface {
	Pairs() []KeyValue
}

// KeyValue is a single key-value pair.
type KeyValue struct {
	Key   string
	Value string
}

// --- Optional Interfaces ---

// Indented controls JSON/YAML indentation.
// Without it, JSON is compact and YAML uses its default indent.
type Indented interface {
	Indent() string
}

// Headed provides column headers for CSV, Table, and Markdown.
// Without it, CSV has no header row and Table renders without column headers.
type Headed interface {
	Header() []string
}

// Titled renders a title above the table.
// Default: no title.
type Titled interface {
	Title() string
}

// Bordered controls the table border style.
// Default: BorderRounded.
type Bordered interface {
	Border() BorderStyle
}

// Aligned sets per-column alignment.
// Default: AlignLeft. Also used by Markdown for alignment markers.
type Aligned interface {
	Alignments() []Alignment
}

// Footered renders a footer row below the table.
// Default: no footer.
type Footered interface {
	Footer() []string
}

// Numbered prepends a row number column.
// Default: no row numbers.
type Numbered interface {
	NumberHeader() string
}

// Captioned renders a line below the table.
// Default: no caption.
type Captioned interface {
	Caption() string
}

// Truncated sets maximum column widths for Table format.
// Cells exceeding the max are truncated with "...".
// A zero value means no limit for that column.
type Truncated interface {
	MaxWidths() []int
}

// Delimited controls the CSV field delimiter.
// Default: comma.
type Delimited interface {
	Delimiter() rune
}

// Separator controls the delimiter between list items.
// Default: newline.
type Separator interface {
	Sep() string
}

// Exported prefixes env pairs with "export ".
// Default: no prefix. Use with [Quoted] for shell-safe output.
type Exported interface {
	Export() bool
}

// Quoted wraps env values in double quotes.
// Default: unquoted. Useful for values that may contain spaces or special
// characters.
type Quoted interface {
	Quote() bool
}

// Styled provides per-column style functions for Table format.
// Each function wraps the fully formatted cell string (after truncation and
// alignment). Nil entries mean no styling for that column. Style functions
// are applied as the last step before writing, so ANSI codes never affect
// width calculations.
type Styled interface {
	Styles() []func(string) string
}

// Sorted is a metadata-only interface that declares a default sort column.
// The package does NOT sort; callers (CLI frameworks) can read this to apply
// sorting before rendering.
type Sorted interface {
	Sort() (column int, descending bool)
}

// Grouped returns a group key for the item. When consecutive items have
// different group keys, a separator line is inserted between them in Table
// format.
type Grouped interface {
	Group() string
}

// Wrapped provides per-column maximum widths for text wrapping in Table
// format. Cells exceeding the width wrap to multiple visual lines within
// the same row. A zero value means no wrapping for that column.
type Wrapped interface {
	WrapWidths() []int
}

// Paged controls header repetition for Table format. The header row is
// re-printed every PageSize data rows.
type Paged interface {
	PageSize() int
}

// Formatter is an escape hatch checked per-item. If Format returns non-nil
// bytes, those bytes are written directly. If it returns (nil, nil), the
// item falls through to default rendering.
type Formatter interface {
	Format(Format) ([]byte, error)
}

// --- Value Types ---

// BorderStyle controls table border characters.
type BorderStyle int

const (
	BorderRounded BorderStyle = iota // ╭─╮╰╯│┬┴├┤┼
	BorderNone                       // No borders, space-separated columns
	BorderASCII                      // +-+|
	BorderHeavy                      // ┏━┓┗┛┃┳┻┣┫╋
	BorderDouble                     // ╔═╗╚╝║╦╩╠╣╬
)

// Alignment controls column text alignment.
type Alignment int

const (
	AlignLeft Alignment = iota
	AlignCenter
	AlignRight
)

// Write formats items and writes to w.
func Write[T any](w io.Writer, f Format, items ...T) error {
	if len(items) > 0 {
		if fmtr, ok := any(items[0]).(Formatter); ok {
			_ = fmtr // type check on first item
			return writeFormatted(w, f, items)
		}
	}
	switch f {
	case JSON:
		return writeJSON(w, items)
	case YAML:
		return writeYAML(w, items)
	case CSV:
		return writeCSV(w, items)
	case Table:
		return writeTable(w, items)
	case Markdown:
		return writeMarkdown(w, items)
	case List:
		return writeList(w, items)
	case ENV:
		return writeENV(w, items)
	case Plain:
		return writePlain(w, items)
	case TSV:
		return writeTSV(w, items)
	case JSONL:
		return writeJSONL(w, items)
	case HTML:
		return writeHTML(w, items)
	default:
		if tmpl, ok := strings.CutPrefix(string(f), goTemplatePrefix); ok {
			return writeGoTemplate(w, tmpl, items)
		}
		return fmt.Errorf("%w: %q", ErrUnsupportedFormat, f)
	}
}

func writeFormatted[T any](w io.Writer, f Format, items []T) error {
	var fallback []T
	for _, item := range items {
		fmtr := any(item).(Formatter)
		data, err := fmtr.Format(f)
		if err != nil {
			return err
		}
		if data != nil {
			if _, werr := w.Write(data); werr != nil {
				return werr
			}
			continue
		}
		fallback = append(fallback, item)
	}
	if len(fallback) == 0 {
		return nil
	}
	// Temporarily strip the Formatter interface by routing through standard dispatch.
	switch f {
	case JSON:
		return writeJSON(w, fallback)
	case YAML:
		return writeYAML(w, fallback)
	case CSV:
		return writeCSV(w, fallback)
	case Table:
		return writeTable(w, fallback)
	case Markdown:
		return writeMarkdown(w, fallback)
	case List:
		return writeList(w, fallback)
	case ENV:
		return writeENV(w, fallback)
	case Plain:
		return writePlain(w, fallback)
	case TSV:
		return writeTSV(w, fallback)
	case JSONL:
		return writeJSONL(w, fallback)
	case HTML:
		return writeHTML(w, fallback)
	default:
		if tmpl, ok := strings.CutPrefix(string(f), goTemplatePrefix); ok {
			return writeGoTemplate(w, tmpl, fallback)
		}
		return fmt.Errorf("%w: %q", ErrUnsupportedFormat, f)
	}
}

// Marshal formats items and returns the bytes.
func Marshal[T any](f Format, items ...T) ([]byte, error) {
	var buf bytes.Buffer
	if err := Write(&buf, f, items...); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
