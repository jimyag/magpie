package provider

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/yetone/magpie/internal/catalog"
)

func externalCodex(path string, baseTop map[string]string, baseTables map[string]map[string]string) (Provider, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Provider{}, false
	}
	top, tables := parseTOML(string(b))
	providerID := top["model_provider"]
	if providerID == "" {
		providerID = baseTop["model_provider"]
	}
	providerTable := func(ts map[string]map[string]string) map[string]string {
		if p := ts["model_providers."+providerID]; p != nil {
			return p
		}
		return ts[`model_providers."`+providerID+`"`]
	}
	mp := maps.Clone(providerTable(baseTables))
	if mp == nil {
		mp = map[string]string{}
	}
	for k, v := range providerTable(tables) {
		mp[k] = v
	}
	baseURL := mp["base_url"]
	if baseURL == "" {
		baseURL = top["openai_base_url"]
	}
	if baseURL == "" {
		return Provider{}, false
	}
	key := mp["experimental_bearer_token"]
	if key == "" && mp["env_key"] != "" {
		key = os.Getenv(mp["env_key"])
	}
	if key == "" && mp["env_key"] == "" {
		key = os.Getenv("OPENAI_API_KEY")
	}
	eps := endpoints{responses: baseURL}
	if mp["wire_api"] == "chat" {
		eps = endpoints{chat: baseURL}
	}
	model := top["model"]
	if model == "" {
		model = baseTop["model"]
	}
	name := "codex-" + Slug(strings.TrimSuffix(filepath.Base(path), ".config.toml"))
	models := []string{model}
	var sourceModels []catalog.Model
	if catalogPath := top["model_catalog_json"]; catalogPath != "" {
		sourceModels = codexCatalogModels(path, catalogPath)
	} else if catalogPath := baseTop["model_catalog_json"]; catalogPath != "" {
		sourceModels = codexCatalogModels(path, catalogPath)
	}
	for _, m := range sourceModels {
		models = append(models, m.ID)
	}
	p, skip := imported(name, key, eps, models, true)
	if skip != "" {
		return Provider{}, false
	}
	p.ID, p.Source, p.SourceModels = name, path, sourceModels
	return normalize(p), true
}

func codexCatalogModels(profilePath, catalogPath string) []catalog.Model {
	if strings.HasPrefix(catalogPath, "~/") {
		home, _ := os.UserHomeDir()
		catalogPath = filepath.Join(home, strings.TrimPrefix(catalogPath, "~/"))
	} else if !filepath.IsAbs(catalogPath) {
		catalogPath = filepath.Join(filepath.Dir(profilePath), catalogPath)
	}
	b, err := os.ReadFile(catalogPath)
	if err != nil {
		return nil
	}
	var list struct {
		Models []struct {
			Slug        string `json:"slug"`
			DisplayName string `json:"display_name"`
			Visibility  string `json:"visibility"`
			Levels      []struct {
				Effort string `json:"effort"`
			} `json:"supported_reasoning_levels"`
		} `json:"models"`
	}
	if json.Unmarshal(b, &list) != nil {
		return nil
	}
	var models []catalog.Model
	for _, m := range list.Models {
		if m.Visibility != "hide" && m.Slug != "" {
			model := catalog.Model{ID: m.Slug, Name: m.DisplayName}
			if model.Name == "" {
				model.Name = model.ID
			}
			for _, level := range m.Levels {
				model.Efforts = append(model.Efforts, level.Effort)
			}
			models = append(models, model)
		}
	}
	return models
}
