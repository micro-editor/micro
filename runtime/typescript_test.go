package config

import (
	"testing"

	"github.com/micro-editor/micro/v2/pkg/highlight"
)

func TestTypeScriptDivisionIsNotRegexp(t *testing.T) {
	data, err := Asset("syntax/typescript.yaml")
	if err != nil {
		t.Fatal(err)
	}
	file, err := highlight.ParseFile(data)
	if err != nil {
		t.Fatal(err)
	}
	def, err := highlight.ParseDef(file, nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, group := range highlight.NewHighlighter(def).HighlightString("const n = 10 / 5 / 2;")[0] {
		if group.String() == "constant" {
			t.Fatal("division highlighted as a regular expression")
		}
	}
}
