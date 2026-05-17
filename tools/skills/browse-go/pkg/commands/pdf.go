package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// pdfArgs holds parsed arguments for the pdf command.
type pdfArgs struct {
	output           string
	format           string
	width            string
	height           string
	marginTop        string
	marginRight      string
	marginBottom     string
	marginLeft       string
	headerTemplate   string
	footerTemplate   string
	pageNumbers      bool
	tagged           bool
	outline          bool
	printBackground  bool
	preferCSSPageSize bool
	toc              bool
}

// parsePdfArgs parses the full pdf flag surface.
// Supports --from-file <json> for large payloads (make-pdf contract).
func parsePdfArgs(args []string) (*pdfArgs, error) {
	// --from-file short-circuits everything
	for i := 0; i < len(args); i++ {
		if args[i] == "--from-file" {
			if i+1 >= len(args) {
				return nil, fmt.Errorf("pdf: --from-file requires a path")
			}
			return parsePdfFromFile(args[i+1])
		}
	}

	result := &pdfArgs{
		output: filepath.Join(os.TempDir(), "browse-page.pdf"),
	}

	var margins string
	var positional []string

	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--format", "--page-size":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("pdf: %s requires a value", a)
			}
			i++
			result.format = args[i]
		case "--width":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("pdf: --width requires a value")
			}
			i++
			result.width = args[i]
		case "--height":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("pdf: --height requires a value")
			}
			i++
			result.height = args[i]
		case "--margins":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("pdf: --margins requires a value")
			}
			i++
			margins = args[i]
		case "--margin-top":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("pdf: --margin-top requires a value")
			}
			i++
			result.marginTop = args[i]
		case "--margin-right":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("pdf: --margin-right requires a value")
			}
			i++
			result.marginRight = args[i]
		case "--margin-bottom":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("pdf: --margin-bottom requires a value")
			}
			i++
			result.marginBottom = args[i]
		case "--margin-left":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("pdf: --margin-left requires a value")
			}
			i++
			result.marginLeft = args[i]
		case "--header-template":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("pdf: --header-template requires a value")
			}
			i++
			result.headerTemplate = args[i]
		case "--footer-template":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("pdf: --footer-template requires a value")
			}
			i++
			result.footerTemplate = args[i]
		case "--page-numbers":
			result.pageNumbers = true
		case "--tagged":
			result.tagged = true
		case "--outline":
			result.outline = true
		case "--print-background":
			result.printBackground = true
		case "--prefer-css-page-size":
			result.preferCSSPageSize = true
		case "--toc":
			result.toc = true
		default:
			if strings.HasPrefix(a, "--") {
				return nil, fmt.Errorf("pdf: unknown flag %s", a)
			}
			positional = append(positional, a)
		}
	}

	if len(positional) > 0 {
		result.output = positional[0]
	}

	if margins != "" {
		if result.marginTop != "" || result.marginRight != "" || result.marginBottom != "" || result.marginLeft != "" {
			return nil, fmt.Errorf("pdf: --margins is mutex with --margin-top/--margin-right/--margin-bottom/--margin-left")
		}
		result.marginTop = margins
		result.marginRight = margins
		result.marginBottom = margins
		result.marginLeft = margins
	}

	if result.format != "" && (result.width != "" || result.height != "") {
		return nil, fmt.Errorf("pdf: --format is mutex with --width/--height")
	}
	if result.pageNumbers && result.footerTemplate != "" {
		return nil, fmt.Errorf("pdf: --page-numbers is mutex with --footer-template")
	}

	return result, nil
}

