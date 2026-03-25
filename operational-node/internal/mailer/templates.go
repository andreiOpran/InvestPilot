package mailer

import (
	"bytes"
	"embed"
	"strings"
	"text/template"
)

//go:embed templates/*.txt
var templateFiles embed.FS

// BuildEmailContent parses the named template (embedded) and executes it with data
// the template file should start with a "Subject:" line followed by a blank line and the body
func BuildEmailContent(templateName string, data interface{}) (string, string, error) {
	tmpl, err := template.ParseFS(templateFiles, "templates/"+templateName+".txt")
	if err != nil {
		return "", "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", "", err
	}

	content := buf.String()
	// expect first line to be 'Subject: ...' and a blank line, then body
	// extract subject and normalize newlines
	content = strings.ReplaceAll(content, "\r\n", "\n")
	parts := strings.SplitN(content, "\n\n", 2)
	var subject, body string
	if len(parts) >= 1 {
		// first line may contain "Subject:" prefix
		firstLine := strings.SplitN(parts[0], "\n", 2)[0]
		subject = strings.TrimSpace(strings.TrimPrefix(firstLine, "Subject:"))
	}
	if len(parts) == 2 {
		body = strings.TrimSpace(parts[1])
	}

	return subject, body, nil
}
