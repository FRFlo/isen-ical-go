// Package template fournit la fonctionnalité de rendu de templates avec html/template de Go
package template

import (
	"bytes"
	"html/template"
)

// Render exécute un template avec les données fournies
// Le templateContent doit être une syntaxe de template Go valide
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
