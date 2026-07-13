package email

import (
	"bytes"
	"embed"
	"html/template"
)

//go:embed templates/*.html
var templateFS embed.FS

var templates = template.Must(template.ParseFS(templateFS, "templates/*.html"))

type TemplateData struct {
	Name        string
	VerifyURL   string
	ResetURL    string
	InviteURL   string
	InviterName string
}

func renderTemplate(name string, data TemplateData) (string, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, name+".html", data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
