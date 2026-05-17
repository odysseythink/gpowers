package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"browse-go/pkg/security"
	"browse-go/pkg/telemetry"

	"github.com/chromedp/cdproto/accessibility"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/css"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/performance"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/cdproto/tracing"
	"github.com/chromedp/chromedp"
)

func (r *Registry) registerCdp() {
	r.Register("cdp", CommandDesc{Category: "Meta", Description: "Raw CDP method dispatch (deny-default)", Usage: "cdp <Domain.method> [json-params]"},
		func(ctx *ExecContext) (string, error) {
			if len(ctx.Args) == 0 || ctx.Args[0] == "help" || ctx.Args[0] == "--help" {
				return cdpHelp(), nil
			}

			qualified := ctx.Args[0]
			_, _, err := parseQualified(qualified)
			if err != nil {
				return "", err
			}

			// Optional JSON params; default to {}
			params := map[string]any{}
			if len(ctx.Args) > 1 && ctx.Args[1] != "" {
				if err := json.Unmarshal([]byte(ctx.Args[1]), &params); err != nil {
					return "", fmt.Errorf("cannot parse params as JSON: %w\nCause: argument %q is not valid JSON.\nAction: pass a JSON object literal, e.g. '{\"backendNodeId\":42}'.", err, ctx.Args[1])
				}
			}

			entry := LookupCdpMethod(qualified)
			if entry == nil {
				domain, method, _ := parseQualified(qualified)
				telemetry.Log(telemetry.Event{"event": "cdp_method_denied", "domain": domain, "method": method})
				return "", fmt.Errorf(
					"DENIED: %s is not on the CDP allowlist.\n"+
						"Cause: deny-default posture; method has not been audited.\n"+
						"Action: if this method is genuinely needed, add it to CDPAllowlist with justification + scope + output.",
					qualified,
				)
			}
			domain, method, _ := parseQualified(qualified)
			telemetry.Log(telemetry.Event{"event": "cdp_method_called", "domain": domain, "method": method, "allowed": true, "scope": string(entry.Scope)})

			tabCtx := ctx.Session.Context()

			// Dispatch with a 5s timeout
			dispatchCtx, cancel := context.WithTimeout(tabCtx, 5*time.Second)
			defer cancel()

			var rawResult any
			var dispatchErr error

			switch qualified {
			// ─── Accessibility ─────────────────────────────────────
			case "Accessibility.getFullAXTree":
				nodes, err := accessibility.GetFullAXTree().Do(dispatchCtx)
				rawResult, dispatchErr = nodes, err
			case "Accessibility.getPartialAXTree":
				nodeID := cdp.NodeID(0)
				if v, ok := params["nodeId"].(float64); ok {
					nodeID = cdp.NodeID(v)
				}
				nodes, err := accessibility.GetPartialAXTree().WithNodeID(nodeID).Do(dispatchCtx)
				rawResult, dispatchErr = nodes, err
			case "Accessibility.getRootAXNode":
				node, err := accessibility.GetRootAXNode().Do(dispatchCtx)
				rawResult, dispatchErr = node, err

			// ─── DOM ───────────────────────────────────────────────
			case "DOM.describeNode":
				nodeID := cdp.NodeID(0)
				if v, ok := params["nodeId"].(float64); ok {
					nodeID = cdp.NodeID(v)
				}
				backendNodeID := cdp.BackendNodeID(0)
				if v, ok := params["backendNodeId"].(float64); ok {
					backendNodeID = cdp.BackendNodeID(v)
				}
				depth := int64(-1)
				if v, ok := params["depth"].(float64); ok {
					depth = int64(v)
				}
				node, err := dom.DescribeNode().WithNodeID(nodeID).WithBackendNodeID(backendNodeID).WithDepth(depth).Do(dispatchCtx)
				rawResult, dispatchErr = node, err
			case "DOM.getBoxModel":
				nodeID := cdp.NodeID(0)
				if v, ok := params["nodeId"].(float64); ok {
					nodeID = cdp.NodeID(v)
				}
				backendNodeID := cdp.BackendNodeID(0)
				if v, ok := params["backendNodeId"].(float64); ok {
					backendNodeID = cdp.BackendNodeID(v)
				}
				model, err := dom.GetBoxModel().WithNodeID(nodeID).WithBackendNodeID(backendNodeID).Do(dispatchCtx)
				rawResult, dispatchErr = model, err
			case "DOM.getNodeForLocation":
				x := int64(0)
				if v, ok := params["x"].(float64); ok {
					x = int64(v)
				}
				y := int64(0)
				if v, ok := params["y"].(float64); ok {
					y = int64(v)
				}
				backendNodeID, frameID, nodeID, err := dom.GetNodeForLocation(x, y).Do(dispatchCtx)
				rawResult = map[string]any{"backendNodeID": backendNodeID, "frameID": frameID, "nodeID": nodeID}
				dispatchErr = err

			// ─── CSS ───────────────────────────────────────────────
			case "CSS.getMatchedStylesForNode":
				nodeID := cdp.NodeID(0)
				if v, ok := params["nodeId"].(float64); ok {
					nodeID = cdp.NodeID(v)
				}
				inlineStyle, attributesStyle, matchedCSSRules, pseudoElements, inherited, inheritedPseudoElements, cssKeyframesRules, cssPositionTryRules, activePositionFallbackIndex, cssPropertyRules, cssPropertyRegistrations, cssAtRules, parentLayoutNodeID, cssFunctionRules, err := css.GetMatchedStylesForNode(nodeID).Do(dispatchCtx)
				rawResult = map[string]any{
					"inlineStyle": inlineStyle, "attributesStyle": attributesStyle,
					"matchedCSSRules": matchedCSSRules, "pseudoElements": pseudoElements,
					"inherited": inherited, "inheritedPseudoElements": inheritedPseudoElements,
					"cssKeyframesRules": cssKeyframesRules, "cssPositionTryRules": cssPositionTryRules,
					"activePositionFallbackIndex": activePositionFallbackIndex,
					"cssPropertyRules": cssPropertyRules, "cssPropertyRegistrations": cssPropertyRegistrations,
					"cssAtRules": cssAtRules, "parentLayoutNodeID": parentLayoutNodeID,
					"cssFunctionRules": cssFunctionRules,
				}
				dispatchErr = err
			case "CSS.getComputedStyleForNode":
				nodeID := cdp.NodeID(0)
				if v, ok := params["nodeId"].(float64); ok {
					nodeID = cdp.NodeID(v)
				}
				computedStyle, extraFields, err := css.GetComputedStyleForNode(nodeID).Do(dispatchCtx)
				rawResult = map[string]any{"computedStyle": computedStyle, "extraFields": extraFields}
				dispatchErr = err
			case "CSS.getInlineStylesForNode":
				nodeID := cdp.NodeID(0)
				if v, ok := params["nodeId"].(float64); ok {
					nodeID = cdp.NodeID(v)
				}
				inlineStyle, attributesStyle, err := css.GetInlineStylesForNode(nodeID).Do(dispatchCtx)
				rawResult = map[string]any{"inlineStyle": inlineStyle, "attributesStyle": attributesStyle}
				dispatchErr = err

			// ─── Performance ───────────────────────────────────────
			case "Performance.getMetrics":
				var metrics []*performance.Metric
				dispatchErr = chromedp.Run(dispatchCtx, chromedp.ActionFunc(func(c context.Context) error {
					var err error
					metrics, err = performance.GetMetrics().Do(c)
					return err
				}))
				rawResult = metrics
			case "Performance.enable":
				dispatchErr = chromedp.Run(dispatchCtx, performance.Enable())
				rawResult = map[string]string{"status": "enabled"}
			case "Performance.disable":
				dispatchErr = chromedp.Run(dispatchCtx, performance.Disable())
				rawResult = map[string]string{"status": "disabled"}

			// ─── Tracing ───────────────────────────────────────────
			case "Tracing.start":
				dispatchErr = chromedp.Run(dispatchCtx, tracing.Start())
				rawResult = map[string]string{"status": "started"}
			case "Tracing.end":
				dispatchErr = chromedp.Run(dispatchCtx, tracing.End())
				rawResult = map[string]string{"status": "ended"}

			// ─── Emulation ─────────────────────────────────────────
			case "Emulation.setDeviceMetricsOverride":
				width := int64(1280)
				if v, ok := params["width"].(float64); ok {
					width = int64(v)
				}
				height := int64(720)
				if v, ok := params["height"].(float64); ok {
					height = int64(v)
				}
				dispatchErr = chromedp.Run(dispatchCtx, emulation.SetDeviceMetricsOverride(width, height, 1, false))
				rawResult = map[string]int64{"width": width, "height": height}
			case "Emulation.clearDeviceMetricsOverride":
				dispatchErr = chromedp.Run(dispatchCtx, emulation.ClearDeviceMetricsOverride())
				rawResult = map[string]string{"status": "cleared"}
			case "Emulation.setUserAgentOverride":
				ua := ""
				if v, ok := params["userAgent"].(string); ok {
					ua = v
				}
				dispatchErr = chromedp.Run(dispatchCtx, emulation.SetUserAgentOverride(ua))
				rawResult = map[string]string{"userAgent": ua}

			// ─── Page ──────────────────────────────────────────────
			case "Page.captureScreenshot":
				format := "png"
				if v, ok := params["format"].(string); ok {
					format = v
				}
				quality := int64(0)
				if v, ok := params["quality"].(float64); ok {
					quality = int64(v)
				}
				act := page.CaptureScreenshot()
				if format == "jpeg" {
					act = act.WithFormat(page.CaptureScreenshotFormatJpeg)
				}
				if quality > 0 {
					act = act.WithQuality(quality)
				}
				data, err := act.Do(dispatchCtx)
				rawResult, dispatchErr = map[string]int{"size": len(data)}, err
			case "Page.printToPDF":
				act := page.PrintToPDF()
				if v, ok := params["landscape"].(bool); ok && v {
					act = act.WithLandscape(true)
				}
				data, _, err := act.Do(dispatchCtx)
				rawResult, dispatchErr = map[string]int{"size": len(data)}, err

			// ─── Network ───────────────────────────────────────────
			case "Network.enable":
				dispatchErr = chromedp.Run(dispatchCtx, network.Enable())
				rawResult = map[string]string{"status": "enabled"}
			case "Network.disable":
				dispatchErr = chromedp.Run(dispatchCtx, network.Disable())
				rawResult = map[string]string{"status": "disabled"}


			// ─── Target ────────────────────────────────────────────
			case "Target.getTargets":
				infos, err := target.GetTargets().Do(dispatchCtx)
				rawResult = map[string]any{"targetInfos": infos}
				dispatchErr = err
			case "Target.attachToTarget":
				targetID := target.ID("")
				if v, ok := params["targetId"].(string); ok {
					targetID = target.ID(v)
				}
				flatten := true
				if v, ok := params["flatten"].(bool); ok {
					flatten = v
				}
				sessionID, err := target.AttachToTarget(targetID).WithFlatten(flatten).Do(dispatchCtx)
				rawResult = map[string]any{"sessionId": sessionID}
				dispatchErr = err
			case "Target.detachFromTarget":
				act := target.DetachFromTarget()
				if v, ok := params["sessionId"].(string); ok {
					act = act.WithSessionID(target.SessionID(v))
				}
				dispatchErr = act.Do(dispatchCtx)
				rawResult = map[string]string{"status": "detached"}
			case "Target.getTargetInfo":
				targetID := target.ID("")
				if v, ok := params["targetId"].(string); ok {
					targetID = target.ID(v)
				}
				info, err := target.GetTargetInfo().WithTargetID(targetID).Do(dispatchCtx)
				rawResult = info
				dispatchErr = err

			// ─── Page ──────────────────────────────────────────────
			case "Page.getFrameTree":
				ft, err := page.GetFrameTree().Do(dispatchCtx)
				rawResult = ft
				dispatchErr = err

			// ─── Runtime ───────────────────────────────────────────
			case "Runtime.getProperties":
				objectID := runtime.RemoteObjectID("")
				if v, ok := params["objectId"].(string); ok {
					objectID = runtime.RemoteObjectID(v)
				}
				result, internalProperties, privateProperties, exceptionDetails, err := runtime.GetProperties(objectID).Do(dispatchCtx)
				rawResult = map[string]any{
					"result": result, "internalProperties": internalProperties,
					"privateProperties": privateProperties, "exceptionDetails": exceptionDetails,
				}
				dispatchErr = err

			default:
				return "", fmt.Errorf("DENIED: %s is not on the CDP allowlist", qualified)
			}

			if dispatchErr != nil {
				// Check for context timeout
				if dispatchCtx.Err() == context.DeadlineExceeded {
					return "", fmt.Errorf("CDPBridgeTimeout: %s did not return within 5s", qualified)
				}
				return "", fmt.Errorf("CDP call failed: %w", dispatchErr)
			}

			jsonOut, _ := json.MarshalIndent(rawResult, "", "  ")
			if entry.Output == CdpOutputUntrusted {
				return security.WrapUntrusted(string(jsonOut), "cdp:"+qualified), nil
			}
			return string(jsonOut), nil
		})
}

func cdpHelp() string {
	return `$B cdp — raw CDP method dispatch (deny-default escape hatch)

Usage: $B cdp <Domain.method> [json-params]

Allowed methods are listed in the CDP allowlist. Examples:
  $B cdp Accessibility.getFullAXTree {}
  $B cdp Performance.getMetrics {}
  $B cdp DOM.describeNode '{"backendNodeId":42,"depth":3}'`
}

func parseQualified(name string) (domain, method string, err error) {
	idx := strings.Index(name, ".")
	if idx <= 0 || idx == len(name)-1 {
		return "", "", fmt.Errorf(
			"Usage: $B cdp <Domain.method> [json-params]\n"+
				"Cause: %q is not in Domain.method format.\n"+
				"Action: e.g. $B cdp Accessibility.getFullAXTree {}",
			name,
		)
	}
	return name[:idx], name[idx+1:], nil
}
