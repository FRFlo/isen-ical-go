// Package template provides template rendering functionality using Go's html/template
package template

import (
	"bytes"
	"html/template"
)

// Render executes a template with the provided data
// The templateContent should be valid Go template syntax
func Render(templateContent string, data interface{}) (string, error) {
	tmpl, err := template.New("page").Parse(templateContent)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