func parsePdfFromFile(path string) (*pdfArgs, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("pdf: read --from-file: %w", err)
	}
	var raw struct {
		Output           string `json:"output"`
		Format           string `json:"format"`
		Width            string `json:"width"`
		Height           string `json:"height"`
		MarginTop        string `json:"marginTop"`
		MarginRight      string `json:"marginRight"`
		MarginBottom     string `json:"marginBottom"`
		MarginLeft       string `json:"marginLeft"`
		HeaderTemplate   string `json:"headerTemplate"`
		FooterTemplate   string `json:"footerTemplate"`
		PageNumbers      bool   `json:"pageNumbers"`
		Tagged           bool   `json:"tagged"`
		Outline          bool   `json:"outline"`
		PrintBackground  bool   `json:"printBackground"`
		PreferCSSPageSize bool `json:"preferCSSPageSize"`
		Toc              bool   `json:"toc"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("pdf: invalid JSON in --from-file: %w", err)
	}
	result := &pdfArgs{
		output:            raw.Output,
		format:            raw.Format,
		width:             raw.Width,
		height:            raw.Height,
		marginTop:         raw.MarginTop,
		marginRight:       raw.MarginRight,
		marginBottom:      raw.MarginBottom,
		marginLeft:        raw.MarginLeft,
		headerTemplate:    raw.HeaderTemplate,
		footerTemplate:    raw.FooterTemplate,
		pageNumbers:       raw.PageNumbers,
		tagged:            raw.Tagged,
		outline:           raw.Outline,
		printBackground:   raw.PrintBackground,
		preferCSSPageSize: raw.PreferCSSPageSize,
		toc:               raw.Toc,
	}
	if result.output == "" {
		result.output = filepath.Join(os.TempDir(), "browse-page.pdf")
	}
	return result, nil
}

// parseDimension converts a dimension string to inches (float64).
// Supports: "1in", "72pt", "25mm", "2.54cm", or bare number (pixels, /96).
func parseDimension(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty dimension")
	}

	// Try unit suffixes
	lower := strings.ToLower(s)
	if strings.HasSuffix(lower, "in") {
		v, err := strconv.ParseFloat(strings.TrimSuffix(lower, "in"), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid dimension %q", s)
		}
		return v, nil
	}
	if strings.HasSuffix(lower, "pt") {
		v, err := strconv.ParseFloat(strings.TrimSuffix(lower, "pt"), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid dimension %q", s)
		}
		return v / 72.0, nil
	}
	if strings.HasSuffix(lower, "mm") {
		v, err := strconv.ParseFloat(strings.TrimSuffix(lower, "mm"), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid dimension %q", s)
		}
		return v / 25.4, nil
	}
	if strings.HasSuffix(lower, "cm") {
		v, err := strconv.ParseFloat(strings.TrimSuffix(lower, "cm"), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid dimension %q", s)
		}
		return v / 2.54, nil
	}

	// Bare number: treat as pixels (96 DPI is the web/CSS default)
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid dimension %q", s)
	}
	return v / 96.0, nil
}

// buildPrintToPDFParams builds chromedp params from parsed args.
func buildPrintToPDFParams(args *pdfArgs) (*page.PrintToPDFParams, error) {
	params := page.PrintToPDF()

	// Paper size
	switch args.format {
	case "letter":
		params = params.WithPaperWidth(8.5).WithPaperHeight(11)
	case "legal":
		params = params.WithPaperWidth(8.5).WithPaperHeight(14)
	case "a4", "":
		params = params.WithPaperWidth(8.27).WithPaperHeight(11.7)
	default:
		return nil, fmt.Errorf("pdf: unknown format %q (use letter, a4, legal)", args.format)
	}

	if args.width != "" {
		w, err := parseDimension(args.width)
		if err != nil {
			return nil, fmt.Errorf("pdf: width: %w", err)
		}
		params = params.WithPaperWidth(w)
	}
	if args.height != "" {
		h, err := parseDimension(args.height)
		if err != nil {
			return nil, fmt.Errorf("pdf: height: %w", err)
		}
		params = params.WithPaperHeight(h)
	}

	// Margins
	if args.marginTop != "" {
		v, err := parseDimension(args.marginTop)
		if err != nil {
			return nil, fmt.Errorf("pdf: margin-top: %w", err)
		}
		params = params.WithMarginTop(v)
	}
	if args.marginBottom != "" {
		v, err := parseDimension(args.marginBottom)
		if err != nil {
			return nil, fmt.Errorf("pdf: margin-bottom: %w", err)
		}
		params = params.WithMarginBottom(v)
	}
	if args.marginLeft != "" {
		v, err := parseDimension(args.marginLeft)
		if err != nil {
			return nil, fmt.Errorf("pdf: margin-left: %w", err)
		}
		params = params.WithMarginLeft(v)
	}
	if args.marginRight != "" {
		v, err := parseDimension(args.marginRight)
		if err != nil {
			return nil, fmt.Errorf("pdf: margin-right: %w", err)
		}
		params = params.WithMarginRight(v)
	}

	// Header/footer
	if args.headerTemplate != "" {
		params = params.WithHeaderTemplate(args.headerTemplate)
		params = params.WithDisplayHeaderFooter(true)
	}
	if args.footerTemplate != "" {
		params = params.WithFooterTemplate(args.footerTemplate)
		params = params.WithDisplayHeaderFooter(true)
	}
	if args.pageNumbers {
		params = params.WithFooterTemplate(`<div style="font-size:9px;width:100%;text-align:center;margin-top:5px;"><span class="pageNumber"></span> / <span class="totalPages"></span></div>`)
		params = params.WithDisplayHeaderFooter(true)
	}

	// Other flags
	if args.printBackground {
		params = params.WithPrintBackground(true)
	}
	if args.preferCSSPageSize {
		params = params.WithPreferCSSPageSize(true)
	}
	if args.tagged {
		params = params.WithGenerateTaggedPDF(true)
	}
	if args.outline {
		params = params.WithGenerateDocumentOutline(true)
	}

	// TOC is handled by injectToc/cleanupToc in handlePdf before/after printing.

	return params, nil
}

// handlePdf is the command handler registered in registerMeta.
func handlePdf(ctx *ExecContext) (string, error) {
	args, err := parsePdfArgs(ctx.Args)
	if err != nil {
		return "", err
	}

	params, err := buildPrintToPDFParams(args)
	if err != nil {
		return "", err
	}

	// If --toc is requested, inject a table of contents into the page before printing.
	if args.toc {
		if err := injectToc(ctx.Session.Context()); err != nil {
			return "", fmt.Errorf("toc injection failed: %w", err)
		}
		// Schedule cleanup after printing
		defer cleanupToc(ctx.Session.Context())
	}

	var buf []byte
	if err := chromedp.Run(ctx.Session.Context(), chromedp.ActionFunc(func(c context.Context) error {
		var err error
		buf, _, err = params.Do(c)
		return err
	})); err != nil {
		return "", fmt.Errorf("pdf failed: %w", err)
	}

	if err := os.WriteFile(args.output, buf, 0644); err != nil {
		return "", fmt.Errorf("write pdf failed: %w", err)
	}

	msg := "PDF saved to " + args.output
	if args.toc {
		msg += "\nTable of contents included."
	}
	return msg, nil
}

// tocData holds extracted heading information.
type tocData struct {
	Level int    `json:"level"`
	Text  string `json:"text"`
	ID    string `json:"id"`
}

// injectToc extracts headings from the current page and injects a
// table-of-contents block at the top of <body>.
func injectToc(c context.Context) error {
	script := `
		(() => {
			// 1. Extract headings
			const headings = Array.from(document.querySelectorAll('h1, h2, h3, h4, h5, h6'));
			const tocItems = [];
			headings.forEach((h, idx) => {
				if (!h.id) {
					h.id = '__browse-toc-h-' + idx;
				}
				tocItems.push({
					level: parseInt(h.tagName[1]),
					text: h.textContent.trim().replace(/\s+/g, ' '),
					id: h.id
				});
			});

			if (tocItems.length === 0) return {ok: true, count: 0};

			// 2. Build TOC HTML
			const tocBlock = document.createElement('div');
			tocBlock.id = '__browse-toc-block';
			tocBlock.style.cssText = 'page-break-after:always;margin-bottom:2em;padding:1.5em;border:1px solid #ddd;background:#fafafa;font-family:system-ui,-apple-system,sans-serif;';

			const title = document.createElement('h2');
			title.textContent = 'Table of Contents';
			title.style.cssText = 'margin-top:0;margin-bottom:1em;font-size:1.4em;border-bottom:2px solid #333;padding-bottom:0.3em;';
			tocBlock.appendChild(title);

			const list = document.createElement('ul');
			list.style.cssText = 'list-style:none;padding:0;margin:0;';
			tocItems.forEach(item => {
				const li = document.createElement('li');
				const indent = (item.level - 1) * 1.5;
				li.style.cssText = 'margin:0.35em 0;padding-left:' + indent + 'em;';
				const a = document.createElement('a');
				a.href = '#' + item.id;
				a.textContent = item.text;
				a.style.cssText = 'text-decoration:none;color:#0366d6;';
				li.appendChild(a);
				list.appendChild(li);
			});
			tocBlock.appendChild(list);

			// 3. Inject at top of body
			document.body.insertBefore(tocBlock, document.body.firstChild);

			return {ok: true, count: tocItems.length};
		})()
	`
	var result struct {
		OK    bool `json:"ok"`
		Count int  `json:"count"`
	}
	if err := chromedp.Run(c, chromedp.Evaluate(script, &result)); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("toc injection script failed")
	}
	return nil
}

// cleanupToc removes the injected TOC block from the page.
func cleanupToc(c context.Context) error {
	script := `
		(() => {
			const toc = document.getElementById('__browse-toc-block');
			if (toc) toc.remove();
			return true;
		})()
	`
	var ok bool
	return chromedp.Run(c, chromedp.Evaluate(script, &ok))
}
