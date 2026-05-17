// Package inspector implements CDP-based CSS inspection and live modification.
//
// Provides:
//   - Full CSS rule cascade inspection (matched rules, computed styles, inline styles)
//   - Box model measurement
//   - Live CSS modification via CSS.setStyleTexts with inline fallback
//   - Modification history with undo/reset
package inspector

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/css"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
)

// ─── Types ──────────────────────────────────────────────────

// Result holds the full inspection output for an element.
type Result struct {
	Selector     string            `json:"selector"`
	TagName      string            `json:"tagName"`
	ID           string            `json:"id,omitempty"`
	Classes      []string          `json:"classes,omitempty"`
	Attributes   map[string]string `json:"attributes,omitempty"`
	BoxModel     BoxModel          `json:"boxModel"`
	Computed     map[string]string `json:"computed"`
	MatchedRules []MatchedRule     `json:"matchedRules"`
	InlineStyles map[string]string `json:"inlineStyles,omitempty"`
	Pseudo       []PseudoElement   `json:"pseudoElements,omitempty"`
}

// BoxModel describes the element's layout box.
type BoxModel struct {
	Content struct {
		X, Y, Width, Height float64 `json:"x,y,width,height"`
	} `json:"content"`
	Padding struct {
		Top, Right, Bottom, Left float64 `json:"top,right,bottom,left"`
	} `json:"padding"`
	Border struct {
		Top, Right, Bottom, Left float64 `json:"top,right,bottom,left"`
	} `json:"border"`
	Margin struct {
		Top, Right, Bottom, Left float64 `json:"top,right,bottom,left"`
	} `json:"margin"`
}

// MatchedRule is a CSS rule that matches the element.
type MatchedRule struct {
	Selector    string       `json:"selector"`
	Properties  []Property   `json:"properties"`
	Source      string       `json:"source"`
	SourceLine  int          `json:"sourceLine"`
	Specificity Specificity  `json:"specificity"`
	Media       string       `json:"media,omitempty"`
	UserAgent   bool         `json:"userAgent"`
}

// Property is a single CSS property declaration.
type Property struct {
	Name       string `json:"name"`
	Value      string `json:"value"`
	Important  bool   `json:"important"`
	Overridden bool   `json:"overridden"`
}

// Specificity is a CSS selector specificity {a,b,c}.
type Specificity struct {
	A int `json:"a"`
	B int `json:"b"`
	C int `json:"c"`
}

// PseudoElement describes styles for ::before, ::after, etc.
type PseudoElement struct {
	Pseudo string       `json:"pseudo"`
	Rules  []PseudoRule `json:"rules"`
}

// PseudoRule is a rule within a pseudo-element.
type PseudoRule struct {
	Selector   string `json:"selector"`
	Properties string `json:"properties"`
}

// Modification records a single style change.
type Modification struct {
	Selector  string `json:"selector"`
	Property  string `json:"property"`
	OldValue  string `json:"oldValue"`
	NewValue  string `json:"newValue"`
	Source    string `json:"source"`
	Timestamp int64  `json:"timestamp"`
	Method    string `json:"method"` // "cdp" or "inline"
}

// Event is an SSE event emitted by the Inspector.
type Event struct {
	Type      string       `json:"type"`      // "apply", "undo", "reset"
	Timestamp int64        `json:"timestamp"`
	Mod       Modification `json:"mod,omitempty"`
	Index     int          `json:"index,omitempty"` // for undo
	Count     int          `json:"count,omitempty"` // for reset
}

// Inspector holds state for CSS inspection and modification.
type Inspector struct {
	mu      sync.Mutex
	history []Modification
}

// New creates a new Inspector.
func New() *Inspector {
	return &Inspector{}
}

// ─── Pub/Sub ──────────────────────────────────────────────

var (
	subscribers sync.Map // map[*chan Event]struct{}
)

// Subscribe registers a channel to receive live inspector events.
// Call Unsubscribe with the returned channel when done.
func Subscribe() *chan Event {
	ch := make(chan Event, 16)
	subscribers.Store(&ch, struct{}{})
	return &ch
}

// Unsubscribe removes a channel from live notifications.
func Unsubscribe(ch *chan Event) {
	subscribers.Delete(ch)
	close(*ch)
}

