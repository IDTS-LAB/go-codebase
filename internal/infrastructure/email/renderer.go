package email

import (
	"bytes"
	"embed"
	"html/template"
)

//go:embed templates/*.html
var templateFS embed.FS

type TemplateData struct {
	Name        string
	VerifyURL   string
	ResetURL    string
	InviteURL   string
	InviterName string
}

func renderTemplate(name string, data TemplateData) (string, error) {
	tmplPath := "templates/" + name + ".html"
	t, err := template.ParseFS(templateFS, tmplPath)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
