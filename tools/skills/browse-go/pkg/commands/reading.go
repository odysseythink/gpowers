package commands

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"browse-go/pkg/security"

	"github.com/chromedp/chromedp"
)

func (r *Registry) registerReading() {
	r.Register("text", CommandDesc{Category: "Reading", Description: "Cleaned page text"},
		func(ctx *ExecContext) (string, error) {
			// L2: Hidden element stripping — mark hidden elements before extraction.
			found, err := security.MarkHiddenElements(ctx.Session.Context())
			if err != nil {
				found = nil
			}
			var text string
			if err := chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(`
				(() => {
					const b = document.body;
					if (!b) return '';
					const c = b.cloneNode(true);
					c.querySelectorAll('script,style,noscript,svg').forEach(e => e.remove());
					c.querySelectorAll('[data-gstack-hidden]').forEach(e => e.remove());
					return Array.from(c.innerText.split('\n')).map(l => l.trim()).filter(l => l.length > 0).join('\n');
				})()
			`, &text)); err != nil {
				return "", fmt.Errorf("text extraction failed: %w", err)
			}
			_ = security.CleanupHiddenMarkers(ctx.Session.Context())
			result := stripLoneSurrogates(text)
			if len(found) > 0 {
				result = fmt.Sprintf("<!-- %d hidden elements stripped -->\n%s", len(found), result)
			}
			return result, nil
		})

	r.Register("html", CommandDesc{Category: "Reading", Description: "innerHTML of selector or full page", Usage: "html [selector]"},
		func(ctx *ExecContext) (string, error) {
			var result string
			if len(ctx.Args) > 0 {
				sel, err := ctx.Session.ResolveRef(ctx.Args[0])
				if err != nil {
					return "", err
				}
				if err := chromedp.Run(ctx.Session.Context(), chromedp.InnerHTML(sel, &result)); err != nil {
					return "", fmt.Errorf("html failed: %w", err)
				}
			} else {
				if err := chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(`
					(() => {
						const dt = document.doctype;
						const pre = dt ? '<!DOCTYPE ' + dt.name + '>\n' : '';
						return pre + document.documentElement.outerHTML;
					})()
				`, &result)); err != nil {
					return "", fmt.Errorf("html failed: %w", err)
				}
			}
			return stripLoneSurrogates(result), nil
		})

	r.Register("links", CommandDesc{Category: "Reading", Description: "All links as text → href"},
		func(ctx *ExecContext) (string, error) {
			var links []struct {
				Text string `json:"text"`
				Href string `json:"href"`
			}
			if err := chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(`
				[...document.querySelectorAll('a[href]')].map(a => ({
					text: (a.textContent || '').trim().slice(0,120),
					href: a.href
				})).filter(l => l.text && l.href)
			`, &links)); err != nil {
				return "", fmt.Errorf("links failed: %w", err)
			}
			lines := make([]string, len(links))
			for i, l := range links {
				lines[i] = fmt.Sprintf("%s → %s", l.Text, l.Href)
			}
			return strings.Join(lines, "\n"), nil
		})

	r.Register("forms", CommandDesc{Category: "Reading", Description: "Form fields as JSON"},
		func(ctx *ExecContext) (string, error) {
			var forms []map[string]interface{}
			if err := chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(`
				[...document.querySelectorAll('form')].map((form, i) => {
					const fields = [...form.querySelectorAll('input,select,textarea')].map(el => {
						const o = {
							tag: el.tagName.toLowerCase(),
							name: el.name || '',
							id: el.id || '',
							type: el.type || undefined,
							value: el.value || '',
							placeholder: el.placeholder || '',
							required: el.required || false
						};
						if (el.tagName === 'SELECT') {
							o.options = [...el.options].map(opt => ({value: opt.value, text: opt.text}));
						}
						return o;
					});
					return {index: i, action: form.action || '', method: form.method || 'get', fields};
				})
			`, &forms)); err != nil {
				return "", fmt.Errorf("forms failed: %w", err)
			}
			b, _ := json.MarshalIndent(forms, "", "  ")
			return string(b), nil
		})

	r.Register("accessibility", CommandDesc{Category: "Reading", Description: "Full ARIA tree"},
		func(ctx *ExecContext) (string, error) {
			var tree string
			if err := chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(`
				(() => {
					function walk(node, depth) {
						const role = node.getAttribute('role') || node.tagName.toLowerCase();
						const name = node.getAttribute('aria-label') || node.textContent?.trim().slice(0,60) || '';
						let line = '  '.repeat(depth) + role;
						if (name) line += ': ' + name;
						const out = [line];
						for (const child of node.children) {
							if (child.getAttribute('role') || ['BUTTON','A','INPUT','SELECT','TEXTAREA'].includes(child.tagName)) {
								out.push(...walk(child, depth+1));
							}
						}
						return out;
					}
					return walk(document.body, 0).join('\n');
				})()
			`, &tree)); err != nil {
				return "", fmt.Errorf("accessibility failed: %w", err)
			}
			return tree, nil
		})

	r.Register("is", CommandDesc{Category: "Reading", Description: "Element state assertion", Usage: "is <visible|hidden|enabled|disabled|checked|editable|focused> <sel|@ref>"},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) < 2 {
				return "", fmt.Errorf("usage: is <property> <sel|@ref>\nProperties: visible, hidden, enabled, disabled, checked, editable, focused")
			}
			property := ctx.Args[0]
			sel, err := ctx.Session.ResolveRef(ctx.Args[1])
			if err != nil {
				return "", err
			}

			var result bool
			script := fmt.Sprintf(`
				(() => {
					const el = document.querySelector(%s);
					if (!el) return false;
					switch (%q) {
						case 'visible': {
							const r = el.getBoundingClientRect();
							return r.width > 0 && r.height > 0 && r.bottom > 0 && r.right > 0 &&
								window.getComputedStyle(el).visibility !== 'hidden' &&
								window.getComputedStyle(el).display !== 'none';
						}
						case 'hidden': {
							const r = el.getBoundingClientRect();
							return r.width === 0 || r.height === 0 ||
								window.getComputedStyle(el).visibility === 'hidden' ||
								window.getComputedStyle(el).display === 'none';
						}
						case 'enabled': return !el.disabled;
						case 'disabled': return !!el.disabled;
						case 'checked': return !!el.checked;
						case 'editable': {
							if (el.contentEditable === 'true') return true;
							return ['INPUT','TEXTAREA','SELECT'].includes(el.tagName) &&
								!el.disabled && !el.readOnly;
						}
						case 'focused': return el === document.activeElement;
						default: return false;
					}
				})()
			`, strconv.Quote(sel), property)

			if err := chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(script, &result)); err != nil {
				return "", fmt.Errorf("is %s failed: %w", property, err)
			}
			return fmt.Sprintf("%t", result), nil
		})

	r.Register("data", CommandDesc{Category: "Reading", Description: "Extract structured data", Usage: "data [--jsonld] [--og] [--twitter] [--meta]"},
		func(ctx *ExecContext) (string, error) {
			wantJsonLd := false
			wantOg := false
			wantTwitter := false
			wantMeta := false
			for _, a := range ctx.Args {
				switch a {
				case "--jsonld":
					wantJsonLd = true
				case "--og":
					wantOg = true
				case "--twitter":
					wantTwitter = true
				case "--meta":
					wantMeta = true
				}
			}
			// Default: all categories
			if !wantJsonLd && !wantOg && !wantTwitter && !wantMeta {
				wantJsonLd, wantOg, wantTwitter, wantMeta = true, true, true, true
			}

			var result map[string]interface{}
			script := fmt.Sprintf(`
				(() => {
					const data = {};
					if (%v) {
						const jsonLd = [];
						document.querySelectorAll('script[type="application/ld+json"]').forEach(s => {
							try { jsonLd.push(JSON.parse(s.textContent || '')); } catch {}
						});
						data.jsonLd = jsonLd;
					}
					if (%v) {
						const og = {};
						document.querySelectorAll('meta[property^="og:"]').forEach(m => {
							const prop = (m.getAttribute('property') || '').replace('og:', '');
							og[prop] = m.getAttribute('content') || '';
						});
						data.openGraph = og;
					}
					if (%v) {
						const tw = {};
						document.querySelectorAll('meta[name^="twitter:"]').forEach(m => {
							const name = (m.getAttribute('name') || '').replace('twitter:', '');
							tw[name] = m.getAttribute('content') || '';
						});
						data.twitterCards = tw;
					}
					if (%v) {
						const meta = {};
						const canonical = document.querySelector('link[rel="canonical"]');
						if (canonical) meta.canonical = canonical.getAttribute('href') || '';
						const desc = document.querySelector('meta[name="description"]');
						if (desc) meta.description = desc.getAttribute('content') || '';
						const keywords = document.querySelector('meta[name="keywords"]');
						if (keywords) meta.keywords = keywords.getAttribute('content') || '';
						const author = document.querySelector('meta[name="author"]');
						if (author) meta.author = author.getAttribute('content') || '';
						const title = document.querySelector('title');
						if (title) meta.title = title.textContent || '';
						data.meta = meta;
					}
					return data;
				})()
			`, wantJsonLd, wantOg, wantTwitter, wantMeta)

			if err := chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(script, &result)); err != nil {
				return "", fmt.Errorf("data extraction failed: %w", err)
			}
			b, _ := json.MarshalIndent(result, "", "  ")
			return string(b), nil
		})

	r.Register("media", CommandDesc{Category: "Reading", Description: "Extract media elements", Usage: "media [--images] [--videos] [--audio] [selector]"},
		func(ctx *ExecContext) (string, error) {
			filter := ""
			selector := ""
			for _, a := range ctx.Args {
				switch a {
				case "--images":
					filter = "images"
				case "--videos":
					filter = "videos"
				case "--audio":
					filter = "audio"
				default:
					if !startsWithDash(a) && selector == "" {
						selector = a
					}
				}
			}

			var result map[string]interface{}
			script := fmt.Sprintf(`
				(() => {
					const root = %s ? document.querySelector(%s) || document : document;
					const filter = %q;
					const images = [], videos = [], audio = [], backgroundImages = [];

					if (!filter || filter === 'images') {
						root.querySelectorAll('img').forEach((el, i) => {
							const r = el.getBoundingClientRect();
							images.push({
								index: i, src: el.src, srcset: el.srcset || '', currentSrc: el.currentSrc || '',
								alt: el.alt || '', width: el.width, height: el.height,
								naturalWidth: el.naturalWidth, naturalHeight: el.naturalHeight,
								loading: el.loading || '', dataSrc: el.dataset.src || el.dataset.lazySrc || el.dataset.original || '',
								visible: r.width > 0 && r.height > 0 && r.bottom > 0 && r.right > 0
							});
						});
						const allEls = root.querySelectorAll('*');
						for (let i = 0; i < Math.min(allEls.length, 500); i++) {
							const style = window.getComputedStyle(allEls[i]);
							const bg = style.backgroundImage;
							if (bg && bg !== 'none') {
								const m = bg.match(/url\(["']?([^"')]+)["']?\)/);
								if (m && m[1] && !m[1].startsWith('data:')) {
									backgroundImages.push({index: backgroundImages.length, url: m[1], selector: allEls[i].tagName.toLowerCase() + (allEls[i].id ? '#' + allEls[i].id : '')});
								}
							}
						}
					}
					if (!filter || filter === 'videos') {
						root.querySelectorAll('video').forEach((el, i) => {
							const sources = [...el.querySelectorAll('source')].map(s => ({src: s.src, type: s.type || ''}));
							const isHLS = sources.some(s => s.src.includes('.m3u8') || s.type.includes('mpegURL'));
							const isDASH = sources.some(s => s.src.includes('.mpd') || s.type.includes('dash'));
							videos.push({
								index: i, src: el.src || '', currentSrc: el.currentSrc || '', poster: el.poster || '',
								width: el.videoWidth || el.width, height: el.videoHeight || el.height,
								duration: el.duration || 0, type: el.type || '', sources, isHLS, isDASH
							});
						});
					}
					if (!filter || filter === 'audio') {
						root.querySelectorAll('audio').forEach((el, i) => {
							audio.push({
								index: i, src: el.src || '', currentSrc: el.currentSrc || '',
								duration: el.duration || 0, type: el.type || ''
							});
						});
					}
					return {images, videos, audio, backgroundImages, total: images.length + videos.length + audio.length + backgroundImages.length};
				})()
			`, strconv.Quote(selector), strconv.Quote(selector), filter)

			if err := chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(script, &result)); err != nil {
				return "", fmt.Errorf("media extraction failed: %w", err)
			}
			b, _ := json.MarshalIndent(result, "", "  ")
			return string(b), nil
		})

	r.Register("inspect", CommandDesc{Category: "Reading", Description: "Inspect element details", Usage: "inspect <sel|@ref> [--all]"},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) == 0 {
				return "", fmt.Errorf("usage: inspect <sel|@ref> [--all]")
			}
			allStyles := false
			selector := ""
			for _, a := range ctx.Args {
				if a == "--all" {
					allStyles = true
				} else if selector == "" {
					selector = a
				}
			}
			sel, err := ctx.Session.ResolveRef(selector)
			if err != nil {
				return "", err
			}

			var result map[string]interface{}
			script := fmt.Sprintf(`
				(() => {
					const el = document.querySelector(%s);
					if (!el) return null;
					const rect = el.getBoundingClientRect();
					const computed = window.getComputedStyle(el);
					const styles = {};
					const keys = %v ? Array.from(computed) : [
						'display','position','width','height','top','left','margin','padding',
						'border','background-color','color','font-size','font-family','z-index',
						'opacity','visibility','overflow','flex-direction','grid-template-columns'
					];
					keys.forEach(k => { styles[k] = computed.getPropertyValue(k); });
					return {
						tag: el.tagName.toLowerCase(),
						id: el.id || undefined,
						class: el.className || undefined,
						selector: %s,
						dimensions: {width: rect.width, height: rect.height, x: rect.x, y: rect.y},
						inlineStyles: el.getAttribute('style') || undefined,
						computedStyles: styles,
						innerHTMLPreview: el.innerHTML.slice(0, 500)
					};
				})()
			`, strconv.Quote(sel), allStyles, strconv.Quote(sel))

			if err := chromedp.Run(ctx.Session.Context(), chromedp.Evaluate(script, &result)); err != nil {
				return "", fmt.Errorf("inspect failed: %w", err)
			}
			if result == nil {
				return "", fmt.Errorf("element not found: %s", sel)
			}
			b, _ := json.MarshalIndent(result, "", "  ")
			return string(b), nil
		})
}

// stripLoneSurrogates removes unpaired UTF-16 surrogates to prevent
// downstream JSON consumers from rejecting the payload.
func stripLoneSurrogates(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 0xD800 && r <= 0xDBFF {
			// High surrogate — check next rune
			if i+1 < len(s) {
				nextRune := rune(s[i+1])
				if nextRune >= 0xDC00 && nextRune <= 0xDFFF {
					b.WriteRune(r)
					continue
				}
			}
			b.WriteRune('\uFFFD')
		} else if r >= 0xDC00 && r <= 0xDFFF {
			// Low surrogate — check prev rune
			if i > 0 {
				prevRune := rune(s[i-1])
				if prevRune >= 0xD800 && prevRune <= 0xDBFF {
					b.WriteRune(r)
					continue
				}
			}
			b.WriteRune('\uFFFD')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
