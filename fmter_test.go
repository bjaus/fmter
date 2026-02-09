package fmter_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/bjaus/fmter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		fmter.Markdown, fmter.List, fmter.ENV,
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
