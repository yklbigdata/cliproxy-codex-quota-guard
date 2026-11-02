package main

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	statusResourcePath  = "/status"
	resourceContentType = "text/html; charset=utf-8"
)

type managementRegistration struct {
	Resources []managementResource `json:"resources,omitempty"`
}

type managementResource struct {
	Path        string `json:"Path"`
	Menu        string `json:"Menu"`
	Description string `json:"Description"`
}

type managementRequest struct {
	Method         string
	Path           string
	Headers        http.Header
	Query          url.Values
	Body           []byte
	HostCallbackID string `json:"host_callback_id,omitempty"`
}

type managementResponse struct {
	StatusCode int         `json:"StatusCode"`
	Headers    http.Header `json:"Headers"`
	Body       []byte      `json:"Body"`
}

func managementRegistrationPayload() managementRegistration {
	return managementRegistration{
		Resources: []managementResource{{
			Path:        statusResourcePath,
			Menu:        pluginName,
			Description: "Codex 5h/7d quota per credential together with the guard decision history.",
		}},
	}
}

func handleManagement(raw []byte) ([]byte, error) {
	var req managementRequest
	if len(raw) > 0 {
		if errUnmarshal := json.Unmarshal(raw, &req); errUnmarshal != nil {
			return nil, fmt.Errorf("decode management request: %w", errUnmarshal)
		}
	}
	path := strings.TrimSpace(req.Path)
	if path != "" && !strings.HasSuffix(path, statusResourcePath) {
		return okEnvelope(htmlResponse(http.StatusNotFound, []byte("not found")))
	}
	return okEnvelope(htmlResponse(http.StatusOK, []byte(renderStatusPage(guard.snapshot()))))
}

func htmlResponse(statusCode int, body []byte) managementResponse {
	return managementResponse{
		StatusCode: statusCode,
		Headers: http.Header{
			"content-type": []string{resourceContentType},
		},
		Body: body,
	}
}