func emitEvent(e Event) {
	subscribers.Range(func(key, value interface{}) bool {
		ch := key.(*chan Event)
		select {
		case *ch <- e:
		default:
			// Subscriber slow — drop silently
		}
		return true
	})
}

// ─── Core Inspection ────────────────────────────────────────

// Inspect returns full CSS cascade data for an element matching selector.
func (in *Inspector) Inspect(ctx context.Context, selector string, includeUA bool) (*Result, error) {
	var result Result
	err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(c context.Context) error {
			if err := dom.Enable().Do(c); err != nil {
				return err
			}
			if err := css.Enable().Do(c); err != nil {
				return err
			}

			doc, err := dom.GetDocument().WithDepth(0).Do(c)
			if err != nil {
				return err
			}

			nodeID, err := dom.QuerySelector(doc.NodeID, selector).Do(c)
			if err != nil {
				return fmt.Errorf("element not found: %s — %w", selector, err)
			}
			if nodeID == 0 {
				return fmt.Errorf("element not found: %s", selector)
			}

			node, err := dom.DescribeNode().WithNodeID(nodeID).WithDepth(0).Do(c)
			if err != nil {
				return err
			}

			result = buildResult(node, selector)

			// Box model
			model, err := dom.GetBoxModel().WithNodeID(nodeID).Do(c)
			if err != nil {
				if !strings.Contains(err.Error(), "box model") {
					return err
				}
			} else if model != nil {
				result.BoxModel = convertBoxModel(model)
			}

			// Matched styles (returns many values)
			_, _, matchedCSSRules, pseudoElements, _, _, _, _, _, _, _, _, _, _, err := css.GetMatchedStylesForNode(nodeID).Do(c)
			if err != nil {
				return err
			}

			// Computed styles
			computedStyle, _, err := css.GetComputedStyleForNode(nodeID).Do(c)
			if err != nil {
				return err
			}
			result.Computed = filterKeyProperties(computedStyle)

			// Inline styles
			inlineStyle, _, err := css.GetInlineStylesForNode(nodeID).Do(c)
			if err != nil {
				return err
			}
			if inlineStyle != nil {
				result.InlineStyles = styleToMap(inlineStyle)
			}

			result.MatchedRules = processMatchedRules(matchedCSSRules, includeUA)
			result.Pseudo = processPseudoElements(pseudoElements)

			return nil
		}),
	)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func buildResult(node *cdp.Node, selector string) Result {
	r := Result{
		Selector:   selector,
		TagName:    strings.ToLower(node.LocalName),
		Attributes: make(map[string]string),
		Classes:    []string{},
	}
	for i := 0; i+1 < len(node.Attributes); i += 2 {
		k, v := node.Attributes[i], node.Attributes[i+1]
		r.Attributes[k] = v
		if k == "id" {
			r.ID = v
		}
		if k == "class" {
			r.Classes = strings.Fields(v)
		}
	}
	return r
}

func convertBoxModel(m *dom.BoxModel) BoxModel {
	c := m.Content
	p := m.Padding
	b := m.Border
	mg := m.Margin
	var bm BoxModel
	bm.Content.X = c[0]
	bm.Content.Y = c[1]
	bm.Content.Width = c[2] - c[0]
	bm.Content.Height = c[5] - c[1]

	bm.Padding.Top = c[1] - p[1]
	bm.Padding.Right = p[2] - c[2]
	bm.Padding.Bottom = p[5] - c[5]
	bm.Padding.Left = c[0] - p[0]

	bm.Border.Top = p[1] - b[1]
	bm.Border.Right = b[2] - p[2]
	bm.Border.Bottom = b[5] - p[5]
	bm.Border.Left = p[0] - b[0]

	bm.Margin.Top = b[1] - mg[1]
	bm.Margin.Right = mg[2] - b[2]
	bm.Margin.Bottom = mg[5] - b[5]
	bm.Margin.Left = b[0] - mg[0]
	return bm
}

