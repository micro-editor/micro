package config

import (
	"testing"

	"github.com/micro-editor/micro/v2/pkg/highlight"
)

func TestXMLEmptyTagWithSpace(t *testing.T) {
	data, err := Asset("syntax/xml.yaml")
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

	matches := highlight.NewHighlighter(def).HighlightString("<tag attr=\"text\" />\nplain")
	if group := matches[1][0]; group != 0 {
		t.Fatalf("text after empty tag highlighted as %q", group)
	}
}
