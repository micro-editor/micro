package config

import (
	"reflect"
	"testing"

	"github.com/micro-editor/micro/v2/pkg/highlight"
)

func TestC23IntegerLiteralSeparators(t *testing.T) {
	runtimeFile := FindRuntimeFile(RTSyntax, "c")
	if runtimeFile == nil {
		t.Fatal("C syntax file not found")
	}
	data, err := runtimeFile.Data()
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

	literals := []string{
		"42",
		"077",
		"0xFFFF",
		"0b1010",
		"1'000'000",
		"0'7",
		"0xFFFF'FFFF",
		"0Xabc'DEFu",
		"0b1010'0101",
		"0B1'0LL",
	}
	want := highlight.LineMatch{0: highlight.Groups["constant.number"]}

	for _, literal := range literals {
		t.Run(literal, func(t *testing.T) {
			got := highlight.NewHighlighter(def).HighlightString(literal)[0]
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("HighlightString(%q) = %v, want %v", literal, got, want)
			}
		})
	}
}
