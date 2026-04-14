package templates

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"time"
)

type Engine struct {
	templates *template.Template
	fs        fs.FS
	devMode   bool
}

func NewEngine(fsys fs.FS, devMode bool) (*Engine, error) {
	e := &Engine{
		fs:      fsys,
		devMode: devMode,
	}

	if err := e.parse(); err != nil {
		return nil, err
	}

	return e, nil
}

func (e *Engine) parse() error {
	funcMap := template.FuncMap{
		"timeAgo":     timeAgo,
		"shortSHA":    shortSHA,
		"statusColor": statusColor,
		"statusIcon":  statusIcon,
		"upper":       strings.ToUpper,
		"lower":       strings.ToLower,
		"title":       title,
		"join":        strings.Join,
		"dict":        dict,
		"split":       splitLines,
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseFS(e.fs, "templates/layouts/*.html", "templates/pages/*.html", "templates/partials/*.html")
	if err != nil {
		return fmt.Errorf("parsing templates: %w", err)
	}

	e.templates = tmpl
	return nil
}

func (e *Engine) Render(w http.ResponseWriter, name string, data any) error {
	if e.devMode {
		if err := e.parse(); err != nil {
			return err
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return e.templates.ExecuteTemplate(w, name, data)
}

func (e *Engine) RenderPartial(w http.ResponseWriter, name string, data any) error {
	if e.devMode {
		if err := e.parse(); err != nil {
			return err
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return e.templates.ExecuteTemplate(w, name, data)
}

func (e *Engine) RenderToString(name string, data any) (string, error) {
	if e.devMode {
		if err := e.parse(); err != nil {
			return "", err
		}
	}

	var buf strings.Builder
	if err := e.templates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Template functions

func timeAgo(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

func statusColor(status, conclusion string) string {
	if status == "completed" {
		switch conclusion {
		case "success":
			return "green"
		case "failure":
			return "red"
		case "cancelled":
			return "gray"
		default:
			return "yellow"
		}
	}
	switch status {
	case "in_progress":
		return "yellow"
	case "queued":
		return "blue"
	default:
		return "gray"
	}
}

func statusIcon(status, conclusion string) template.HTML {
	if status == "completed" {
		switch conclusion {
		case "success":
			return template.HTML(`<svg class="w-4 h-4 text-green-500" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"/></svg>`)
		case "failure":
			return template.HTML(`<svg class="w-4 h-4 text-red-500" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd"/></svg>`)
		case "cancelled":
			return template.HTML(`<svg class="w-4 h-4 text-gray-400" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM7 9a1 1 0 000 2h6a1 1 0 100-2H7z" clip-rule="evenodd"/></svg>`)
		}
	}
	switch status {
	case "in_progress":
		return template.HTML(`<svg class="w-4 h-4 text-yellow-500 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/></svg>`)
	case "queued":
		return template.HTML(`<svg class="w-4 h-4 text-blue-400" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm1-12a1 1 0 10-2 0v4a1 1 0 00.293.707l2.828 2.829a1 1 0 101.415-1.415L11 9.586V6z" clip-rule="evenodd"/></svg>`)
	default:
		return template.HTML(`<svg class="w-4 h-4 text-gray-400" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm1-11a1 1 0 10-2 0v2H7a1 1 0 100 2h2v2a1 1 0 102 0v-2h2a1 1 0 100-2h-2V7z" clip-rule="evenodd"/></svg>`)
	}
}

func dict(values ...any) map[string]any {
	if len(values)%2 != 0 {
		return nil
	}
	d := make(map[string]any, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			continue
		}
		d[key] = values[i+1]
	}
	return d
}

func splitLines(s string) []string {
	return strings.Split(s, "\n")
}

func title(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
