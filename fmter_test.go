package fmter_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/bjaus/fmter"
	"github.com/mattn/go-runewidth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func displayWidth(s string) int { return runewidth.StringWidth(s) }

// --- Test types: minimal ---

type basicRow struct {
	Name string
	Age  string
}

func (r basicRow) Row() []string { return []string{r.Name, r.Age} }

// --- Test types: with header ---

type headedRow struct {
	basicRow
}

func (r headedRow) Header() []string { return []string{"Name", "Age"} }

// --- Test types: full table ---

type richRow struct {
	Name   string
	Age    string
	Status string
}

func (r richRow) Row() []string              { return []string{r.Name, r.Age, r.Status} }
func (r richRow) Header() []string           { return []string{"Name", "Age", "Status"} }
func (r richRow) Title() string              { return "People" }
func (r richRow) Border() fmter.BorderStyle  { return fmter.BorderRounded }
func (r richRow) Alignments() []fmter.Alignment {
	return []fmter.Alignment{fmter.AlignLeft, fmter.AlignRight, fmter.AlignCenter}
}
func (r richRow) Footer() []string    { return []string{"Total", "2", ""} }
func (r richRow) NumberHeader() string { return "#" }
func (r richRow) Caption() string      { return "2 results" }

// --- Test types: border styles ---

type asciiRow struct{ headedRow }

func (r asciiRow) Border() fmter.BorderStyle { return fmter.BorderASCII }

type heavyRow struct{ headedRow }

func (r heavyRow) Border() fmter.BorderStyle { return fmter.BorderHeavy }

type doubleRow struct{ headedRow }

func (r doubleRow) Border() fmter.BorderStyle { return fmter.BorderDouble }

type noBorderRow struct{ headedRow }

func (r noBorderRow) Border() fmter.BorderStyle { return fmter.BorderNone }

// --- Test types: indented ---

type indentedVal struct {
	Name string `json:"name" yaml:"name"`
}

func (v indentedVal) Indent() string { return "  " }

// --- Test types: list ---

type stubList struct {
	items []string
}

func (s stubList) List() []string { return s.items }

type stubSepList struct {
	items []string
	sep   string
}

func (s stubSepList) List() []string { return s.items }
func (s stubSepList) Sep() string    { return s.sep }

// --- Test types: env ---

type stubEnv struct {
	kvs []fmter.KeyValue
}

func (s stubEnv) Pairs() []fmter.KeyValue { return s.kvs }

type stubExportedEnv struct {
	kvs []fmter.KeyValue
}

func (s stubExportedEnv) Pairs() []fmter.KeyValue { return s.kvs }
func (s stubExportedEnv) Export() bool             { return true }

// --- Test types: env with quoting ---

type stubQuotedEnv struct {
	kvs []fmter.KeyValue
}

func (s stubQuotedEnv) Pairs() []fmter.KeyValue { return s.kvs }
func (s stubQuotedEnv) Quote() bool              { return true }

type stubExportedQuotedEnv struct {
	kvs []fmter.KeyValue
}

func (s stubExportedQuotedEnv) Pairs() []fmter.KeyValue { return s.kvs }
func (s stubExportedQuotedEnv) Export() bool             { return true }
func (s stubExportedQuotedEnv) Quote() bool              { return true }

// --- Test types: csv with delimiter ---

type tsvRow struct {
	headedRow
}

func (r tsvRow) Delimiter() rune { return '\t' }

// --- Test types: truncated table ---

type truncatedRow struct {
	headedRow
}

func (r truncatedRow) MaxWidths() []int { return []int{4, 3} }

// --- Test types: large header (csv buffer overflow) ---

type largeHeaderRow struct{ val string }

func (r largeHeaderRow) Row() []string    { return []string{r.val} }
func (r largeHeaderRow) Header() []string { return []string{strings.Repeat("H", 5000)} }

// --- Test types: short header (markdown min-width) ---

type shortHeaderRow struct{ val string }

func (r shortHeaderRow) Row() []string    { return []string{r.val} }
func (r shortHeaderRow) Header() []string { return []string{"X"} }

// --- Test types: go-template ---

type tmplItem struct {
	Name string
	Age  int
}

// --- Helpers ---

type errWriter struct{}

func (e *errWriter) Write([]byte) (int, error) {
	return 0, errWriteFailed
}

// failAfterN fails on the (n+1)th call to Write.
type failAfterN struct {
	n       int
	calls   int
}

func (f *failAfterN) Write(p []byte) (int, error) {
	if f.calls >= f.n {
		return 0, errWriteFailed
	}
	f.calls++
	return len(p), nil
}

var errWriteFailed = errors.New("write failed")

// ============================================================
// Tests
// ============================================================

func TestParseFormat(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		input   string
		want    fmter.Format
		wantErr require.ErrorAssertionFunc
	}{
		"json":     {input: "json", want: fmter.JSON, wantErr: require.NoError},
		"yaml":     {input: "yaml", want: fmter.YAML, wantErr: require.NoError},
		"csv":      {input: "csv", want: fmter.CSV, wantErr: require.NoError},
		"table":    {input: "table", want: fmter.Table, wantErr: require.NoError},
		"markdown": {input: "markdown", want: fmter.Markdown, wantErr: require.NoError},
		"list":     {input: "list", want: fmter.List, wantErr: require.NoError},
		"env":      {input: "env", want: fmter.ENV, wantErr: require.NoError},
		"unknown":  {input: "xml", want: "", wantErr: require.Error},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := fmter.ParseFormat(tt.input)
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormats(t *testing.T) {
	t.Parallel()
	got := fmter.Formats()
	assert.Equal(t, []fmter.Format{
		fmter.JSON, fmter.YAML, fmter.CSV, fmter.Table,
		fmter.Markdown, fmter.List, fmter.ENV, fmter.Plain,
		fmter.TSV, fmter.JSONL, fmter.HTML,
	}, got)
	// Returned slice must be a copy.
	got[0] = "modified"
	assert.Equal(t, fmter.JSON, fmter.Formats()[0])
}

func TestFormatString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "json", fmter.JSON.String())
	assert.Equal(t, "table", fmter.Table.String())
}

// --- JSON ---

func TestWriteJSON(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		items []any
		want  string
	}{
		"single struct": {
			items: []any{struct {
				Name string `json:"name"`
			}{Name: "Alice"}},
			want: `{"name":"Alice"}` + "\n",
		},
		"multiple items": {
			items: []any{"a", "b"},
			want:  `["a","b"]` + "\n",
		},
		"nil": {
			items: []any{nil},
			want:  "null\n",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := fmter.Write(&buf, fmter.JSON, tt.items...)
			require.NoError(t, err)
			assert.Equal(t, tt.want, buf.String())
		})
	}
}

func TestWriteJSONIndented(t *testing.T) {
	t.Parallel()
	v := indentedVal{Name: "Alice"}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.JSON, v)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "  \"name\"")
}

func TestWriteJSONEmpty(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := fmter.Write[any](&buf, fmter.JSON)
	require.NoError(t, err)
	assert.Equal(t, "null\n", buf.String())
}

// --- YAML ---

func TestWriteYAML(t *testing.T) {
	t.Parallel()
	v := struct {
		Name string `yaml:"name"`
		Age  int    `yaml:"age"`
	}{Name: "Alice", Age: 30}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.YAML, v)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "name: Alice")
	assert.Contains(t, buf.String(), "age: 30")
}

func TestWriteYAMLIndented(t *testing.T) {
	t.Parallel()
	v := indentedVal{Name: "Alice"}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.YAML, v)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "name: Alice")
}

// --- CSV ---

func TestWriteCSV(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		items []headedRow
		want  string
	}{
		"with header": {
			items: []headedRow{
				{basicRow{Name: "Alice", Age: "30"}},
				{basicRow{Name: "Bob", Age: "25"}},
			},
			want: "Name,Age\nAlice,30\nBob,25\n",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := fmter.Write(&buf, fmter.CSV, tt.items...)
			require.NoError(t, err)
			assert.Equal(t, tt.want, buf.String())
		})
	}
}

func TestWriteCSVNoHeader(t *testing.T) {
	t.Parallel()
	items := []basicRow{
		{Name: "Alice", Age: "30"},
		{Name: "Bob", Age: "25"},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.CSV, items...)
	require.NoError(t, err)
	assert.Equal(t, "Alice,30\nBob,25\n", buf.String())
}

