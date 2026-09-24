package provider

import (
	"os"
	"path/filepath"
)

// externalProviders reads the agent configs on each lookup so file edits take
// effect without copying credentials into magpie's store.
// ponytail: scan small config directories per lookup; cache by file mtime if routing throughput suffers.
func externalProviders() []Provider {
	home, _ := os.UserHomeDir()
	var out []Provider
	claudeFiles, _ := filepath.Glob(filepath.Join(home, ".claude", "providers", "*.json"))
	for _, path := range claudeFiles {
		if p, ok := externalClaude(path); ok {
			out = append(out, p)
		}
	}
	codexHome := os.Getenv("CODEX_HOME")
	if codexHome == "" {
		codexHome = filepath.Join(home, ".codex")
	}
	base, _ := os.ReadFile(filepath.Join(codexHome, "config.toml"))
	baseTop, baseTables := parseTOML(string(base))
	codexFiles, _ := filepath.Glob(filepath.Join(codexHome, "*.config.toml"))
	for _, path := range codexFiles {
		if p, ok := externalCodex(path, baseTop, baseTables); ok {
			out = append(out, p)
		}
	}
	return out
}

func externalPath(id string) string {
	for _, p := range externalProviders() {
		if p.ID == id {
			return p.Source
		}
	}
	return ""
}