var keyCSSProps = map[string]bool{
	"display": true, "position": true, "top": true, "right": true, "bottom": true, "left": true,
	"float": true, "clear": true, "z-index": true, "overflow": true, "overflow-x": true, "overflow-y": true,
	"width": true, "height": true, "min-width": true, "max-width": true, "min-height": true, "max-height": true,
	"margin-top": true, "margin-right": true, "margin-bottom": true, "margin-left": true,
	"padding-top": true, "padding-right": true, "padding-bottom": true, "padding-left": true,
	"border-top-width": true, "border-right-width": true, "border-bottom-width": true, "border-left-width": true,
	"border-style": true, "border-color": true,
	"font-family": true, "font-size": true, "font-weight": true, "line-height": true,
	"color": true, "background-color": true, "background-image": true, "opacity": true,
	"box-shadow": true, "border-radius": true, "transform": true, "transition": true,
	"flex-direction": true, "flex-wrap": true, "justify-content": true, "align-items": true, "gap": true,
	"grid-template-columns": true, "grid-template-rows": true,
	"text-align": true, "text-decoration": true, "visibility": true, "cursor": true, "pointer-events": true,
}

func filterKeyProperties(props []*css.ComputedStyleProperty) map[string]string {
	out := make(map[string]string)
	for _, p := range props {
		if keyCSSProps[p.Name] {
			out[p.Name] = p.Value
		}
	}
	return out
}

func styleToMap(s *css.Style) map[string]string {
	out := make(map[string]string)
	if s == nil {
		return out
	}
	for _, p := range s.CSSProperties {
		if p.Name != "" && p.Value != "" && !p.Disabled {
			out[p.Name] = p.Value
		}
	}
	return out
}

func processMatchedRules(rules []*css.RuleMatch, includeUA bool) []MatchedRule {
	if len(rules) == 0 {
		return nil
	}

	var matched []MatchedRule
	seen := make(map[string]int)

	for _, match := range rules {
		if match == nil || match.Rule == nil {
			continue
		}
		rule := match.Rule
		isUA := rule.Origin == css.StyleSheetOriginUserAgent
		if isUA && !includeUA {
			continue
		}

		selText := ""
		if rule.SelectorList != nil && len(rule.SelectorList.Selectors) > 0 {
			idx := 0
			if len(match.MatchingSelectors) > 0 {
				idx = int(match.MatchingSelectors[0])
			}
			if idx < len(rule.SelectorList.Selectors) {
				selText = rule.SelectorList.Selectors[idx].Text
			}
		}

		source := "inline"
		line := 0
		if rule.StyleSheetID != "" {
			source = string(rule.Origin)
		}
		if rule.Style != nil && rule.Style.Range != nil {
			line = int(rule.Style.Range.StartLine)
		}

		var media string
		if rule.Media != nil && len(rule.Media) > 0 {
			parts := make([]string, 0, len(rule.Media))
			for _, med := range rule.Media {
				if med.Text != "" {
					parts = append(parts, med.Text)
				}
			}
			media = strings.Join(parts, ", ")
		}

		spec := computeSpecificity(selText)

		var props []Property
		if rule.Style != nil {
			for _, p := range rule.Style.CSSProperties {
				if p.Name == "" || p.Disabled {
					continue
				}
				if strings.HasPrefix(p.Name, "-") && !keyCSSProps[p.Name] {
					continue
				}
				props = append(props, Property{
					Name:      p.Name,
					Value:     p.Value,
					Important: p.Important || strings.Contains(p.Value, "!important"),
				})
			}
		}

		matched = append(matched, MatchedRule{
			Selector:    selText,
			Properties:  props,
			Source:      source,
			SourceLine:  line,
			Specificity: spec,
			Media:       media,
			UserAgent:   isUA,
		})
	}

	// Sort by specificity (highest first)
	for i := range matched {
		for j := i + 1; j < len(matched); j++ {
			if compareSpecificity(matched[i].Specificity, matched[j].Specificity) < 0 {
				matched[i], matched[j] = matched[j], matched[i]
			}
		}
	}

	// Mark overridden
	for i, r := range matched {
		for j := range r.Properties {
			name := r.Properties[j].Name
			if prevIdx, ok := seen[name]; !ok {
				seen[name] = i
			} else {
				prev := matched[prevIdx].Properties
				var prevProp *Property
				for k := range prev {
					if prev[k].Name == name {
						prevProp = &prev[k]
						break
					}
				}
				if r.Properties[j].Important && prevProp != nil && !prevProp.Important {
					prevProp.Overridden = true
					seen[name] = i
				} else {
					matched[i].Properties[j].Overridden = true
				}
			}
		}
	}

	return matched
}

