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

// ActivationStyle describes how likely a skill description is to trigger
// Claude's automatic invocation based on community research into description wording.
type ActivationStyle int

const (
	// ActivationNeutral is the default when the description doesn't clearly signal
	// directive or passive intent.
	ActivationNeutral ActivationStyle = iota
	// ActivationDirective means the description uses imperative language
	// ("ALWAYS invoke", "MUST use") which correlates with ~98% auto-activation rates.
	ActivationDirective
	// ActivationPassive means the description uses descriptive language
	// ("Use when", "Helps with") which correlates with ~69% auto-activation rates.
	ActivationPassive
)

// AssessActivationStyle returns the invocation style based on description wording.
// Directive descriptions activate reliably; passive descriptions are often ignored.
func AssessActivationStyle(description string) ActivationStyle {
	upper := strings.ToUpper(strings.TrimSpace(description))
	for _, kw := range []string{"ALWAYS ", "MUST ", "NEVER ", "DO NOT "} {
		if strings.Contains(upper, kw) {
			return ActivationDirective
		}
	}
	for _, kw := range []string{"USE WHEN", "HELPS ", "CAN BE USED", "USEFUL FOR", "ASSISTS "} {
		if strings.Contains(upper, kw) {
			return ActivationPassive
		}
	}
	return ActivationNeutral
}

// Skill represents a single discovered Claude Code skill.
type Skill struct {
	Name            string
	Description     string
	Plugin          string
	FilePath        string
	Content         string
	Frontmatter     string
	ActivationStyle ActivationStyle
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

		fm, rawFM, body, err := parseFrontmatter(content)
		if err != nil {
			continue
		}

		name := fm.Name
		if name == "" {
			name = entry.Name()
		}

		skills = append(skills, Skill{
			Name:            name,
			Description:     fm.Description,
			Plugin:          pluginName,
			FilePath:        skillFile,
			Content:         body,
			Frontmatter:     rawFM,
			ActivationStyle: AssessActivationStyle(fm.Description),
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

func parseFrontmatter(content []byte) (fm frontmatter, rawYAML string, body string, err error) {
	trimmed := bytes.TrimSpace(content)
	if !bytes.HasPrefix(trimmed, []byte("---")) {
		return fm, "", string(content), nil
	}

	rest := trimmed[3:]
	idx := bytes.Index(rest, []byte("\n---"))
	if idx == -1 {
		return fm, "", string(content), nil
	}

	yamlBlock := rest[:idx]
	bodyBytes := rest[idx+4:]

	if err = yaml.Unmarshal(yamlBlock, &fm); err != nil {
		return fm, "", string(content), err
	}

	return fm, string(bytes.TrimSpace(yamlBlock)), string(bytes.TrimSpace(bodyBytes)), nil
}
