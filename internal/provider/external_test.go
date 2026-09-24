package provider

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestExternalProviders(t *testing.T) {
	isolate(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	t.Setenv("CODEX_HOME", filepath.Join(home, "codex"))
	t.Setenv("DS_TEST_KEY", "sk-from-env")
	claudeFile := filepath.Join(home, ".claude", "providers", "aliyun.json")
	codexFile := filepath.Join(home, "codex", "ds.config.toml")
	for _, path := range []string{claudeFile, codexFile} {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(claudeFile, []byte(`{"env":{"ANTHROPIC_BASE_URL":"https://relay.example.com/anthropic","ANTHROPIC_AUTH_TOKEN":"sk-claude","ANTHROPIC_MODEL":"claude-sonnet-5","CLAUDE_CODE_SUBAGENT_MODEL":"claude-haiku-5"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "codex", "config.toml"), []byte("[model_providers.deepseek]\nbase_url = \"https://relay.example.com/v1\"\nwire_api = \"responses\"\nenv_key = \"DS_TEST_KEY\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codexFile, []byte("model_provider = \"deepseek\"\nmodel = \"deepseek-chat\"\nmodel_catalog_json = \"models.json\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "codex", "models.json"), []byte(`{"models":[{"slug":"deepseek-chat","visibility":"list"},{"slug":"deepseek-reasoner","display_name":"DeepSeek Reasoner","visibility":"list","supported_reasoning_levels":[{"effort":"high"}]},{"slug":"old-model","visibility":"hide"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	claude, err := Find("claude-aliyun")
	if err != nil || !claude.Ready() || claude.Source != claudeFile || claude.Key != "sk-claude" || !slices.Contains(claude.Models, "claude-haiku-5") {
		t.Fatalf("Claude config: %+v, %v", claude, err)
	}
	codex, err := Find("codex-ds")
	if err != nil || !codex.Ready() || codex.Source != codexFile || codex.Key != "sk-from-env" || codex.Responses != "https://relay.example.com/v1" || !slices.Contains(codex.Models, "deepseek-reasoner") || slices.Contains(codex.Models, "old-model") {
		t.Fatalf("Codex profile: %+v, %v", codex, err)
	}
	if models := codex.Available(); len(models) != 2 || models[1].Name != "DeepSeek Reasoner" || !slices.Equal(models[1].Efforts, []string{"high"}) {
		t.Fatalf("Codex catalog metadata: %+v", models)
	}
	if p, model, ok := Resolve("codex-ds/deepseek-chat"); !ok || p.ID != "codex-ds" || model != "deepseek-chat" {
		t.Fatalf("resolve: %+v %q %v", p, model, ok)
	}
	if p, model, ok := Resolve("codex-ds/deepseek-reasoner"); !ok || p.ID != "codex-ds" || model != "deepseek-reasoner" {
		t.Fatalf("catalog model: %+v %q %v", p, model, ok)
	}
	if err := Save(*codex); err == nil {
		t.Fatal("saved a read-only profile")
	}
	if err := Delete("claude-aliyun"); err == nil {
		t.Fatal("deleted a read-only config")
	}
	if _, err := os.Stat(Path()); !os.IsNotExist(err) {
		t.Fatalf("external providers copied into magpie's store: %v", err)
	}
	t.Setenv("DS_TEST_KEY", "")
	codex, err = Find("codex-ds")
	if err != nil || codex.Ready() {
		t.Fatalf("missing environment key: %+v, %v", codex, err)
	}
	if err := os.Remove(claudeFile); err != nil {
		t.Fatal(err)
	}
	if _, err := Find("claude-aliyun"); err == nil {
		t.Fatal("removed config still available")
	}
	if err := os.WriteFile(claudeFile, []byte(`{"env":{"ANTHROPIC_BASE_URL":"https://relay.example.com/anthropic","ANTHROPIC_MODEL":"claude-sonnet-5"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	claude, err = Find("claude-aliyun")
	if err != nil || claude.Ready() || claude.Source != claudeFile {
		t.Fatalf("keyless Claude config: %+v, %v", claude, err)
	}
}
