# fmter

[![Go Reference](https://pkg.go.dev/badge/github.com/bjaus/fmter.svg)](https://pkg.go.dev/github.com/bjaus/fmter)
[![Go Report Card](https://goreportcard.com/badge/github.com/bjaus/fmter)](https://goreportcard.com/report/github.com/bjaus/fmter)
[![CI](https://github.com/bjaus/fmter/actions/workflows/ci.yml/badge.svg)](https://github.com/bjaus/fmter/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/bjaus/fmter/branch/main/graph/badge.svg)](https://codecov.io/gh/bjaus/fmter)

Multi-format output renderer for Go CLI tools. One type, many formats — like the AWS CLI's `--output` flag.

Define your data type once, implement a few small interfaces, and let `fmter` render it as JSON, YAML, a rich table, CSV, Markdown, a flat list, env vars, or a custom Go template.

## Install

```bash
go get github.com/bjaus/fmter
```

## Quick Start

```go
type Service struct {
    Name   string `json:"name" yaml:"name"`
    Status string `json:"status" yaml:"status"`
    Port   int    `json:"port" yaml:"port"`
}

// Rower unlocks CSV and Table formats.
func (s Service) Row() []string { return []string{s.Name, s.Status, fmt.Sprint(s.Port)} }

// Headed adds column headers to CSV, Table, and Markdown.
func (s Service) Header() []string { return []string{"Name", "Status", "Port"} }
```

Now render in any format based on a CLI flag:

```go
f, err := fmter.ParseFormat(flagValue) // "json", "table", "csv", etc.
if err != nil {
    log.Fatal(err)
}

services := []Service{
    {Name: "api", Status: "running", Port: 8080},
    {Name: "web", Status: "stopped", Port: 3000},
}

fmter.Write(os.Stdout, f, services...)
```

**JSON** (works on any value, no interface needed):
```json
[{"name":"api","status":"running","port":8080},{"name":"web","status":"stopped","port":3000}]
```

**Table** (requires `Rower`):
```
╭──────┬─────────┬──────╮
│ Name │ Status  │ Port │
├──────┼─────────┼──────┤
│ api  │ running │ 8080 │
│ web  │ stopped │ 3000 │
╰──────┴─────────┴──────╯
```

**CSV** (requires `Rower`):
```
Name,Status,Port
api,running,8080
web,stopped,3000
```

## How It Works

The package uses a **progressive interface** design. A minimal interface gets you working, and optional interfaces enhance the output:

```
JSON / YAML ──── any value (no interface needed)
CSV / Table ──── Rower (row data)
Markdown ─────── Rower + Headed
List ─────────── Lister
ENV ──────────── Mappable
GoTemplate ───── any value
```

Implement more interfaces to unlock more features — each one is independent and optional:

```go
// Add a title bar above the table.
func (s Service) Title() string { return "Services" }

// Use ASCII border characters instead of Unicode.
func (s Service) Border() fmter.BorderStyle { return fmter.BorderASCII }

// Right-align the Port column.
func (s Service) Alignments() []fmter.Alignment {
    return []fmter.Alignment{fmter.AlignLeft, fmter.AlignLeft, fmter.AlignRight}
}

// Add a footer row.
func (s Service) Footer() []string { return []string{"Total", "", "2"} }

// Add automatic row numbers.
func (s Service) NumberHeader() string { return "#" }
```

## Formats

| Format | Required | Description |
|---|---|---|
| `json` | any value | Compact JSON (implement `Indented` for pretty-print) |
| `yaml` | any value | YAML via `gopkg.in/yaml.v3` |
| `csv` | `Rower` | RFC 4180 CSV (+ `Headed`, `Delimited`) |
| `table` | `Rower` | Rich bordered table with many options |
| `markdown` | `Rower` + `Headed` | GitHub-flavored Markdown table |
| `list` | `Lister` | Flat string list (+ `Separator`) |
| `env` | `Mappable` | `KEY=VALUE` pairs (+ `Exported`, `Quoted`) |
| `go-template=...` | any value | Custom Go `text/template` |

## Interfaces

### Required (one per format family)

| Interface | Method | Used By |
|---|---|---|
| `Rower` | `Row() []string` | CSV, Table, Markdown |
| `Lister` | `List() []string` | List |
| `Mappable` | `Pairs() []KeyValue` | ENV |

### Optional (enhance any format)

| Interface | Method | Effect |
|---|---|---|
| `Headed` | `Header() []string` | Column headers (CSV, Table, Markdown) |
| `Indented` | `Indent() string` | Pretty-print indent (JSON, YAML) |
| `Titled` | `Title() string` | Title bar above table |
| `Bordered` | `Border() BorderStyle` | Table border style |
| `Aligned` | `Alignments() []Alignment` | Per-column alignment (Table, Markdown) |
| `Footered` | `Footer() []string` | Footer row below table |
| `Numbered` | `NumberHeader() string` | Auto row numbers |
| `Captioned` | `Caption() string` | Text below table |
| `Truncated` | `MaxWidths() []int` | Max column widths with `...` |
| `Delimited` | `Delimiter() rune` | Custom CSV delimiter |
| `Separator` | `Sep() string` | Custom list separator |
| `Exported` | `Export() bool` | `export ` prefix for ENV |
| `Quoted` | `Quote() bool` | Double-quote ENV values |

## Table Border Styles

```go
fmter.BorderRounded  // ╭─╮╰╯│  (default)
fmter.BorderASCII    // +-+|
fmter.BorderHeavy    // ┏━┓┗┛┃
fmter.BorderDouble   // ╔═╗╚╝║
fmter.BorderNone     // No borders, space-separated
```

## API

```go
// Write formats items and writes to w.
fmter.Write(w, fmter.JSON, items...)

// Marshal returns the formatted bytes.
data, err := fmter.Marshal(fmter.Table, items...)

// ParseFormat converts a CLI flag string to a Format.
f, err := fmter.ParseFormat("table")

// GoTemplate creates a parameterized format.
fmter.Write(w, fmter.GoTemplate("{{.Name}}: {{.Status}}"), items...)

// IsSupported checks if a type implements the required interfaces.
if fmter.IsSupported[Service](fmter.CSV) { ... }

// Formats returns all static format names.
for _, f := range fmter.Formats() { ... }
```

## Errors

All errors wrap sentinel values for `errors.Is` checks:

```go
errors.Is(err, fmter.ErrUnsupportedFormat) // unknown format string
errors.Is(err, fmter.ErrMissingInterface)  // type doesn't implement required interface
errors.Is(err, fmter.ErrInvalidTemplate)   // bad go-template syntax
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

[MIT](LICENSE)