func processPseudoElements(pseudoList []*css.PseudoElementMatches) []PseudoElement {
	if len(pseudoList) == 0 {
		return nil
	}
	var out []PseudoElement
	for _, pseudo := range pseudoList {
		name := "::" + string(pseudo.PseudoType)
		var rules []PseudoRule
		for _, match := range pseudo.Matches {
			if match.Rule == nil || match.Rule.SelectorList == nil {
				continue
			}
			sel := match.Rule.SelectorList.Text
			var props []string
			if match.Rule.Style != nil {
				for _, p := range match.Rule.Style.CSSProperties {
					if p.Name != "" && !p.Disabled {
						props = append(props, fmt.Sprintf("%s: %s", p.Name, p.Value))
					}
				}
			}
			if len(props) > 0 {
				rules = append(rules, PseudoRule{
					Selector:   sel,
					Properties: strings.Join(props, "; "),
				})
			}
		}
		if len(rules) > 0 {
			out = append(out, PseudoElement{Pseudo: name, Rules: rules})
		}
	}
	return out
}

// ─── Specificity ────────────────────────────────────────────

func computeSpecificity(selector string) Specificity {
	var a, b, c int
	reID := regexp.MustCompile(`#[a-zA-Z_-][\w-]*`)
	a += len(reID.FindAllString(selector, -1))
	reClass := regexp.MustCompile(`\.[a-zA-Z_-][\w-]*`)
	b += len(reClass.FindAllString(selector, -1))
	reAttr := regexp.MustCompile(`\[[^\]]+\]`)
	b += len(reAttr.FindAllString(selector, -1))
	rePseudo := regexp.MustCompile(`(?<!:):[a-zA-Z][\w-]*`)
	b += len(rePseudo.FindAllString(selector, -1))
	reType := regexp.MustCompile(`(?:^|[\s+~>])([a-zA-Z][\w-]*)`)
	c += len(reType.FindAllString(selector, -1))
	rePseudoElem := regexp.MustCompile(`::[a-zA-Z][\w-]*`)
	c += len(rePseudoElem.FindAllString(selector, -1))
	return Specificity{A: a, B: b, C: c}
}

func compareSpecificity(s1, s2 Specificity) int {
	if s1.A != s2.A {
		return s1.A - s2.A
	}
	if s1.B != s2.B {
		return s1.B - s2.B
	}
	return s1.C - s2.C
}

// ─── Modification ───────────────────────────────────────────

var dangerousCSS = regexp.MustCompile(`(?i)url\s*\(|expression\s*\(|@import|javascript:|data:`)