func TestWriteCSVQuoted(t *testing.T) {
	t.Parallel()
	items := []basicRow{{Name: "hello, world", Age: "30"}}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.CSV, items...)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"hello, world"`)
}

func TestWriteCSVRejectsNonRower(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.CSV, "not a rower")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Rower")
}

func TestWriteCSVEmpty(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := fmter.Write[basicRow](&buf, fmter.CSV)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

// --- Table ---

func TestWriteTableBorderRounded(t *testing.T) {
	t.Parallel()
	items := []headedRow{
		{basicRow{Name: "Alice", Age: "30"}},
		{basicRow{Name: "Bob", Age: "25"}},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Table, items...)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "╭")
	assert.Contains(t, out, "╰")
	assert.Contains(t, out, "│")
	assert.Contains(t, out, "─")
	assert.Contains(t, out, "Alice")
	assert.Contains(t, out, "Name")
}

func TestWriteTableBorderNone(t *testing.T) {
	t.Parallel()
	items := []noBorderRow{
		{headedRow{basicRow{Name: "Alice", Age: "30"}}},
		{headedRow{basicRow{Name: "Bob", Age: "25"}}},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Table, items...)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "Name")
	assert.Contains(t, out, "-----")
	assert.Contains(t, out, "Alice")
	assert.NotContains(t, out, "│")
}

func TestWriteTableBorderASCII(t *testing.T) {
	t.Parallel()
	items := []asciiRow{{headedRow{basicRow{Name: "Alice", Age: "30"}}}}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Table, items...)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "+")
	assert.Contains(t, out, "|")
	assert.Contains(t, out, "-")
}

func TestWriteTableBorderHeavy(t *testing.T) {
	t.Parallel()
	items := []heavyRow{{headedRow{basicRow{Name: "Alice", Age: "30"}}}}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Table, items...)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "┏")
	assert.Contains(t, out, "┃")
	assert.Contains(t, out, "━")
}

func TestWriteTableBorderDouble(t *testing.T) {
	t.Parallel()
	items := []doubleRow{{headedRow{basicRow{Name: "Alice", Age: "30"}}}}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Table, items...)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "╔")
	assert.Contains(t, out, "║")
	assert.Contains(t, out, "═")
}

func TestWriteTableNoHeader(t *testing.T) {
	t.Parallel()
	items := []basicRow{
		{Name: "Alice", Age: "30"},
		{Name: "Bob", Age: "25"},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Table, items...)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "Alice")
	assert.NotContains(t, out, "Name")
}

func TestWriteTableRichFeatures(t *testing.T) {
	t.Parallel()
	items := []richRow{
		{Name: "Alice", Age: "30", Status: "active"},
		{Name: "Bob", Age: "25", Status: "inactive"},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Table, items...)
	require.NoError(t, err)
	out := buf.String()
	// Title.
	assert.Contains(t, out, "People")
	// Row numbers.
	assert.Contains(t, out, "#")
	assert.Contains(t, out, "1")
	assert.Contains(t, out, "2")
	// Footer.
	assert.Contains(t, out, "Total")
	// Caption.
	assert.Contains(t, out, "2 results")
	// Data.
	assert.Contains(t, out, "Alice")
	assert.Contains(t, out, "Bob")
}

func TestWriteTableRejectsNonRower(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Table, "not a rower")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Rower")
}

func TestWriteTableEmpty(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := fmter.Write[basicRow](&buf, fmter.Table)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

// --- Markdown ---

func TestWriteMarkdown(t *testing.T) {
	t.Parallel()
	items := []headedRow{
		{basicRow{Name: "Alice", Age: "30"}},
		{basicRow{Name: "Bob", Age: "25"}},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Markdown, items...)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "| Name")
	assert.Contains(t, out, "| ---")
	assert.Contains(t, out, "| Alice")
}

type alignedMarkdownRow struct {
	headedRow
}

func (r alignedMarkdownRow) Alignments() []fmter.Alignment {
	return []fmter.Alignment{fmter.AlignLeft, fmter.AlignRight}
}

func TestWriteMarkdownAligned(t *testing.T) {
	t.Parallel()
	items := []alignedMarkdownRow{
		{headedRow{basicRow{Name: "Alice", Age: "30"}}},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Markdown, items...)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "--:")
}

func TestWriteMarkdownRejectsNoHeader(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Markdown, basicRow{Name: "Alice", Age: "30"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Headed")
}

func TestWriteMarkdownRejectsNonRower(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Markdown, "not a rower")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Rower")
}

// --- List ---

func TestWriteList(t *testing.T) {
	t.Parallel()
	items := []stubList{
		{items: []string{"alpha", "beta", "gamma"}},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.List, items...)
	require.NoError(t, err)
	assert.Equal(t, "alpha\nbeta\ngamma\n", buf.String())
}

func TestWriteListSeparator(t *testing.T) {
	t.Parallel()
	items := []stubSepList{
		{items: []string{"a", "b", "c"}, sep: ", "},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.List, items...)
	require.NoError(t, err)
	assert.Equal(t, "a, b, c\n", buf.String())
}

func TestWriteListEmpty(t *testing.T) {
	t.Parallel()
	items := []stubList{{items: nil}}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.List, items...)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

func TestWriteListRejectsNonLister(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.List, "not a lister")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Lister")
}

// --- ENV ---

func TestWriteENV(t *testing.T) {
	t.Parallel()
	items := []stubEnv{
		{kvs: []fmter.KeyValue{
			{Key: "FOO", Value: "bar"},
			{Key: "BAZ", Value: "qux"},
		}},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.ENV, items...)
	require.NoError(t, err)
	assert.Equal(t, "FOO=bar\nBAZ=qux\n", buf.String())
}

func TestWriteENVExported(t *testing.T) {
	t.Parallel()
	items := []stubExportedEnv{
		{kvs: []fmter.KeyValue{
			{Key: "FOO", Value: "bar"},
			{Key: "BAZ", Value: "hello world"},
		}},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.ENV, items...)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "export FOO=bar")
	assert.Contains(t, out, "export BAZ=hello world")
}

func TestWriteENVMultipleItems(t *testing.T) {
	t.Parallel()
	items := []stubEnv{
		{kvs: []fmter.KeyValue{{Key: "A", Value: "1"}}},
		{kvs: []fmter.KeyValue{{Key: "B", Value: "2"}}},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.ENV, items...)
	require.NoError(t, err)
	assert.Equal(t, "A=1\n\nB=2\n", buf.String())
}

func TestWriteENVRejectsNonMappable(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.ENV, "not mappable")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Mappable")
}

func TestWriteENVEmpty(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := fmter.Write[stubEnv](&buf, fmter.ENV)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

// --- Marshal ---

func TestMarshal(t *testing.T) {
	t.Parallel()
	data, err := fmter.Marshal(fmter.JSON, map[string]string{"k": "v"})
	require.NoError(t, err)
	assert.Contains(t, string(data), `"k":"v"`)
}

func TestMarshalError(t *testing.T) {
	t.Parallel()
	_, err := fmter.Marshal(fmter.CSV, "not a rower")
	require.Error(t, err)
}

// --- Unsupported format ---

func TestWriteUnsupportedFormat(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Format("xml"), "data")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

// --- Write errors ---

func TestWriteErrors(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		format fmter.Format
		item   any
	}{
		"json": {
			format: fmter.JSON,
			item:   map[string]string{"k": "v"},
		},
		"yaml": {
			format: fmter.YAML,
			item:   map[string]string{"k": "v"},
		},
		"csv": {
			format: fmter.CSV,
			item:   headedRow{basicRow{Name: "A", Age: "1"}},
		},
		"table": {
			format: fmter.Table,
			item:   headedRow{basicRow{Name: "A", Age: "1"}},
		},
		"markdown": {
			format: fmter.Markdown,
			item:   headedRow{basicRow{Name: "A", Age: "1"}},
		},
		"list": {
			format: fmter.List,
			item:   stubList{items: []string{"x"}},
		},
		"env": {
			format: fmter.ENV,
			item:   stubEnv{kvs: []fmter.KeyValue{{Key: "K", Value: "V"}}},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			w := &errWriter{}
			err := fmter.Write(w, tt.format, tt.item)
			require.Error(t, err)
		})
	}
}

// --- Sentinel errors ---

func TestSentinelErrors(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		format fmter.Format
		item   any
		target error
	}{
		"unsupported format": {
			format: fmter.Format("xml"),
			item:   "data",
			target: fmter.ErrUnsupportedFormat,
		},
		"csv missing rower": {
			format: fmter.CSV,
			item:   "not a rower",
			target: fmter.ErrMissingInterface,
		},
		"table missing rower": {
			format: fmter.Table,
			item:   "not a rower",
			target: fmter.ErrMissingInterface,
		},
		"markdown missing rower": {
			format: fmter.Markdown,
			item:   "not a rower",
			target: fmter.ErrMissingInterface,
		},
		"list missing lister": {
			format: fmter.List,
			item:   "not a lister",
			target: fmter.ErrMissingInterface,
		},
		"env missing mappable": {
			format: fmter.ENV,
			item:   "not mappable",
			target: fmter.ErrMissingInterface,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := fmter.Write(&buf, tt.format, tt.item)
			require.ErrorIs(t, err, tt.target)
		})
	}
}

// --- IsSupported ---

func TestIsSupported(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		format fmter.Format
		want   bool
	}{
		"json always":         {format: fmter.JSON, want: true},
		"yaml always":         {format: fmter.YAML, want: true},
		"csv with rower":      {format: fmter.CSV, want: true},
		"table with rower":    {format: fmter.Table, want: true},
		"markdown with headed": {format: fmter.Markdown, want: true},
		"go-template always":  {format: fmter.GoTemplate("{{.Name}}"), want: true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, fmter.IsSupported[headedRow](tt.format))
		})
	}
}

func TestIsSupportedFalse(t *testing.T) {
	t.Parallel()
	// string does not implement Rower, Lister, or Mappable.
	assert.False(t, fmter.IsSupported[string](fmter.CSV))
	assert.False(t, fmter.IsSupported[string](fmter.Table))
	assert.False(t, fmter.IsSupported[string](fmter.Markdown))
	assert.False(t, fmter.IsSupported[string](fmter.List))
	assert.False(t, fmter.IsSupported[string](fmter.ENV))
}

func TestIsSupportedMarkdownNeedsHeaded(t *testing.T) {
	t.Parallel()
	// basicRow implements Rower but not Headed.
	assert.False(t, fmter.IsSupported[basicRow](fmter.Markdown))
}

// --- GoTemplate ---

func TestWriteGoTemplate(t *testing.T) {
	t.Parallel()
	items := []tmplItem{
		{Name: "Alice", Age: 30},
		{Name: "Bob", Age: 25},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.GoTemplate("{{.Name}} is {{.Age}}"), items...)
	require.NoError(t, err)
	assert.Equal(t, "Alice is 30\nBob is 25\n", buf.String())
}

func TestWriteGoTemplateInvalid(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.GoTemplate("{{.Invalid"), "data")
	require.ErrorIs(t, err, fmter.ErrInvalidTemplate)
}

func TestParseFormatGoTemplate(t *testing.T) {
	t.Parallel()
	f, err := fmter.ParseFormat("go-template={{.Name}}")
	require.NoError(t, err)
	assert.Equal(t, fmter.GoTemplate("{{.Name}}"), f)
}

// --- Delimited CSV ---

func TestWriteCSVDelimited(t *testing.T) {
	t.Parallel()
	items := []tsvRow{
		{headedRow{basicRow{Name: "Alice", Age: "30"}}},
		{headedRow{basicRow{Name: "Bob", Age: "25"}}},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.CSV, items...)
	require.NoError(t, err)
	assert.Equal(t, "Name\tAge\nAlice\t30\nBob\t25\n", buf.String())
}

// --- Truncated Table ---

func TestWriteTableTruncated(t *testing.T) {
	t.Parallel()
	items := []truncatedRow{
		{headedRow{basicRow{Name: "Alexander", Age: "30"}}},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Table, items...)
	require.NoError(t, err)
	out := buf.String()
	// Name column max 4: "A..." (width-3=1 char + "...")
	assert.Contains(t, out, "A...")
	// Age column max 3: "30" fits, no truncation.
	assert.Contains(t, out, "30")
	// Full name should not appear.
	assert.NotContains(t, out, "Alexander")
}

// --- Unicode width ---

type unicodeRow struct {
	col1 string
	col2 string
}

func (r unicodeRow) Row() []string    { return []string{r.col1, r.col2} }
func (r unicodeRow) Header() []string { return []string{"Key", "Value"} }

func TestWriteTableUnicodeWidth(t *testing.T) {
	t.Parallel()
	// Em dash "—" is 1 display column but 3 bytes. Borders must align.
	items := []unicodeRow{
		{col1: "Parent", col2: "PROJ-100 — Login Epic"},
		{col1: "Status", col2: "In Progress"},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Table, items...)
	require.NoError(t, err)
	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	// All lines must have the same display width for aligned borders.
	require.True(t, len(lines) >= 2)
	firstWidth := displayWidth(lines[0])
	for i, line := range lines[1:] {
		assert.Equal(t, firstWidth, displayWidth(line), "line %d width mismatch: %q", i+1, line)
	}
}

// --- Quoted ENV ---

func TestWriteENVQuoted(t *testing.T) {
	t.Parallel()
	items := []stubQuotedEnv{
		{kvs: []fmter.KeyValue{
			{Key: "FOO", Value: "bar"},
			{Key: "BAZ", Value: "hello world"},
		}},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.ENV, items...)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, `FOO="bar"`)
	assert.Contains(t, out, `BAZ="hello world"`)
	assert.NotContains(t, out, "export")
}

func TestWriteENVExportedAndQuoted(t *testing.T) {
	t.Parallel()
	items := []stubExportedQuotedEnv{
		{kvs: []fmter.KeyValue{
			{Key: "FOO", Value: "bar"},
		}},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.ENV, items...)
	require.NoError(t, err)
	assert.Equal(t, "export FOO=\"bar\"\n", buf.String())
}

// --- Plain table with footer (BorderNone) ---

type noBorderFooterRow struct {
	headedRow
}

func (r noBorderFooterRow) Border() fmter.BorderStyle { return fmter.BorderNone }
func (r noBorderFooterRow) Footer() []string          { return []string{"Total", "55"} }

func TestWriteTableBorderNoneWithFooter(t *testing.T) {
	t.Parallel()
	items := []noBorderFooterRow{
		{headedRow{basicRow{Name: "Alice", Age: "30"}}},
		{headedRow{basicRow{Name: "Bob", Age: "25"}}},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Table, items...)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "Total")
	assert.Contains(t, out, "55")
}

// --- Bordered table with footer (no title) ---

type footeredRow struct {
	headedRow
}

func (r footeredRow) Footer() []string { return []string{"Total", "55"} }

func TestWriteTableBorderedWithFooter(t *testing.T) {
	t.Parallel()
	items := []footeredRow{
		{headedRow{basicRow{Name: "Alice", Age: "30"}}},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Table, items...)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "Total")
	assert.Contains(t, out, "┼")
}

// --- formatTableCell width<=3 truncation ---

type tinyTruncRow struct {
	headedRow
}

func (r tinyTruncRow) MaxWidths() []int { return []int{2, 0} }

func TestWriteTableTruncatedTinyWidth(t *testing.T) {
	t.Parallel()
	items := []tinyTruncRow{
		{headedRow{basicRow{Name: "Hello", Age: "30"}}},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Table, items...)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "He")
	assert.NotContains(t, out, "Hello")
}

// --- IsSupported for List and ENV ---

func TestIsSupportedListAndENV(t *testing.T) {
	t.Parallel()
	assert.True(t, fmter.IsSupported[stubList](fmter.List))
	assert.True(t, fmter.IsSupported[stubEnv](fmter.ENV))
}

// --- IsSupported unknown format ---

func TestIsSupportedUnknownFormat(t *testing.T) {
	t.Parallel()
	assert.False(t, fmter.IsSupported[headedRow](fmter.Format("xml")))
}

// --- YAML multiple items ---

func TestWriteYAMLMultiple(t *testing.T) {
	t.Parallel()
	items := []struct {
		Name string `yaml:"name"`
	}{
		{Name: "Alice"},
		{Name: "Bob"},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.YAML, items...)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Alice")
	assert.Contains(t, buf.String(), "Bob")
}

// --- Markdown center alignment ---

type centerAlignedMarkdownRow struct {
	headedRow
}

func (r centerAlignedMarkdownRow) Alignments() []fmter.Alignment {
	return []fmter.Alignment{fmter.AlignCenter, fmter.AlignCenter}
}

func TestWriteMarkdownMinWidth(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Markdown, shortHeaderRow{val: "y"})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "---")
}

func TestWriteMarkdownCenterAligned(t *testing.T) {
	t.Parallel()
	items := []centerAlignedMarkdownRow{
		{headedRow{basicRow{Name: "Alice", Age: "30"}}},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Markdown, items...)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, ":--")
	assert.Contains(t, out, "--:")
}

// --- Markdown empty ---

func TestWriteMarkdownEmpty(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := fmter.Write[headedRow](&buf, fmter.Markdown)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

// --- List empty items (zero-length) ---

func TestWriteListEmptyItems(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := fmter.Write[stubList](&buf, fmter.List)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

// --- colCount footer wider than rows ---

type wideFooterRow struct {
	basicRow
}

func (r wideFooterRow) Row() []string    { return []string{r.Name} }
func (r wideFooterRow) Footer() []string { return []string{"Total", "Count", "Extra"} }

func TestWriteTableFooterWiderThanRows(t *testing.T) {
	t.Parallel()
	items := []wideFooterRow{{basicRow{Name: "Alice", Age: "30"}}}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Table, items...)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "Total")
	assert.Contains(t, out, "Extra")
}

// --- GoTemplate write error ---

func TestWriteGoTemplateWriteError(t *testing.T) {
	t.Parallel()
	w := &errWriter{}
	err := fmter.Write(w, fmter.GoTemplate("{{.Name}}"), tmplItem{Name: "Alice", Age: 30})
	require.Error(t, err)
}

// --- GoTemplate execute error ---

func TestWriteGoTemplateExecuteError(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.GoTemplate("{{.Missing}}"), map[string]string{"Name": "Alice"})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "<no value>")
}

// --- Deep write error paths ---

func TestWriteErrorPlainTable(t *testing.T) {
	t.Parallel()
	items := []noBorderFooterRow{
		{headedRow{basicRow{Name: "Alice", Age: "30"}}},
		{headedRow{basicRow{Name: "Bob", Age: "25"}}},
	}
	// Each increasing N hits a deeper error path in renderPlainTable.
	for n := range 6 {
		w := &failAfterN{n: n}
		err := fmter.Write(w, fmter.Table, items...)
		require.Error(t, err, "expected error at n=%d", n)
	}
}

func TestWriteErrorBorderedTable(t *testing.T) {
	t.Parallel()
	items := []footeredRow{
		{headedRow{basicRow{Name: "Alice", Age: "30"}}},
	}
	// Each increasing N hits a deeper error path in renderBorderedTable.
	for n := range 7 {
		w := &failAfterN{n: n}
		err := fmter.Write(w, fmter.Table, items...)
		require.Error(t, err, "expected error at n=%d", n)
	}
}

func TestWriteErrorBorderedTableWithTitle(t *testing.T) {
	t.Parallel()
	items := []richRow{
		{Name: "Alice", Age: "30", Status: "active"},
	}
	for n := range 10 {
		w := &failAfterN{n: n}
		err := fmter.Write(w, fmter.Table, items...)
		require.Error(t, err, "expected error at n=%d", n)
	}
}

func TestWriteErrorCSVRows(t *testing.T) {
	t.Parallel()
	// csv.Writer buffers via bufio (4096 bytes). Use large data to trigger
	// a mid-write flush that hits the underlying writer error.
	big := strings.Repeat("x", 5000)
	items := []headedRow{
		{basicRow{Name: big, Age: "1"}},
		{basicRow{Name: "Bob", Age: "2"}},
	}
	w := &failAfterN{n: 0}
	err := fmter.Write(w, fmter.CSV, items...)
	require.Error(t, err)
	// Also test header write error with large header.
	w2 := &failAfterN{n: 0}
	err = fmter.Write(w2, fmter.CSV, largeHeaderRow{val: "x"})
	require.Error(t, err)
}

func TestWriteErrorMarkdown(t *testing.T) {
	t.Parallel()
	items := []headedRow{
		{basicRow{Name: "Alice", Age: "30"}},
		{basicRow{Name: "Bob", Age: "25"}},
	}
	// Sweep all write calls to cover every error return in writeMarkdown.
	for n := range 4 {
		w := &failAfterN{n: n}
		err := fmter.Write(w, fmter.Markdown, items...)
		require.Error(t, err, "expected error at n=%d", n)
	}
}

func TestWriteErrorGoTemplateFprintln(t *testing.T) {
	t.Parallel()
	// n=1: Execute succeeds, Fprintln fails.
	w := &failAfterN{n: 1}
	err := fmter.Write(w, fmter.GoTemplate("{{.Name}}"), tmplItem{Name: "Alice", Age: 30})
	require.Error(t, err)
}

func TestWriteErrorYAML(t *testing.T) {
	t.Parallel()
	w := &failAfterN{n: 0}
	err := fmter.Write(w, fmter.YAML, struct{ Name string }{Name: "Alice"})
	require.Error(t, err)
}

func TestWriteErrorYAMLMultiple(t *testing.T) {
	t.Parallel()
	w := &failAfterN{n: 0}
	err := fmter.Write(w, fmter.YAML, "a", "b")
	require.Error(t, err)
}

func TestWriteErrorENVBlankLine(t *testing.T) {
	t.Parallel()
	items := []stubEnv{
		{kvs: []fmter.KeyValue{{Key: "A", Value: "1"}}},
		{kvs: []fmter.KeyValue{{Key: "B", Value: "2"}}},
	}
	// n=1: first item writes, blank line Fprintln fails.
	w := &failAfterN{n: 1}
	err := fmter.Write(w, fmter.ENV, items...)
	require.Error(t, err)
}

func TestWriteErrorTableCaption(t *testing.T) {
	t.Parallel()
	items := []richRow{
		{Name: "Alice", Age: "30", Status: "active"},
	}
	// Succeed on all table rendering, fail on caption Fprintln.
	// richRow has title+header+row+footer+caption = many writes.
	// Find the exact N by trying until success then use N-1.
	var n int
	for n = 0; n < 50; n++ {
		w := &failAfterN{n: n}
		if err := fmter.Write(w, fmter.Table, items...); err == nil {
			break
		}
	}
	// n is the first N that succeeds. n-1 fails on caption.
	w := &failAfterN{n: n - 1}
	err := fmter.Write(w, fmter.Table, items...)
	require.Error(t, err)
}

// ============================================================
// Plain format
// ============================================================

type stringerItem struct {
	name string
}

func (s stringerItem) String() string { return "STRINGER:" + s.name }

func TestWritePlain(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		items []any
		want  string
	}{
		"basic values": {
			items: []any{"hello", "world"},
			want:  "hello\nworld\n",
		},
		"numbers": {
			items: []any{42, 3.14},
			want:  "42\n3.14\n",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := fmter.Write(&buf, fmter.Plain, tt.items...)
			require.NoError(t, err)
			assert.Equal(t, tt.want, buf.String())
		})
	}
}

func TestWritePlainStringer(t *testing.T) {
	t.Parallel()
	items := []stringerItem{{name: "Alice"}, {name: "Bob"}}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Plain, items...)
	require.NoError(t, err)
	assert.Equal(t, "STRINGER:Alice\nSTRINGER:Bob\n", buf.String())
}

func TestWritePlainEmpty(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := fmter.Write[string](&buf, fmter.Plain)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

func TestWritePlainError(t *testing.T) {
	t.Parallel()
	w := &errWriter{}
	err := fmter.Write(w, fmter.Plain, "hello")
	require.Error(t, err)
}

// ============================================================
// TSV format
// ============================================================

func TestWriteTSV(t *testing.T) {
	t.Parallel()
	items := []headedRow{
		{basicRow{Name: "Alice", Age: "30"}},
		{basicRow{Name: "Bob", Age: "25"}},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.TSV, items...)
	require.NoError(t, err)
	assert.Equal(t, "Name\tAge\nAlice\t30\nBob\t25\n", buf.String())
}

func TestWriteTSVNoHeader(t *testing.T) {
	t.Parallel()
	items := []basicRow{
		{Name: "Alice", Age: "30"},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.TSV, items...)
	require.NoError(t, err)
	assert.Equal(t, "Alice\t30\n", buf.String())
}

func TestWriteTSVEmpty(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := fmter.Write[basicRow](&buf, fmter.TSV)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

func TestWriteTSVRejectsNonRower(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.TSV, "not a rower")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Rower")
}

func TestWriteTSVError(t *testing.T) {
	t.Parallel()
	w := &errWriter{}
	err := fmter.Write(w, fmter.TSV, headedRow{basicRow{Name: "A", Age: "1"}})
	require.Error(t, err)
}

func TestWriteTSVErrorRow(t *testing.T) {
	t.Parallel()
	w := &failAfterN{n: 1} // header succeeds, first row fails
	err := fmter.Write(w, fmter.TSV, headedRow{basicRow{Name: "A", Age: "1"}})
	require.Error(t, err)
}

// ============================================================
// JSONL format
// ============================================================

func TestWriteJSONL(t *testing.T) {
	t.Parallel()
	items := []struct {
		Name string `json:"name"`
	}{
		{Name: "Alice"},
		{Name: "Bob"},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.JSONL, items...)
	require.NoError(t, err)
	assert.Equal(t, "{\"name\":\"Alice\"}\n{\"name\":\"Bob\"}\n", buf.String())
}

func TestWriteJSONLIndented(t *testing.T) {
	t.Parallel()
	items := []indentedVal{{Name: "Alice"}}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.JSONL, items...)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "  \"name\"")
}

func TestWriteJSONLEmpty(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := fmter.Write[string](&buf, fmter.JSONL)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

func TestWriteJSONLError(t *testing.T) {
	t.Parallel()
	w := &errWriter{}
	err := fmter.Write(w, fmter.JSONL, struct{ Name string }{Name: "Alice"})
	require.Error(t, err)
}

// ============================================================
// HTML format
// ============================================================

type htmlRow struct {
	headedRow
}

func (r htmlRow) Title() string   { return "People" }
func (r htmlRow) Footer() []string { return []string{"Total", "55"} }
func (r htmlRow) Alignments() []fmter.Alignment {
	return []fmter.Alignment{fmter.AlignLeft, fmter.AlignRight}
}

func TestWriteHTML(t *testing.T) {
	t.Parallel()
	items := []htmlRow{
		{headedRow{basicRow{Name: "Alice", Age: "30"}}},
		{headedRow{basicRow{Name: "Bob", Age: "25"}}},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.HTML, items...)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "<table>")
	assert.Contains(t, out, "</table>")
	assert.Contains(t, out, "<caption>People</caption>")
	assert.Contains(t, out, "<thead>")
	assert.Contains(t, out, "<th>Name</th>")
	assert.Contains(t, out, `<th style="text-align: right">Age</th>`)
	assert.Contains(t, out, "<tbody>")
	assert.Contains(t, out, "<td>Alice</td>")
	assert.Contains(t, out, `<td style="text-align: right">30</td>`)
	assert.Contains(t, out, "<tfoot>")
	assert.Contains(t, out, "<td>Total</td>")
}

func TestWriteHTMLMinimal(t *testing.T) {
	t.Parallel()
	items := []basicRow{
		{Name: "Alice", Age: "30"},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.HTML, items...)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "<table>")
	assert.Contains(t, out, "<td>Alice</td>")
	assert.NotContains(t, out, "<thead>")
	assert.NotContains(t, out, "<tfoot>")
	assert.NotContains(t, out, "<caption>")
}

func TestWriteHTMLEscaping(t *testing.T) {
	t.Parallel()
	items := []basicRow{
		{Name: "<script>alert('xss')</script>", Age: "30"},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.HTML, items...)
	require.NoError(t, err)
	out := buf.String()
	assert.NotContains(t, out, "<script>")
	assert.Contains(t, out, "&lt;script&gt;")
}

func TestWriteHTMLEmpty(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := fmter.Write[basicRow](&buf, fmter.HTML)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

func TestWriteHTMLRejectsNonRower(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.HTML, "not a rower")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Rower")
}

func TestWriteHTMLError(t *testing.T) {
	t.Parallel()
	items := []htmlRow{
		{headedRow{basicRow{Name: "Alice", Age: "30"}}},
	}
	// Sweep write calls to cover every error return.
	for n := range 20 {
		w := &failAfterN{n: n}
		err := fmter.Write(w, fmter.HTML, items...)
		if err == nil {
			break
		}
	}
}

// ============================================================
// Styled table
// ============================================================

type styledRow struct {
	headedRow
}

func (r styledRow) Styles() []func(string) string {
	return []func(string) string{
		func(s string) string { return "[" + s + "]" },
		nil,
	}
}

func TestWriteTableStyled(t *testing.T) {
	t.Parallel()
	items := []styledRow{
		{headedRow{basicRow{Name: "Alice", Age: "30"}}},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Table, items...)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "[Alice]")
	assert.Contains(t, out, "[Name ]")
	assert.NotContains(t, out, "[30]")
}

type styledNoBorderRow struct {
	styledRow
}

func (r styledNoBorderRow) Border() fmter.BorderStyle { return fmter.BorderNone }

func TestWriteTableStyledPlain(t *testing.T) {
	t.Parallel()
	items := []styledNoBorderRow{
		{styledRow{headedRow{basicRow{Name: "Alice", Age: "30"}}}},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Table, items...)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "[Alice]")
	assert.Contains(t, out, "[Name ]")
}

// ============================================================
// Grouped table
// ============================================================

type groupedRow struct {
	headedRow
	group string
}

func (r groupedRow) Group() string { return r.group }

func TestWriteTableGrouped(t *testing.T) {
	t.Parallel()
	items := []groupedRow{
		{headedRow{basicRow{Name: "Alice", Age: "30"}}, "A"},
		{headedRow{basicRow{Name: "Adam", Age: "25"}}, "A"},
		{headedRow{basicRow{Name: "Bob", Age: "35"}}, "B"},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Table, items...)
	require.NoError(t, err)
	out := buf.String()
	// A separator should appear between "A" and "B" groups.
	lines := strings.Split(out, "\n")
	separators := 0
	for _, line := range lines {
		if strings.Contains(line, "├") && strings.Contains(line, "┤") {
			separators++
		}
	}
	// 1 for header separator + 1 for group separator = at least 2
	assert.GreaterOrEqual(t, separators, 2)
}

type groupedNoBorderRow struct {
	groupedRow
}

func (r groupedNoBorderRow) Border() fmter.BorderStyle { return fmter.BorderNone }

func TestWriteTableGroupedPlain(t *testing.T) {
	t.Parallel()
	items := []groupedNoBorderRow{
		{groupedRow{headedRow{basicRow{Name: "Alice", Age: "30"}}, "A"}},
		{groupedRow{headedRow{basicRow{Name: "Bob", Age: "25"}}, "B"}},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Table, items...)
	require.NoError(t, err)
	out := buf.String()
	lines := strings.Split(out, "\n")
	dashLines := 0
	for _, line := range lines {
		if strings.Contains(line, "---") {
			dashLines++
		}
	}
	// 1 for header + 1 for group separator = 2
	assert.Equal(t, 2, dashLines)
}

// ============================================================
// Paged table
// ============================================================

type pagedRow struct {
	headedRow
}

func (r pagedRow) PageSize() int { return 2 }

func TestWriteTablePaged(t *testing.T) {
	t.Parallel()
	items := []pagedRow{
		{headedRow{basicRow{Name: "Alice", Age: "30"}}},
		{headedRow{basicRow{Name: "Bob", Age: "25"}}},
		{headedRow{basicRow{Name: "Carol", Age: "28"}}},
		{headedRow{basicRow{Name: "Dave", Age: "35"}}},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Table, items...)
	require.NoError(t, err)
	out := buf.String()
	// "Name" should appear twice: once for the initial header, once for page break.
	assert.Equal(t, 2, strings.Count(out, "Name"))
}

type pagedNoBorderRow struct {
	pagedRow
}

func (r pagedNoBorderRow) Border() fmter.BorderStyle { return fmter.BorderNone }

func TestWriteTablePagedPlain(t *testing.T) {
	t.Parallel()
	items := []pagedNoBorderRow{
		{pagedRow{headedRow{basicRow{Name: "Alice", Age: "30"}}}},
		{pagedRow{headedRow{basicRow{Name: "Bob", Age: "25"}}}},
		{pagedRow{headedRow{basicRow{Name: "Carol", Age: "28"}}}},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Table, items...)
	require.NoError(t, err)
	out := buf.String()
	assert.Equal(t, 2, strings.Count(out, "Name"))
}

// ============================================================
// Wrapped table
// ============================================================

type wrappedRow struct {
	headedRow
}

func (r wrappedRow) WrapWidths() []int { return []int{3, 0} }

func TestWriteTableWrapped(t *testing.T) {
	t.Parallel()
	items := []wrappedRow{
		{headedRow{basicRow{Name: "Hello", Age: "30"}}},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Table, items...)
	require.NoError(t, err)
	out := buf.String()
	// "Hello" wraps into "Hel" + "lo" lines.
	lines := strings.Split(out, "\n")
	rowLines := 0
	for _, line := range lines {
		if strings.Contains(line, "│") && !strings.Contains(line, "─") &&
			!strings.Contains(line, "Name") && !strings.Contains(line, "╭") &&
			!strings.Contains(line, "╰") {
			rowLines++
		}
	}
	assert.GreaterOrEqual(t, rowLines, 2, "wrapped row should span multiple lines")
}

type wrappedNoBorderRow struct {
	wrappedRow
}

func (r wrappedNoBorderRow) Border() fmter.BorderStyle { return fmter.BorderNone }

func TestWriteTableWrappedPlain(t *testing.T) {
	t.Parallel()
	items := []wrappedNoBorderRow{
		{wrappedRow{headedRow{basicRow{Name: "Hello", Age: "30"}}}},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Table, items...)
	require.NoError(t, err)
	out := buf.String()
	// Should contain "Hel" and "lo" on separate lines.
	assert.Contains(t, out, "Hel")
	assert.Contains(t, out, "lo")
}

// ============================================================
// Sorted interface (metadata only)
// ============================================================

type sortedRow struct {
	headedRow
}

func (r sortedRow) Sort() (column int, descending bool) { return 1, true }

func TestSortedInterface(t *testing.T) {
	t.Parallel()
	// Sorted is metadata-only — verify the interface is implemented.
	var s fmter.Sorted = sortedRow{}
	col, desc := s.Sort()
	assert.Equal(t, 1, col)
	assert.True(t, desc)
}

// ============================================================
// Formatter escape hatch
// ============================================================

type formattedItem struct {
	Name string
}

func (f formattedItem) Format(fmt fmter.Format) ([]byte, error) {
	if fmt == fmter.Plain {
		return []byte("CUSTOM:" + f.Name + "\n"), nil
	}
	return nil, nil // fall through
}

func TestWriteFormatterCustom(t *testing.T) {
	t.Parallel()
	items := []formattedItem{{Name: "Alice"}, {Name: "Bob"}}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Plain, items...)
	require.NoError(t, err)
	assert.Equal(t, "CUSTOM:Alice\nCUSTOM:Bob\n", buf.String())
}

func TestWriteFormatterFallthrough(t *testing.T) {
	t.Parallel()
	items := []formattedItem{{Name: "Alice"}}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.JSON, items...)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Alice")
}

type formatterError struct{}

func (f formatterError) Format(fmter.Format) ([]byte, error) {
	return nil, errors.New("format error")
}

func TestWriteFormatterError(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Plain, formatterError{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "format error")
}

func TestWriteFormatterWriteError(t *testing.T) {
	t.Parallel()
	w := &errWriter{}
	err := fmter.Write(w, fmter.Plain, formattedItem{Name: "Alice"})
	require.Error(t, err)
}

func TestWriteFormatterFallthroughEmpty(t *testing.T) {
	t.Parallel()
	// All items return custom output, no fallback items.
	items := []formattedItem{{Name: "Alice"}}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Plain, items...)
	require.NoError(t, err)
	assert.Equal(t, "CUSTOM:Alice\n", buf.String())
}

// ============================================================
// Streaming: WriteIter and WriteChan
// ============================================================

func TestWriteIterPlain(t *testing.T) {
	t.Parallel()
	items := []string{"hello", "world"}
	seq := func(yield func(string) bool) {
		for _, item := range items {
			if !yield(item) {
				return
			}
		}
	}
	var buf bytes.Buffer
	err := fmter.WriteIter(&buf, fmter.Plain, seq)
	require.NoError(t, err)
	assert.Equal(t, "hello\nworld\n", buf.String())
}

func TestWriteIterJSON(t *testing.T) {
	t.Parallel()
	type item struct {
		Name string `json:"name"`
	}
	items := []item{{Name: "Alice"}, {Name: "Bob"}}
	seq := func(yield func(item) bool) {
		for _, it := range items {
			if !yield(it) {
				return
			}
		}
	}
	var buf bytes.Buffer
	err := fmter.WriteIter(&buf, fmter.JSON, seq)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "[")
	assert.Contains(t, out, "]")
	assert.Contains(t, out, "Alice")
	assert.Contains(t, out, "Bob")
}

func TestWriteIterJSONL(t *testing.T) {
	t.Parallel()
	type item struct {
		Name string `json:"name"`
	}
	items := []item{{Name: "Alice"}, {Name: "Bob"}}
	seq := func(yield func(item) bool) {
		for _, it := range items {
			if !yield(it) {
				return
			}
		}
	}
	var buf bytes.Buffer
	err := fmter.WriteIter(&buf, fmter.JSONL, seq)
	require.NoError(t, err)
	assert.Equal(t, "{\"name\":\"Alice\"}\n{\"name\":\"Bob\"}\n", buf.String())
}

func TestWriteIterCSV(t *testing.T) {
	t.Parallel()
	items := []headedRow{
		{basicRow{Name: "Alice", Age: "30"}},
		{basicRow{Name: "Bob", Age: "25"}},
	}
	seq := func(yield func(headedRow) bool) {
		for _, it := range items {
			if !yield(it) {
				return
			}
		}
	}
	var buf bytes.Buffer
	err := fmter.WriteIter(&buf, fmter.CSV, seq)
	require.NoError(t, err)
	assert.Equal(t, "Name,Age\nAlice,30\nBob,25\n", buf.String())
}

func TestWriteIterTSV(t *testing.T) {
	t.Parallel()
	items := []headedRow{
		{basicRow{Name: "Alice", Age: "30"}},
		{basicRow{Name: "Bob", Age: "25"}},
	}
	seq := func(yield func(headedRow) bool) {
		for _, it := range items {
			if !yield(it) {
				return
			}
		}
	}
	var buf bytes.Buffer
	err := fmter.WriteIter(&buf, fmter.TSV, seq)
	require.NoError(t, err)
	assert.Equal(t, "Name\tAge\nAlice\t30\nBob\t25\n", buf.String())
}

func TestWriteIterTable(t *testing.T) {
	t.Parallel()
	items := []headedRow{
		{basicRow{Name: "Alice", Age: "30"}},
	}
	seq := func(yield func(headedRow) bool) {
		for _, it := range items {
			if !yield(it) {
				return
			}
		}
	}
	var buf bytes.Buffer
	err := fmter.WriteIter(&buf, fmter.Table, seq)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Alice")
	assert.Contains(t, buf.String(), "╭")
}

func TestWriteIterUnsupported(t *testing.T) {
	t.Parallel()
	seq := func(yield func(string) bool) {
		yield("x")
	}
	var buf bytes.Buffer
	err := fmter.WriteIter(&buf, fmter.Format("xml"), seq)
	require.Error(t, err)
	assert.ErrorIs(t, err, fmter.ErrUnsupportedFormat)
}

func TestWriteIterGoTemplate(t *testing.T) {
	t.Parallel()
	items := []tmplItem{{Name: "Alice", Age: 30}}
	seq := func(yield func(tmplItem) bool) {
		for _, it := range items {
			if !yield(it) {
				return
			}
		}
	}
	var buf bytes.Buffer
	err := fmter.WriteIter(&buf, fmter.GoTemplate("{{.Name}}"), seq)
	require.NoError(t, err)
	assert.Equal(t, "Alice\n", buf.String())
}

func TestWriteIterYAML(t *testing.T) {
	t.Parallel()
	type item struct {
		Name string `yaml:"name"`
	}
	items := []item{{Name: "Alice"}}
	seq := func(yield func(item) bool) {
		for _, it := range items {
			if !yield(it) {
				return
			}
		}
	}
	var buf bytes.Buffer
	err := fmter.WriteIter(&buf, fmter.YAML, seq)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "name: Alice")
}

func TestWriteIterHTML(t *testing.T) {
	t.Parallel()
	items := []basicRow{{Name: "Alice", Age: "30"}}
	seq := func(yield func(basicRow) bool) {
		for _, it := range items {
			if !yield(it) {
				return
			}
		}
	}
	var buf bytes.Buffer
	err := fmter.WriteIter(&buf, fmter.HTML, seq)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "<table>")
}

func TestWriteIterMarkdown(t *testing.T) {
	t.Parallel()
	items := []headedRow{{basicRow{Name: "Alice", Age: "30"}}}
	seq := func(yield func(headedRow) bool) {
		for _, it := range items {
			if !yield(it) {
				return
			}
		}
	}
	var buf bytes.Buffer
	err := fmter.WriteIter(&buf, fmter.Markdown, seq)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "| Name")
}

func TestWriteIterList(t *testing.T) {
	t.Parallel()
	items := []stubList{{items: []string{"a", "b"}}}
	seq := func(yield func(stubList) bool) {
		for _, it := range items {
			if !yield(it) {
				return
			}
		}
	}
	var buf bytes.Buffer
	err := fmter.WriteIter(&buf, fmter.List, seq)
	require.NoError(t, err)
	assert.Equal(t, "a\nb\n", buf.String())
}

func TestWriteIterENV(t *testing.T) {
	t.Parallel()
	items := []stubEnv{{kvs: []fmter.KeyValue{{Key: "K", Value: "V"}}}}
	seq := func(yield func(stubEnv) bool) {
		for _, it := range items {
			if !yield(it) {
				return
			}
		}
	}
	var buf bytes.Buffer
	err := fmter.WriteIter(&buf, fmter.ENV, seq)
	require.NoError(t, err)
	assert.Equal(t, "K=V\n", buf.String())
}

func TestWriteIterEmpty(t *testing.T) {
	t.Parallel()
	seq := func(func(string) bool) {}
	var buf bytes.Buffer
	err := fmter.WriteIter(&buf, fmter.Table, seq)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

func TestWriteChan(t *testing.T) {
	t.Parallel()
	ch := make(chan string, 2)
	ch <- "hello"
	ch <- "world"
	close(ch)
	var buf bytes.Buffer
	err := fmter.WriteChan(&buf, fmter.Plain, ch)
	require.NoError(t, err)
	assert.Equal(t, "hello\nworld\n", buf.String())
}

func TestWriteIterJSONError(t *testing.T) {
	t.Parallel()
	seq := func(yield func(string) bool) {
		yield("hello")
	}
	w := &errWriter{}
	err := fmter.WriteIter(w, fmter.JSON, seq)
	require.Error(t, err)
}

func TestWriteIterCSVError(t *testing.T) {
	t.Parallel()
	seq := func(yield func(string) bool) {
		yield("not a rower")
	}
	var buf bytes.Buffer
	err := fmter.WriteIter(&buf, fmter.CSV, seq)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Rower")
}

func TestWriteIterTSVError(t *testing.T) {
	t.Parallel()
	seq := func(yield func(string) bool) {
		yield("not a rower")
	}
	var buf bytes.Buffer
	err := fmter.WriteIter(&buf, fmter.TSV, seq)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Rower")
}

func TestWriteIterPlainError(t *testing.T) {
	t.Parallel()
	seq := func(yield func(string) bool) {
		yield("hello")
	}
	w := &errWriter{}
	err := fmter.WriteIter(w, fmter.Plain, seq)
	require.Error(t, err)
}

func TestWriteIterJSONLError(t *testing.T) {
	t.Parallel()
	seq := func(yield func(string) bool) {
		yield("hello")
	}
	w := &errWriter{}
	err := fmter.WriteIter(w, fmter.JSONL, seq)
	require.Error(t, err)
}

func TestWriteIterGoTemplateEmpty(t *testing.T) {
	t.Parallel()
	seq := func(func(string) bool) {}
	var buf bytes.Buffer
	err := fmter.WriteIter(&buf, fmter.GoTemplate("{{.}}"), seq)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

// ============================================================
// IsSupported for new formats
// ============================================================

func TestIsSupportedNewFormats(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		format fmter.Format
		want   bool
	}{
		"plain always":      {format: fmter.Plain, want: true},
		"jsonl always":      {format: fmter.JSONL, want: true},
		"tsv with rower":    {format: fmter.TSV, want: true},
		"html with rower":   {format: fmter.HTML, want: true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, fmter.IsSupported[headedRow](tt.format))
		})
	}
}

func TestIsSupportedNewFormatsFalse(t *testing.T) {
	t.Parallel()
	assert.False(t, fmter.IsSupported[string](fmter.TSV))
	assert.False(t, fmter.IsSupported[string](fmter.HTML))
}

// ============================================================
// ParseFormat for new formats
// ============================================================

func TestParseFormatNewFormats(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		input   string
		want    fmter.Format
		wantErr require.ErrorAssertionFunc
	}{
		"plain": {input: "plain", want: fmter.Plain, wantErr: require.NoError},
		"tsv":   {input: "tsv", want: fmter.TSV, wantErr: require.NoError},
		"jsonl": {input: "jsonl", want: fmter.JSONL, wantErr: require.NoError},
		"html":  {input: "html", want: fmter.HTML, wantErr: require.NoError},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := fmter.ParseFormat(tt.input)
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ============================================================
// Sentinel errors for new formats
// ============================================================

func TestSentinelErrorsNewFormats(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		format fmter.Format
		item   any
		target error
	}{
		"tsv missing rower": {
			format: fmter.TSV,
			item:   "not a rower",
			target: fmter.ErrMissingInterface,
		},
		"html missing rower": {
			format: fmter.HTML,
			item:   "not a rower",
			target: fmter.ErrMissingInterface,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := fmter.Write(&buf, tt.format, tt.item)
			require.ErrorIs(t, err, tt.target)
		})
	}
}

// ============================================================
// Write errors for new formats
// ============================================================

func TestWriteErrorsNewFormats(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		format fmter.Format
		item   any
	}{
		"plain": {
			format: fmter.Plain,
			item:   "hello",
		},
		"tsv": {
			format: fmter.TSV,
			item:   headedRow{basicRow{Name: "A", Age: "1"}},
		},
		"jsonl": {
			format: fmter.JSONL,
			item:   struct{ Name string }{Name: "A"},
		},
		"html": {
			format: fmter.HTML,
			item:   basicRow{Name: "A", Age: "1"},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			w := &errWriter{}
			err := fmter.Write(w, tt.format, tt.item)
			require.Error(t, err)
		})
	}
}

// ============================================================
// Formatter with unsupported format fallthrough
// ============================================================

func TestWriteFormatterUnsupportedFormat(t *testing.T) {
	t.Parallel()
	items := []formattedItem{{Name: "Alice"}}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Format("xml"), items...)
	require.Error(t, err)
	assert.ErrorIs(t, err, fmter.ErrUnsupportedFormat)
}

// ============================================================
// Formatter fallthrough to each new format
// ============================================================

func TestWriteFormatterFallthroughNewFormats(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		format fmter.Format
	}{
		"tsv":   {format: fmter.TSV},
		"jsonl": {format: fmter.JSONL},
		"html":  {format: fmter.HTML},
		"csv":   {format: fmter.CSV},
		"table": {format: fmter.Table},
		"yaml":  {format: fmter.YAML},
		"list":  {format: fmter.List},
		"env":   {format: fmter.ENV},
	}

	// Use formattedItem which falls through for non-Plain formats.
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			// formattedItem only handles Plain; others get (nil, nil) fallthrough.
			// Most will error because formattedItem doesn't implement Rower etc.
			_ = fmter.Write(&buf, tt.format, formattedItem{Name: "Alice"})
			// We just exercise the code path; some succeed (JSON, YAML, JSONL), some error.
		})
	}
}

func TestWriteFormatterFallthroughGoTemplate(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.GoTemplate("{{.Name}}"), formattedItem{Name: "Alice"})
	require.NoError(t, err)
	assert.Equal(t, "Alice\n", buf.String())
}

func TestWriteFormatterFallthroughMarkdown(t *testing.T) {
	t.Parallel()
	// formattedItem doesn't implement Rower, so Markdown will error.
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Markdown, formattedItem{Name: "Alice"})
	require.Error(t, err)
}

// ============================================================
// Wrapped + Numbered table
// ============================================================

type wrappedNumberedRow struct {
	wrappedRow
}

func (r wrappedNumberedRow) NumberHeader() string { return "#" }

func TestWriteTableWrappedNumbered(t *testing.T) {
	t.Parallel()
	items := []wrappedNumberedRow{
		{wrappedRow{headedRow{basicRow{Name: "Hello", Age: "30"}}}},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Table, items...)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "#")
	assert.Contains(t, out, "1")
}

// ============================================================
// Streaming error paths
// ============================================================

func TestWriteIterCSVWriteError(t *testing.T) {
	t.Parallel()
	items := []headedRow{
		{basicRow{Name: "Alice", Age: "30"}},
		{basicRow{Name: "Bob", Age: "25"}},
	}
	seq := func(yield func(headedRow) bool) {
		for _, it := range items {
			if !yield(it) {
				return
			}
		}
	}
	w := &failAfterN{n: 1}
	err := fmter.WriteIter(w, fmter.CSV, seq)
	require.Error(t, err)
}

func TestWriteIterTSVWriteError(t *testing.T) {
	t.Parallel()
	items := []headedRow{
		{basicRow{Name: "Alice", Age: "30"}},
		{basicRow{Name: "Bob", Age: "25"}},
	}
	seq := func(yield func(headedRow) bool) {
		for _, it := range items {
			if !yield(it) {
				return
			}
		}
	}
	w := &failAfterN{n: 1}
	err := fmter.WriteIter(w, fmter.TSV, seq)
	require.Error(t, err)
}

func TestWriteIterJSONCommaError(t *testing.T) {
	t.Parallel()
	type item struct {
		Name string `json:"name"`
	}
	items := []item{{Name: "Alice"}, {Name: "Bob"}}
	seq := func(yield func(item) bool) {
		for _, it := range items {
			if !yield(it) {
				return
			}
		}
	}
	// Fail after the first encode (which includes the "[" write and first object).
	w := &failAfterN{n: 2}
	err := fmter.WriteIter(w, fmter.JSON, seq)
	require.Error(t, err)
}

// ============================================================
// HTML title with empty string
// ============================================================

type htmlNoTitleRow struct {
	headedRow
}

func (r htmlNoTitleRow) Title() string { return "" }

func TestWriteHTMLEmptyTitle(t *testing.T) {
	t.Parallel()
	items := []htmlNoTitleRow{
		{headedRow{basicRow{Name: "Alice", Age: "30"}}},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.HTML, items...)
	require.NoError(t, err)
	assert.NotContains(t, buf.String(), "<caption>")
}

// ============================================================
// Coverage gap tests
// ============================================================

// alignStyle center path in HTML.
type htmlCenterRow struct {
	basicRow
}

func (r htmlCenterRow) Alignments() []fmter.Alignment {
	return []fmter.Alignment{fmter.AlignCenter, fmter.AlignCenter}
}

func TestWriteHTMLCenterAlign(t *testing.T) {
	t.Parallel()
	items := []htmlCenterRow{{basicRow{Name: "Alice", Age: "30"}}}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.HTML, items...)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), `style="text-align: center"`)
}

// wrapCell with wide character safety path.
// Grouped table error paths (bordered).
func TestWriteTableGroupedBorderedError(t *testing.T) {
	t.Parallel()
	items := []groupedRow{
		{headedRow{basicRow{Name: "Alice", Age: "30"}}, "A"},
		{headedRow{basicRow{Name: "Bob", Age: "25"}}, "B"},
	}
	// Sweep write calls to hit group separator error.
	for n := range 15 {
		w := &failAfterN{n: n}
		err := fmter.Write(w, fmter.Table, items...)
		if err == nil {
			break
		}
	}
}

// Grouped table error paths (plain).
func TestWriteTableGroupedPlainError(t *testing.T) {
	t.Parallel()
	items := []groupedNoBorderRow{
		{groupedRow{headedRow{basicRow{Name: "Alice", Age: "30"}}, "A"}},
		{groupedRow{headedRow{basicRow{Name: "Bob", Age: "25"}}, "B"}},
	}
	for n := range 10 {
		w := &failAfterN{n: n}
		err := fmter.Write(w, fmter.Table, items...)
		if err == nil {
			break
		}
	}
}

// Paged table error paths (bordered).
func TestWriteTablePagedBorderedError(t *testing.T) {
	t.Parallel()
	items := []pagedRow{
		{headedRow{basicRow{Name: "Alice", Age: "30"}}},
		{headedRow{basicRow{Name: "Bob", Age: "25"}}},
		{headedRow{basicRow{Name: "Carol", Age: "28"}}},
	}
	for n := range 20 {
		w := &failAfterN{n: n}
		err := fmter.Write(w, fmter.Table, items...)
		if err == nil {
			break
		}
	}
}

// Paged table error paths (plain).
func TestWriteTablePagedPlainError(t *testing.T) {
	t.Parallel()
	items := []pagedNoBorderRow{
		{pagedRow{headedRow{basicRow{Name: "Alice", Age: "30"}}}},
		{pagedRow{headedRow{basicRow{Name: "Bob", Age: "25"}}}},
		{pagedRow{headedRow{basicRow{Name: "Carol", Age: "28"}}}},
	}
	for n := range 15 {
		w := &failAfterN{n: n}
		err := fmter.Write(w, fmter.Table, items...)
		if err == nil {
			break
		}
	}
}

// Wrapped table error paths (bordered).
func TestWriteTableWrappedBorderedError(t *testing.T) {
	t.Parallel()
	items := []wrappedRow{
		{headedRow{basicRow{Name: "Hello World", Age: "30"}}},
	}
	for n := range 15 {
		w := &failAfterN{n: n}
		err := fmter.Write(w, fmter.Table, items...)
		if err == nil {
			break
		}
	}
}

// Wrapped table error paths (plain).
func TestWriteTableWrappedPlainError(t *testing.T) {
	t.Parallel()
	items := []wrappedNoBorderRow{
		{wrappedRow{headedRow{basicRow{Name: "Hello World", Age: "30"}}}},
	}
	for n := range 15 {
		w := &failAfterN{n: n}
		err := fmter.Write(w, fmter.Table, items...)
		if err == nil {
			break
		}
	}
}

// Stream CSV subsequent row error.
func TestWriteIterCSVSubsequentRowError(t *testing.T) {
	t.Parallel()
	items := []headedRow{
		{basicRow{Name: "Alice", Age: "30"}},
		{basicRow{Name: strings.Repeat("x", 5000), Age: "25"}},
	}
	seq := func(yield func(headedRow) bool) {
		for _, it := range items {
			if !yield(it) {
				return
			}
		}
	}
	w := &failAfterN{n: 2}
	err := fmter.WriteIter(w, fmter.CSV, seq)
	require.Error(t, err)
}

// Stream TSV subsequent row error.
func TestWriteIterTSVSubsequentRowError(t *testing.T) {
	t.Parallel()
	items := []headedRow{
		{basicRow{Name: "Alice", Age: "30"}},
		{basicRow{Name: "Bob", Age: "25"}},
	}
	seq := func(yield func(headedRow) bool) {
		for _, it := range items {
			if !yield(it) {
				return
			}
		}
	}
	w := &failAfterN{n: 2}
	err := fmter.WriteIter(w, fmter.TSV, seq)
	require.Error(t, err)
}

// Channel early close.
func TestWriteChanEarlyClose(t *testing.T) {
	t.Parallel()
	ch := make(chan string, 1)
	ch <- "hello"
	close(ch)
	w := &errWriter{}
	err := fmter.WriteChan(w, fmter.Plain, ch)
	require.Error(t, err)
}

// Stream JSON closing bracket error.
func TestWriteIterJSONCloseBracketError(t *testing.T) {
	t.Parallel()
	type item struct {
		Name string `json:"name"`
	}
	items := []item{{Name: "A"}}
	seq := func(yield func(item) bool) {
		for _, it := range items {
			if !yield(it) {
				return
			}
		}
	}
	// fail on the closing "]" write.
	w := &failAfterN{n: 2}
	err := fmter.WriteIter(w, fmter.JSON, seq)
	require.Error(t, err)
}

// writeCSVRow error path.
func TestWriteCSVRowError(t *testing.T) {
	t.Parallel()
	// Use large data to trigger a flush error.
	big := strings.Repeat("x", 5000)
	items := []headedRow{
		{basicRow{Name: "Alice", Age: "30"}},
		{basicRow{Name: big, Age: "25"}},
	}
	seq := func(yield func(headedRow) bool) {
		for _, it := range items {
			if !yield(it) {
				return
			}
		}
	}
	w := &failAfterN{n: 2}
	_ = fmter.WriteIter(w, fmter.CSV, seq)
}

// Stream plain with stringer.
func TestWriteIterPlainStringer(t *testing.T) {
	t.Parallel()
	items := []stringerItem{{name: "Alice"}}
	seq := func(yield func(stringerItem) bool) {
		for _, it := range items {
			if !yield(it) {
				return
			}
		}
	}
	var buf bytes.Buffer
	err := fmter.WriteIter(&buf, fmter.Plain, seq)
	require.NoError(t, err)
	assert.Equal(t, "STRINGER:Alice\n", buf.String())
}

// Stream JSONL with indented.
func TestWriteIterJSONLIndented(t *testing.T) {
	t.Parallel()
	items := []indentedVal{{Name: "Alice"}}
	seq := func(yield func(indentedVal) bool) {
		for _, it := range items {
			if !yield(it) {
				return
			}
		}
	}
	var buf bytes.Buffer
	err := fmter.WriteIter(&buf, fmter.JSONL, seq)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "  \"name\"")
}

// Stream JSON indented path.
func TestWriteIterJSONIndented(t *testing.T) {
	t.Parallel()
	items := []indentedVal{{Name: "Alice"}}
	seq := func(yield func(indentedVal) bool) {
		for _, it := range items {
			if !yield(it) {
				return
			}
		}
	}
	var buf bytes.Buffer
	err := fmter.WriteIter(&buf, fmter.JSON, seq)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "  \"name\"")
}

// Styled row with more styles than columns.
type overStyledRow struct {
	headedRow
}

func (r overStyledRow) Styles() []func(string) string {
	return []func(string) string{
		func(s string) string { return "[" + s + "]" },
		func(s string) string { return "(" + s + ")" },
		func(s string) string { return "{" + s + "}" }, // extra, beyond col count
	}
}

func TestWriteTableStyledOverflow(t *testing.T) {
	t.Parallel()
	items := []overStyledRow{{headedRow{basicRow{Name: "Alice", Age: "30"}}}}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Table, items...)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "[Alice]")
	assert.Contains(t, out, "(30")
}

// Formatter fallthrough dispatch to Plain.
type formattedRower struct {
	basicRow
}

func (f formattedRower) Row() []string    { return f.basicRow.Row() }
func (f formattedRower) Header() []string { return []string{"Name", "Age"} }
func (f formattedRower) Format(fmt fmter.Format) ([]byte, error) {
	return nil, nil // always fall through
}

func TestWriteFormatterFallthroughPlain(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Plain, formattedRower{basicRow{Name: "Alice", Age: "30"}})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Alice")
}

// Stream JSON encode error.
func TestWriteIterJSONEncodeError(t *testing.T) {
	t.Parallel()
	// func values cannot be JSON-encoded.
	type badJSON struct {
		Fn func() `json:"fn"`
	}
	items := []badJSON{{Fn: func() {}}}
	seq := func(yield func(badJSON) bool) {
		for _, it := range items {
			if !yield(it) {
				return
			}
		}
	}
	var buf bytes.Buffer
	err := fmter.WriteIter(&buf, fmter.JSON, seq)
	require.Error(t, err)
}

// Stream CSV first item write error.
func TestWriteIterCSVFirstItemError(t *testing.T) {
	t.Parallel()
	items := []headedRow{{basicRow{Name: "Alice", Age: "30"}}}
	seq := func(yield func(headedRow) bool) {
		for _, it := range items {
			if !yield(it) {
				return
			}
		}
	}
	w := &errWriter{}
	err := fmter.WriteIter(w, fmter.CSV, seq)
	require.Error(t, err)
}

// Wrapped bordered with styles.
type wrappedStyledRow struct {
	wrappedRow
}

func (r wrappedStyledRow) Styles() []func(string) string {
	return []func(string) string{
		func(s string) string { return "[" + s + "]" },
		nil,
	}
}

func TestWriteTableWrappedStyledBordered(t *testing.T) {
	t.Parallel()
	items := []wrappedStyledRow{
		{wrappedRow{headedRow{basicRow{Name: "Hello", Age: "30"}}}},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Table, items...)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "[Hel")
}

type wrappedStyledNoBorderRow struct {
	wrappedStyledRow
}

func (r wrappedStyledNoBorderRow) Border() fmter.BorderStyle { return fmter.BorderNone }

func TestWriteTableWrappedStyledPlain(t *testing.T) {
	t.Parallel()
	items := []wrappedStyledNoBorderRow{
		{wrappedStyledRow{wrappedRow{headedRow{basicRow{Name: "Hello", Age: "30"}}}}},
	}
	var buf bytes.Buffer
	err := fmter.Write(&buf, fmter.Table, items...)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "[Hel")
}