func renderStatusPage(snapshot statusSnapshot) string {
	var out strings.Builder
	out.WriteString("<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\">")
	out.WriteString("<meta name=\"viewport\" content=\"width=device-width,initial-scale=1\">")
	out.WriteString("<title>Codex Quota Guard</title><style>")
	out.WriteString(":root{color-scheme:light dark}")
	out.WriteString("body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;margin:2rem;line-height:1.5;max-width:1200px}")
	out.WriteString("h1{font-size:1.35rem;margin:0 0 .25rem}h2{font-size:1rem;margin:2rem 0 .5rem}")
	out.WriteString(".muted{opacity:.65;font-size:.85rem}")
	out.WriteString("table{border-collapse:collapse;width:100%;font-size:.875rem}")
	out.WriteString("th,td{border-bottom:1px solid rgba(128,128,128,.35);padding:.45rem .6rem;text-align:left;vertical-align:top}")
	out.WriteString("th{font-weight:600;opacity:.8}")
	out.WriteString("code{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:.85em}")
	out.WriteString(".pill{display:inline-block;padding:.05rem .45rem;border-radius:999px;border:1px solid rgba(128,128,128,.5);font-size:.78rem}")
	out.WriteString(".warn{color:#b45309}.bad{color:#b91c1c}.ok{color:#15803d}")
	out.WriteString("dl{display:grid;grid-template-columns:max-content 1fr;gap:.2rem 1rem;margin:0;font-size:.875rem}")
	out.WriteString("dt{opacity:.7}dd{margin:0}")
	out.WriteString("</style></head><body>")
	out.WriteString("<h1>Codex Quota Guard</h1>")
	out.WriteString(fmt.Sprintf("<p class=\"muted\">plugin %s &middot; Codex 5h window driven enable/disable &middot; no request-outcome heuristics</p>",
		html.EscapeString(snapshot.PluginVersion)))

	out.WriteString("<h2>State</h2><dl>")
	out.WriteString(fmt.Sprintf("<dt>worker</dt><dd>%s</dd>", runningLabel(snapshot)))
	out.WriteString(fmt.Sprintf("<dt>last sweep</dt><dd>%s</dd>", timestampLabel(snapshot.LastSweep)))
	out.WriteString(fmt.Sprintf("<dt>next sweep</dt><dd>%s</dd>", timestampLabel(snapshot.NextSweep)))
	out.WriteString(fmt.Sprintf("<dt>sweeps</dt><dd>%d &middot; disables %d &middot; enables %d &middot; errors %d</dd>",
		snapshot.Sweeps, snapshot.Disables, snapshot.Enables, snapshot.Errors))
	if snapshot.LastSweepError != "" {
		out.WriteString(fmt.Sprintf("<dt>last error</dt><dd class=\"bad\">%s</dd>", html.EscapeString(snapshot.LastSweepError)))
	}
	out.WriteString("</dl>")

	cfg := snapshot.Config
	out.WriteString("<h2>Policy</h2><dl>")
	out.WriteString(fmt.Sprintf("<dt>disable at</dt><dd>5h remaining &le; %s</dd>", formatPercent(cfg.DisableBelowRemainingPercent)))
	out.WriteString(fmt.Sprintf("<dt>re-enable at</dt><dd>5h remaining &ge; %s</dd>", formatPercent(cfg.ResumeAboveRemainingPercent)))
	out.WriteString(fmt.Sprintf("<dt>7d window</dt><dd>%s</dd>", weeklyPolicyLabel(cfg)))
	out.WriteString(fmt.Sprintf("<dt>poll interval</dt><dd>%s</dd>", cfg.pollInterval()))
	out.WriteString(fmt.Sprintf("<dt>keep at least one</dt><dd>%t</dd>", cfg.keepAtLeastOne()))
	out.WriteString(fmt.Sprintf("<dt>dry run</dt><dd>%t</dd>", cfg.dryRun()))
	out.WriteString("</dl>")

	out.WriteString("<h2>Credentials</h2>")
	if len(snapshot.Reports) == 0 {
		out.WriteString("<p class=\"muted\">No sweep has completed yet.</p>")
	} else {
		out.WriteString("<table><thead><tr>")
		for _, header := range []string{"credential", "plan", "5h used", "5h reset", "7d used", "file off", "runtime off", "guard", "action", "reason", "checked"} {
			out.WriteString("<th>" + html.EscapeString(header) + "</th>")
		}
		out.WriteString("</tr></thead><tbody>")
		for _, report := range snapshot.Reports {
			out.WriteString("<tr>")
			out.WriteString("<td><code>" + html.EscapeString(report.Name) + "</code><div class=\"muted\">" +
				html.EscapeString(strings.TrimSpace(report.Email)) + "</div></td>")
			out.WriteString("<td>" + escapeOrDash(report.PlanType) + "</td>")
			out.WriteString("<td>" + percentCell(report.PrimaryUsed) + "</td>")
			out.WriteString("<td>" + formatResetIn(report.PrimaryResetIn) + "</td>")
			out.WriteString("<td>" + percentCell(report.WeeklyUsed) + "</td>")
			out.WriteString("<td>" + boolCell(report.FileDisabled) + "</td>")
			out.WriteString("<td>" + boolCell(report.RuntimeDisabled) + "</td>")
			out.WriteString("<td>" + boolCell(report.Managed) + "</td>")
			out.WriteString("<td>" + actionCell(report.Action) + "</td>")
			out.WriteString("<td>" + reasonCell(report) + "</td>")
			out.WriteString("<td class=\"muted\">" + timestampLabel(report.CheckedAt) + "</td>")
			out.WriteString("</tr>")
		}
		out.WriteString("</tbody></table>")
	}
	out.WriteString("</body></html>")
	return out.String()
}

func runningLabel(snapshot statusSnapshot) string {
	if snapshot.Running {
		return fmt.Sprintf("running since %s", timestampLabel(snapshot.StartedAt))
	}
	return "stopped"
}

func weeklyPolicyLabel(cfg pluginConfig) string {
	if !cfg.enforceWeekly() {
		return "ignored"
	}
	return fmt.Sprintf("disable at &le; %s, re-enable at &ge; %s",
		formatPercent(cfg.WeeklyDisableBelowPercent), formatPercent(cfg.WeeklyResumeAbovePercent))
}

func escapeOrDash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return html.EscapeString(value)
}

func percentCell(value *float64) string {
	if value == nil {
		return "-"
	}
	return formatPercent(*value)
}

func boolCell(value bool) string {
	if value {
		return "<span class=\"pill warn\">yes</span>"
	}
	return "<span class=\"pill\">no</span>"
}

func actionCell(action string) string {
	switch action {
	case actionDisabled:
		return "<span class=\"pill warn\">disabled</span>"
	case actionEnabled:
		return "<span class=\"pill ok\">enabled</span>"
	case actionError:
		return "<span class=\"pill bad\">error</span>"
	case actionSkipped:
		return "<span class=\"pill\">skipped</span>"
	default:
		return "<span class=\"pill\">none</span>"
	}
}

func reasonCell(report credentialReport) string {
	text := strings.TrimSpace(report.Reason)
	if report.Error != "" {
		if text != "" {
			text += ": "
		}
		text += report.Error
	}
	if text == "" {
		return "-"
	}
	return html.EscapeString(text)
}

func timestampLabel(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return html.EscapeString(value.Local().Format("2006-01-02 15:04:05"))
}