// Apply modifies a CSS property on an element.
func (in *Inspector) Apply(ctx context.Context, selector, property, value string) (*Modification, error) {
	if !regexp.MustCompile(`^[a-zA-Z-]+$`).MatchString(property) {
		return nil, fmt.Errorf("invalid CSS property name: %s", property)
	}
	if dangerousCSS.MatchString(value) {
		return nil, fmt.Errorf("CSS value rejected: contains potentially dangerous pattern")
	}

	mod := &Modification{
		Selector:  selector,
		Property:  property,
		NewValue:  value,
		Timestamp: time.Now().UnixMilli(),
		Method:    "inline",
	}

	// Inspect to get old value
	res, err := in.Inspect(ctx, selector, false)
	if err != nil {
		return nil, err
	}
	mod.OldValue = res.Computed[property]

	// Try CDP approach
	err = chromedp.Run(ctx, chromedp.ActionFunc(func(c context.Context) error {
		if err := dom.Enable().Do(c); err != nil {
			return err
		}
		if err := css.Enable().Do(c); err != nil {
			return err
		}
		doc, err := dom.GetDocument().WithDepth(0).Do(c)
		if err != nil {
			return err
		}
		nodeID, err := dom.QuerySelector(doc.NodeID, selector).Do(c)
		if err != nil {
			return err
		}
		if nodeID == 0 {
			return fmt.Errorf("element not found: %s", selector)
		}

		_, _, matchedCSSRules, _, _, _, _, _, _, _, _, _, _, _, err := css.GetMatchedStylesForNode(nodeID).Do(c)
		if err != nil {
			return err
		}

		var targetRule *css.RuleMatch
		for _, match := range matchedCSSRules {
			if match.Rule == nil || match.Rule.Origin == css.StyleSheetOriginUserAgent {
				continue
			}
			hasProp := false
			if match.Rule.Style != nil {
				for _, p := range match.Rule.Style.CSSProperties {
					if p.Name == property {
						hasProp = true
						break
					}
				}
			}
			if hasProp && match.Rule.StyleSheetID != "" && match.Rule.Style != nil && match.Rule.Style.Range != nil {
				targetRule = match
				break
			}
		}

		if targetRule != nil {
			var props []string
			for _, p := range targetRule.Rule.Style.CSSProperties {
				if p.Name == property {
					props = append(props, fmt.Sprintf("%s: %s", p.Name, value))
				} else if p.Name != "" && !p.Disabled {
					props = append(props, fmt.Sprintf("%s: %s", p.Name, p.Value))
				}
			}
			newText := strings.Join(props, "; ")
			edit := &css.StyleDeclarationEdit{
				StyleSheetID: targetRule.Rule.StyleSheetID,
				Range:        targetRule.Rule.Style.Range,
				Text:         newText,
			}
			_, err := css.SetStyleTexts([]*css.StyleDeclarationEdit{edit}).Do(c)
			if err == nil {
				mod.Method = "cdp"
				mod.Source = fmt.Sprintf("%s:%d", targetRule.Rule.Origin, int(targetRule.Rule.Style.Range.StartLine))
				return nil
			}
		}
		return fmt.Errorf("cdp fallback")
	}))

	if err != nil || mod.Method == "inline" {
		err = chromedp.Run(ctx, chromedp.Evaluate(fmt.Sprintf(
			`(() => { const el = document.querySelector(%q); if (!el) throw "not found"; el.style.setProperty(%q, %q); })()`,
			selector, property, value), nil))
		if err != nil {
			return nil, err
		}
		mod.Source = "inline"
	}

	in.mu.Lock()
	in.history = append(in.history, *mod)
	in.mu.Unlock()

	emitEvent(Event{
		Type:      "apply",
		Timestamp: mod.Timestamp,
		Mod:       *mod,
	})
	return mod, nil
}

// Undo reverts a modification by index (or last if index < 0).
func (in *Inspector) Undo(ctx context.Context, index int) error {
	in.mu.Lock()
	if index < 0 {
		index = len(in.history) - 1
	}
	if index < 0 || index >= len(in.history) {
		in.mu.Unlock()
		return fmt.Errorf("no modification at index %d", index)
	}
	mod := in.history[index]
	in.mu.Unlock()

	err := chromedp.Run(ctx, chromedp.Evaluate(fmt.Sprintf(
		`(() => { const el = document.querySelector(%q); if (!el) return; if (%q === "") { el.style.removeProperty(%q); } else { el.style.setProperty(%q, %q); } })()`,
		mod.Selector, mod.OldValue, mod.Property, mod.Property, mod.OldValue), nil))
	if err != nil {
		return err
	}

	in.mu.Lock()
	in.history = append(in.history[:index], in.history[index+1:]...)
	in.mu.Unlock()

	emitEvent(Event{
		Type:      "undo",
		Timestamp: time.Now().UnixMilli(),
		Mod:       mod,
		Index:     index,
	})
	return nil
}

// Reset reverts all modifications.
func (in *Inspector) Reset(ctx context.Context) error {
	in.mu.Lock()
	history := make([]Modification, len(in.history))
	copy(history, in.history)
	in.mu.Unlock()

	for i := len(history) - 1; i >= 0; i-- {
		mod := history[i]
		_ = chromedp.Run(ctx, chromedp.Evaluate(fmt.Sprintf(
			`(() => { const el = document.querySelector(%q); if (!el) return; if (%q === "") { el.style.removeProperty(%q); } else { el.style.setProperty(%q, %q); } })()`,
			mod.Selector, mod.OldValue, mod.Property, mod.Property, mod.OldValue), nil))
	}

	in.mu.Lock()
	count := len(in.history)
	in.history = nil
	in.mu.Unlock()

	emitEvent(Event{
		Type:      "reset",
		Timestamp: time.Now().UnixMilli(),
		Count:     count,
	})
	return nil
}

