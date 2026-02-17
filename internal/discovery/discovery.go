package discovery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Skill represents a single discovered Claude Code skill.
type Skill struct {
	Name        string
	Description string
	Plugin      string
	FilePath    string
	Content     string
}

type installedPlugins struct {
	Version int                         `json:"version"`
	Plugins map[string][]pluginInstance `json:"plugins"`
}

type pluginInstance struct {
	InstallPath string `json:"installPath"`
	Version     string `json:"version"`
}

type frontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// discoverSkillsInDir walks subdirectories of dir, reads SKILL.md files,
// and returns discovered skills. Returns nil if dir doesn't exist.
func discoverSkillsInDir(dir string, pluginName string) []Skill {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var skills []Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillFile := filepath.Join(dir, entry.Name(), "SKILL.md")
		content, err := os.ReadFile(skillFile)
		if err != nil {
			continue
		}

		fm, body, err := parseFrontmatter(content)
		if err != nil {
			continue
		}

		name := fm.Name
		if name == "" {
			name = entry.Name()
		}

		skills = append(skills, Skill{
			Name:        name,
			Description: fm.Description,
			Plugin:      pluginName,
			FilePath:    skillFile,
			Content:     body,
		})
	}
	return skills
}

// LocalSkillsDir pairs a .claude/skills path with a display name.
type LocalSkillsDir struct {
	Path string
	Name string
}

// Discover reads installed_plugins.json and finds all skills.
// Skills from localDirs are also included, each labeled with its Name.
func Discover(pluginsFile string, localDirs []LocalSkillsDir) ([]Skill, error) {
	data, err := os.ReadFile(pluginsFile)
	if err != nil {
		return nil, fmt.Errorf("reading plugins file: %w", err)
	}

	var installed installedPlugins
	if err := json.Unmarshal(data, &installed); err != nil {
		return nil, fmt.Errorf("parsing plugins file: %w", err)
	}

	var skills []Skill
	for key, instances := range installed.Plugins {
		if len(instances) == 0 {
			continue
		}
		inst := instances[0]

		pluginName := key
		if idx := strings.Index(key, "@"); idx != -1 {
			pluginName = key[:idx]
		}

		skills = append(skills, discoverSkillsInDir(filepath.Join(inst.InstallPath, "skills"), pluginName)...)
	}

	for _, d := range localDirs {
		skills = append(skills, discoverSkillsInDir(d.Path, d.Name)...)
	}

	return skills, nil
}

func parseFrontmatter(content []byte) (frontmatter, string, error) {
	var fm frontmatter

	trimmed := bytes.TrimSpace(content)
	if !bytes.HasPrefix(trimmed, []byte("---")) {
		return fm, string(content), nil
	}

	rest := trimmed[3:]
	idx := bytes.Index(rest, []byte("\n---"))
	if idx == -1 {
		return fm, string(content), nil
	}

	yamlBlock := rest[:idx]
	body := rest[idx+4:]

	if err := yaml.Unmarshal(yamlBlock, &fm); err != nil {
		return fm, string(content), err
	}

	return fm, string(bytes.TrimSpace(body)), nil
}
