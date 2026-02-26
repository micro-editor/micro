package config

import (
	"testing"

	highlight "github.com/micro-editor/micro/v2/pkg/highlight"
	"github.com/stretchr/testify/assert"
)

func TestAssetDir(t *testing.T) {
	t.Parallel()
	// Test AssetDir
	entries, err := AssetDir("syntax")
	assert.NoError(t, err)
	assert.Contains(t, entries, "go.yaml")
	assert.True(t, len(entries) > 5)
}

func TestINISyntaxMatchesConfFiles(t *testing.T) {
	// Regression test for https://github.com/micro-editor/micro/issues/4004
	// .conf files should be highlighted with INI syntax
	t.Parallel()

	data, err := Asset("syntax/ini.yaml")
	assert.NoError(t, err)

	header, err := highlight.MakeHeaderYaml(data)
	assert.NoError(t, err)

	// .conf files should match
	assert.True(t, header.MatchFileName("sysctl.conf"), "sysctl.conf should match INI syntax")
	assert.True(t, header.MatchFileName("nginx.conf"), "nginx.conf should match INI syntax")
	assert.True(t, header.MatchFileName("test.conf"), "test.conf should match INI syntax")

	// weechat .conf files should still match
	assert.True(t, header.MatchFileName("weechat/irc.conf"), "weechat/irc.conf should match INI syntax")

	// Other INI extensions should still work
	assert.True(t, header.MatchFileName("test.ini"), "test.ini should match INI syntax")
	assert.True(t, header.MatchFileName("test.desktop"), "test.desktop should match INI syntax")

	// Non-INI files should not match
	assert.False(t, header.MatchFileName("test.go"), "test.go should not match INI syntax")
	assert.False(t, header.MatchFileName("test.py"), "test.py should not match INI syntax")
}
