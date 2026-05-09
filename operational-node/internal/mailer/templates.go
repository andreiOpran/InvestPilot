package mailer

import (
	"bytes"
	"embed"
	"html/template"
	"time"
)

//go:embed templates/*.html
var templateFiles embed.FS

var funcMap = template.FuncMap{
	"currentYear": func() int { return time.Now().Year() },
}

// BuildEmailContent parses the named HTML template (embedded) and executes the
// "subject" and "body" named blocks defined within it.
func BuildEmailContent(templateName string, data interface{}) (string, string, error) {
	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templateFiles, "templates/"+templateName+".html")
	if err != nil {
		return "", "", err
	}

	var subjectBuf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&subjectBuf, "subject", data); err != nil {
		return "", "", err
	}

	var bodyBuf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&bodyBuf, "body", data); err != nil {
		return "", "", err
	}

	return subjectBuf.String(), bodyBuf.String(), nil
}
