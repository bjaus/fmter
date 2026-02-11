// Package fmter renders structured data in multiple output formats.
//
// Supported formats are JSON, YAML, CSV, Table, Markdown, List, ENV, Plain,
// TSV, JSONL, HTML, and GoTemplate. The central entry points are [Write] and
// [Marshal], which accept a [Format] constant and variadic items of any type.
// JSON, YAML, Plain, and JSONL work on any value; other formats require the
// items to implement specific interfaces.
//
// # Interface Design
//
// The package uses a layered interface design. A minimal interface unlocks a
// format, and optional interfaces enhance the rendering:
//
//   - [Rower] → CSV, Table, Markdown, TSV, HTML (row data)
//   - [Headed] → adds column headers to CSV, Table, Markdown, TSV, HTML
//   - [Lister] → List format
//   - [Mappable] → ENV format
//
// Use [IsSupported] to check at runtime whether a type implements the required
// interfaces for a given format:
//
//	if fmter.IsSupported[MyType](fmter.CSV) { ... }
//
// # JSON and YAML
//
// Any value works. Implement [Indented] to control indentation:
//
//	fmter.Write(os.Stdout, fmter.JSON, myStruct)
//	fmter.Write(os.Stdout, fmter.YAML, items...)
//
// # CSV
//
// Requires [Rower]. Optional interfaces:
//
//   - [Headed] — header row
//   - [Delimited] — custom field delimiter (default comma)
//
// # TSV
//
// Requires [Rower]. Tab-delimited output with no quoting. Optional:
//
//   - [Headed] — header row
//
// # Table
//
// Requires [Rower]. Optional interfaces control every aspect of rendering:
//
//   - [Headed] — column headers
//   - [Titled] — title above the table
//   - [Bordered] — border style (default [BorderRounded])
//   - [Aligned] — per-column alignment
//   - [Footered] — footer row
//   - [Numbered] — row number column
//   - [Captioned] — line below the table
//   - [Truncated] — max column widths with "..." truncation
//   - [Styled] — per-column style functions (e.g., ANSI colors)
//   - [Grouped] — separator between groups of rows
//   - [Paged] — repeat header every N rows
//   - [Wrapped] — multi-line cells with per-column wrap widths
//
// # Markdown
//
// Requires [Rower] and [Headed]. Renders a GitHub-flavored Markdown table.
// Implement [Aligned] to set column alignment markers.
//
// # HTML
//
// Requires [Rower]. Renders a semantic HTML table. Optional interfaces:
//
//   - [Headed] → <thead>
//   - [Titled] → <caption>
//   - [Footered] → <tfoot>
//   - [Aligned] → text-align style on <td>/<th>
//
// # List
//
// Requires [Lister]. Implement [Separator] to control the delimiter between
// items (default newline).
//
// # ENV
//
// Requires [Mappable]. Optional interfaces:
//
//   - [Exported] — prefix lines with "export "
//   - [Quoted] — wrap values in double quotes
//
// # Plain
//
// Works on any value. Uses [fmt.Stringer] if available, otherwise
// fmt.Sprintf("%v"). One item per line.
//
// # JSONL
//
// Works on any value. One JSON object per line (no array wrapping).
// Implement [Indented] for per-line indentation.
//
// # GoTemplate
//
// Use [GoTemplate] to create a parameterized format that renders each item
// using a Go [text/template]:
//
//	fmter.Write(os.Stdout, fmter.GoTemplate("{{.Name}}: {{.Age}}"), items...)
//
// # Streaming
//
// [WriteIter] and [WriteChan] support streaming output for iterator and
// channel sources. Formats that render items independently (Plain, JSONL,
// CSV, TSV, GoTemplate) write each item as it arrives. Formats that need
// all data for layout (Table, Markdown, HTML) collect items first.
//
// # Formatter
//
// Implement [Formatter] for per-item control. If Format returns non-nil
// bytes, they are written directly; returning (nil, nil) falls through to
// default rendering.
//
// # Format Selection
//
// Use [ParseFormat] to convert a CLI flag string into a [Format]. It
// recognizes all static formats and "go-template=<tmpl>" strings:
//
//	f, err := fmter.ParseFormat(flagValue)
//	fmter.Write(os.Stdout, f, items...)
//
// # Errors
//
// The package exports sentinel errors for programmatic handling:
//
//   - [ErrUnsupportedFormat] — unknown format string
//   - [ErrMissingInterface] — items don't implement the required interface
//   - [ErrInvalidTemplate] — invalid go-template syntax
package fmter
