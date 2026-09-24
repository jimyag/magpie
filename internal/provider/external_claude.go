package provider

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/tidwall/jsonc"
)

func externalClaude(path string) (Provider, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Provider{}, false
	}
	var settings map[string]any
	if json.Unmarshal(jsonc.ToJSON(b), &settings) != nil {
		return Provider{}, false
	}
	env := mapOf(settings["env"])
	name := "claude-" + Slug(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	base, _ := env["ANTHROPIC_BASE_URL"].(string)
	key, _ := env["ANTHROPIC_AUTH_TOKEN"].(string)
	if key == "" {
		key, _ = env["ANTHROPIC_API_KEY"].(string)
	}
	var models []string
	if m, ok := env["ANTHROPIC_MODEL"].(string); ok {
		models = append(models, m)
	}
	for _, k := range slices.Sorted(maps.Keys(env)) {
		if k != "ANTHROPIC_MODEL" && (strings.HasPrefix(k, "ANTHROPIC_") && strings.HasSuffix(k, "_MODEL") || k == "CLAUDE_CODE_SUBAGENT_MODEL") {
			if m, ok := env[k].(string); ok {
				models = append(models, m)
			}
		}
	}
	if len(models) == 0 {
		if m, ok := settings["model"].(string); ok {
			models = append(models, m)
		}
	}
	p, skip := imported(name, key, endpoints{anthropic: base}, models, true)
	if skip != "" {
		return Provider{}, false
	}
	p.ID, p.Source = name, path
	return normalize(p), true
}