// History returns all modifications.
func (in *Inspector) History() []Modification {
	in.mu.Lock()
	defer in.mu.Unlock()
	out := make([]Modification, len(in.history))
	copy(out, in.history)
	return out
}

// Format returns a human-readable string for a Result.
func Format(r *Result) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Element: <%s", r.TagName)
	if r.ID != "" {
		fmt.Fprintf(&b, ` id="%s"`, r.ID)
	}
	if len(r.Classes) > 0 {
		fmt.Fprintf(&b, ` class="%s"`, strings.Join(r.Classes, " "))
	}
	fmt.Fprintln(&b, ">")
	fmt.Fprintf(&b, "Selector: %s\n", r.Selector)

	w := r.BoxModel.Content.Width + r.BoxModel.Padding.Left + r.BoxModel.Padding.Right
	h := r.BoxModel.Content.Height + r.BoxModel.Padding.Top + r.BoxModel.Padding.Bottom
	fmt.Fprintf(&b, "Dimensions: %.0f x %.0f\n\n", w, h)

	fmt.Fprintln(&b, "Box Model:")
	fmt.Fprintf(&b, "  margin:  %.0f  %.0f  %.0f  %.0f\n", r.BoxModel.Margin.Top, r.BoxModel.Margin.Right, r.BoxModel.Margin.Bottom, r.BoxModel.Margin.Left)
	fmt.Fprintf(&b, "  padding: %.0f  %.0f  %.0f  %.0f\n", r.BoxModel.Padding.Top, r.BoxModel.Padding.Right, r.BoxModel.Padding.Bottom, r.BoxModel.Padding.Left)
	fmt.Fprintf(&b, "  border:  %.0f  %.0f  %.0f  %.0f\n", r.BoxModel.Border.Top, r.BoxModel.Border.Right, r.BoxModel.Border.Bottom, r.BoxModel.Border.Left)
	fmt.Fprintf(&b, "  content: %.0f x %.0f\n\n", r.BoxModel.Content.Width, r.BoxModel.Content.Height)

	fmt.Fprintf(&b, "Matched Rules (%d):\n", len(r.MatchedRules))
	for _, rule := range r.MatchedRules {
		var active []string
		for _, p := range rule.Properties {
			if !p.Overridden {
				suffix := ""
				if p.Important {
					suffix = " !important"
				}
				active = append(active, fmt.Sprintf("%s: %s%s", p.Name, p.Value, suffix))
			}
		}
		if len(active) == 0 {
			continue
		}
		fmt.Fprintf(&b, "  %s { %s }\n", rule.Selector, strings.Join(active, "; "))
		fmt.Fprintf(&b, "    -> %s:%d [%d,%d,%d]\n", rule.Source, rule.SourceLine, rule.Specificity.A, rule.Specificity.B, rule.Specificity.C)
	}

	fmt.Fprintln(&b, "\nInline Styles:")
	if len(r.InlineStyles) == 0 {
		fmt.Fprintln(&b, "  (none)")
	} else {
		var pairs []string
		for k, v := range r.InlineStyles {
			pairs = append(pairs, fmt.Sprintf("%s: %s", k, v))
		}
		fmt.Fprintf(&b, "  %s\n", strings.Join(pairs, "; "))
	}

	fmt.Fprintln(&b, "\nComputed (key):")
	for _, prop := range sortedKeys(r.Computed) {
		fmt.Fprintf(&b, "  %s: %s\n", prop, r.Computed[prop])
	}

	if len(r.Pseudo) > 0 {
		fmt.Fprintln(&b, "\nPseudo-elements:")
		for _, pseudo := range r.Pseudo {
			for _, rule := range pseudo.Rules {
				fmt.Fprintf(&b, "  %s %s { %s }\n", pseudo.Pseudo, rule.Selector, rule.Properties)
			}
		}
	}

	return b.String()
}

func sortedKeys(m map[string]string) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	for i := range keys {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}
