package security

import (
	"context"
	"fmt"
	"strings"

	"github.com/chromedp/chromedp"
)

// ARIAInjectionPatterns are regex patterns that detect injection attempts in ARIA labels.
var ARIAInjectionPatterns = []string{
	`ignore\s+(previous|above|all)\s+instructions?`,
	`you\s+are\s+(now|a)\s+`,
	`system\s*:\s*`,
	`\bdo\s+not\s+(follow|obey|listen)`,
	`\bexecute\s+(the\s+)?following`,
	`\bforget\s+(everything|all|your)`,
	`\bnew\s+instructions?\s*:`,
}

// markHiddenElementsJS is the JavaScript that marks hidden elements on a page.
// It sets data-gstack-hidden="true" on elements matching the hidden criteria.
const markHiddenElementsJS = `
(function(ariaPatterns) {
// PLACEHOLDER: %%ARIA_PATTERNS%%
	const found = [];
	const elements = document.querySelectorAll('body *');
	const ariaRegexes = ariaPatterns.map(p => new RegExp(p, 'i'));

	for (const el of elements) {
		if (!(el instanceof HTMLElement)) continue;
		const style = window.getComputedStyle(el);
		const text = (el.textContent || '').trim();
		if (!text) continue;

		let isHidden = false;
		let reason = '';

		if (parseFloat(style.opacity) < 0.1) {
			isHidden = true;
			reason = 'opacity < 0.1';
		} else if (parseFloat(style.fontSize) < 1) {
			isHidden = true;
			reason = 'font-size < 1px';
		} else if (style.position === 'absolute' || style.position === 'fixed') {
			const rect = el.getBoundingClientRect();
			if (rect.right < -100 || rect.bottom < -100 || rect.left > window.innerWidth + 100 || rect.top > window.innerHeight + 100) {
				isHidden = true;
				reason = 'off-screen';
			}
		} else if (style.color === style.backgroundColor && text.length > 10) {
			isHidden = true;
			reason = 'same fg/bg color';
		} else if (style.clipPath === 'inset(100%)' || style.clip === 'rect(0px, 0px, 0px, 0px)') {
			isHidden = true;
			reason = 'clip hiding';
		} else if (style.visibility === 'hidden') {
			isHidden = true;
			reason = 'visibility hidden';
		}

		if (isHidden) {
			el.setAttribute('data-gstack-hidden', 'true');
			found.push('[' + el.tagName.toLowerCase() + '] ' + reason + ': "' + text.slice(0, 60) + '..."');
		}

		const ariaLabel = el.getAttribute('aria-label') || '';
		const ariaLabelledBy = el.getAttribute('aria-labelledby');
		let labelText = ariaLabel;
		if (ariaLabelledBy) {
			const labelEl = document.getElementById(ariaLabelledBy);
			if (labelEl) labelText += ' ' + (labelEl.textContent || '');
		}
		if (labelText) {
			for (const re of ariaRegexes) {
				if (re.test(labelText)) {
					el.setAttribute('data-gstack-hidden', 'true');
					found.push('[' + el.tagName.toLowerCase() + '] ARIA injection: "' + labelText.slice(0, 60) + '..."');
					break;
				}
			}
		}
	}
	return found;
})(%%ARIA_PATTERNS%%)
`

// getCleanTextJS extracts clean innerText after removing hidden elements.
const getCleanTextJS = `
(function() {
	const body = document.body;
	if (!body) return '';
	const clone = body.cloneNode(true);
	clone.querySelectorAll('script, style, noscript, svg').forEach(el => el.remove());
	clone.querySelectorAll('[data-gstack-hidden]').forEach(el => el.remove());
	return clone.innerText
		.split('\n')
		.map(line => line.trim())
		.filter(line => line.length > 0)
		.join('\n');
})()
`

// cleanupHiddenMarkersJS removes data-gstack-hidden attributes.
const cleanupHiddenMarkersJS = `
(function() {
	document.querySelectorAll('[data-gstack-hidden]').forEach(el => {
		el.removeAttribute('data-gstack-hidden');
	});
})()
`

// MarkHiddenElements detects hidden elements on the current page and marks them.
// Returns descriptions of what was found for logging.
func MarkHiddenElements(ctx context.Context) ([]string, error) {
	patternsJSON := "[" + strings.Join(ariaPatternsToJSON(), ",") + "]"
	// Use strings.Replace instead of fmt.Sprintf to avoid interpreting JS % as format verbs.
	js := strings.Replace(markHiddenElementsJS, "%%ARIA_PATTERNS%%", patternsJSON, 1)

	var found []string
	if err := chromedp.Evaluate(js, &found).Do(ctx); err != nil {
		return nil, fmt.Errorf("mark hidden elements: %w", err)
	}
	return found, nil
}

// GetCleanText extracts clean text with hidden elements stripped.
func GetCleanText(ctx context.Context) (string, error) {
	var text string
	if err := chromedp.Evaluate(getCleanTextJS, &text).Do(ctx); err != nil {
		return "", fmt.Errorf("get clean text: %w", err)
	}
	return stripLoneSurrogates(text), nil
}

// CleanupHiddenMarkers removes data-gstack-hidden attributes from the page.
func CleanupHiddenMarkers(ctx context.Context) error {
	return chromedp.Evaluate(cleanupHiddenMarkersJS, nil).Do(ctx)
}

// ariaPatternsToJSON converts ARIA patterns to JSON string array elements.
func ariaPatternsToJSON() []string {
	var parts []string
	for _, p := range ARIAInjectionPatterns {
		parts = append(parts, fmt.Sprintf("%q", p))
	}
	return parts
}

// stripLoneSurrogates removes invalid UTF-16 surrogate pairs that can leak
// through from browser JS execution.
func stripLoneSurrogates(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 0xD800 && r <= 0xDFFF {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
