package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParsePdfArgsBasic(t *testing.T) {
	args, err := parsePdfArgs([]string{"/tmp/my.pdf", "--format", "letter"})
	if err != nil {
		t.Fatal(err)
	}
	if args.output != "/tmp/my.pdf" {
		t.Errorf("output = %q", args.output)
	}
	if args.format != "letter" {
		t.Errorf("format = %q", args.format)
	}
}

func TestParsePdfArgsDefaults(t *testing.T) {
	args, err := parsePdfArgs([]string{})
	if err != nil {
		t.Fatal(err)
	}
	if args.output == "" {
		t.Error("expected default output path")
	}
	if args.format != "" {
		t.Errorf("expected empty format, got %q", args.format)
	}
}

func TestParsePdfArgsMutexFormatWidth(t *testing.T) {
	_, err := parsePdfArgs([]string{"--format", "a4", "--width", "5in"})
	if err == nil {
		t.Error("expected mutex error for format+width")
	}
}

func TestParsePdfArgsMutexMargins(t *testing.T) {
	_, err := parsePdfArgs([]string{"--margins", "1in", "--margin-top", "0.5in"})
	if err == nil {
		t.Error("expected mutex error for margins+margin-top")
	}
}

func TestParsePdfArgsMutexPageNumbersFooter(t *testing.T) {
	_, err := parsePdfArgs([]string{"--page-numbers", "--footer-template", "<div>x</div>"})
	if err == nil {
		t.Error("expected mutex error for page-numbers+footer-template")
	}
}

func TestParsePdfArgsUnknownFlag(t *testing.T) {
	_, err := parsePdfArgs([]string{"--unknown-flag"})
	if err == nil {
		t.Error("expected error for unknown flag")
	}
}

func TestParsePdfFromFile(t *testing.T) {
	tmp := t.TempDir()
	payload := map[string]interface{}{
		"output":            filepath.Join(tmp, "out.pdf"),
		"format":            "legal",
		"width":             "6in",
		"marginTop":         "0.5in",
		"pageNumbers":       true,
		"printBackground":   true,
		"preferCSSPageSize": true,
		"toc":               true,
	}
	data, _ := json.Marshal(payload)
	path := filepath.Join(tmp, "payload.json")
	os.WriteFile(path, data, 0644)

	args, err := parsePdfArgs([]string{"--from-file", path})
	if err != nil {
		t.Fatal(err)
	}
	if args.output != filepath.Join(tmp, "out.pdf") {
		t.Errorf("output = %q", args.output)
	}
	if args.format != "legal" {
		t.Errorf("format = %q", args.format)
	}
	if args.marginTop != "0.5in" {
		t.Errorf("marginTop = %q", args.marginTop)
	}
	if !args.pageNumbers {
		t.Error("expected pageNumbers=true")
	}
	if !args.printBackground {
		t.Error("expected printBackground=true")
	}
	if !args.toc {
		t.Error("expected toc=true")
	}
}

func TestParseDimension(t *testing.T) {
	cases := []struct {
		input string
		want  float64
	}{
		{"1in", 1.0},
		{"72pt", 1.0},
		{"25.4mm", 1.0},
		{"2.54cm", 1.0},
		{"96", 1.0}, // 96 pixels = 1 inch
		{"192", 2.0},
	}
	for _, c := range cases {
		got, err := parseDimension(c.input)
		if err != nil {
			t.Fatalf("parseDimension(%q): %v", c.input, err)
		}
		if got < c.want-0.001 || got > c.want+0.001 {
			t.Errorf("parseDimension(%q) = %f, want %f", c.input, got, c.want)
		}
	}
}

func TestParseDimensionInvalid(t *testing.T) {
	_, err := parseDimension("abc")
	if err == nil {
		t.Error("expected error for invalid dimension")
	}
	_, err = parseDimension("")
	if err == nil {
		t.Error("expected error for empty dimension")
	}
}

func TestBuildPrintToPDFParams(t *testing.T) {
	args := &pdfArgs{
		format:           "a4",
		marginTop:        "1in",
		marginBottom:     "1in",
		marginLeft:       "0.5in",
		marginRight:      "0.5in",
		headerTemplate:   "<div>Header</div>",
		footerTemplate:   "<div>Footer</div>",
		printBackground:  true,
		preferCSSPageSize: true,
		tagged:           true,
		outline:          true,
	}
	params, err := buildPrintToPDFParams(args)
	if err != nil {
		t.Fatal(err)
	}
	if params == nil {
		t.Fatal("expected params")
	}
	// Paper width for A4
	if params.PaperWidth < 8.26 || params.PaperWidth > 8.28 {
		t.Errorf("paper width = %f", params.PaperWidth)
	}
	if !params.DisplayHeaderFooter {
		t.Error("expected DisplayHeaderFooter=true")
	}
	if !params.PrintBackground {
		t.Error("expected PrintBackground=true")
	}
	if !params.PreferCSSPageSize {
		t.Error("expected PreferCSSPageSize=true")
	}
	if !params.GenerateTaggedPDF {
		t.Error("expected GenerateTaggedPDF=true")
	}
	if !params.GenerateDocumentOutline {
		t.Error("expected GenerateDocumentOutline=true")
	}
}

func TestBuildPrintToPDFParamsPageNumbers(t *testing.T) {
	args := &pdfArgs{
		format:      "letter",
		pageNumbers: true,
	}
	params, err := buildPrintToPDFParams(args)
	if err != nil {
		t.Fatal(err)
	}
	if !params.DisplayHeaderFooter {
		t.Error("expected DisplayHeaderFooter=true for page numbers")
	}
	if params.FooterTemplate == "" {
		t.Error("expected FooterTemplate set for page numbers")
	}
}

func TestBuildPrintToPDFParamsCustomSize(t *testing.T) {
	args := &pdfArgs{
		width:  "5in",
		height: "7in",
	}
	params, err := buildPrintToPDFParams(args)
	if err != nil {
		t.Fatal(err)
	}
	if params.PaperWidth != 5.0 {
		t.Errorf("width = %f", params.PaperWidth)
	}
	if params.PaperHeight != 7.0 {
		t.Errorf("height = %f", params.PaperHeight)
	}
}

func TestBuildPrintToPDFParamsUnknownFormat(t *testing.T) {
	args := &pdfArgs{format: "b5"}
	_, err := buildPrintToPDFParams(args)
	if err == nil {
		t.Error("expected error for unknown format")
	}
}
