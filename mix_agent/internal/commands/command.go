package commands

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Command represents a slash command
type Command interface {
	Name() string
	Description() string
	Execute(ctx context.Context, args string) (string, error)
}

// FileCommand represents a command loaded from a .md file
type FileCommand struct {
	name        string
	description string
	content     string
	metadata    CommandMetadata
	filePath    string
}

// CommandMetadata represents YAML frontmatter in command files
type CommandMetadata struct {
	Description  string   `yaml:"description"`
	ArgumentHint string   `yaml:"argument-hint"`
	AllowedTools []string `yaml:"allowed-tools"`
}

// NewFileCommand creates a command from a markdown file
func NewFileCommand(name, filePath string) (*FileCommand, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read command file %s: %w", filePath, err)
	}

	cmd := &FileCommand{
		name:     name,
		filePath: filePath,
	}

	// Parse frontmatter and content
	if err := cmd.parseFile(string(content)); err != nil {
		return nil, fmt.Errorf("failed to parse command file %s: %w", filePath, err)
	}

	return cmd, nil
}

func (c *FileCommand) parseFile(content string) error {
	lines := strings.Split(content, "\n")

	// Check for YAML frontmatter
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		var yamlEnd int
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				yamlEnd = i
				break
			}
		}

		if yamlEnd > 0 {
			// Parse YAML frontmatter
			yamlContent := strings.Join(lines[1:yamlEnd], "\n")
			if err := yaml.Unmarshal([]byte(yamlContent), &c.metadata); err != nil {
				return fmt.Errorf("failed to parse YAML frontmatter: %w", err)
			}

			// Rest is content
			c.content = strings.TrimSpace(strings.Join(lines[yamlEnd+1:], "\n"))
		} else {
			// No closing ---, treat as regular content
			c.content = content
		}
	} else {
		// No frontmatter, entire content is the prompt
		c.content = content
	}

	// Set description from metadata or use default
	if c.metadata.Description != "" {
		c.description = c.metadata.Description
	} else {
		c.description = fmt.Sprintf("Custom command from %s", filepath.Base(c.filePath))
	}

	return nil
}

func (c *FileCommand) Name() string {
	return c.name
}

func (c *FileCommand) Description() string {
	return c.description
}

func (c *FileCommand) Execute(ctx context.Context, args string) (string, error) {
	// Substitute $ARGUMENTS placeholder
	prompt := strings.ReplaceAll(c.content, "$ARGUMENTS", args)

	// Return the processed prompt for execution by the agent
	return prompt, nil
}

// LoadCommandsFromDirectory loads commands from a directory
func LoadCommandsFromDirectory(dir string) (map[string]Command, error) {
	commands := make(map[string]Command)

	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return commands, nil // Return empty map if directory doesn't exist
	}

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-.md files
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		// Get command name from filename
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		// Remove .md extension and use as command name
		name := strings.TrimSuffix(relPath, ".md")
		// Replace path separators with colons for namespacing
		name = strings.ReplaceAll(name, string(filepath.Separator), ":")

		cmd, err := NewFileCommand(name, path)
		if err != nil {
			return fmt.Errorf("failed to load command %s: %w", name, err)
		}

		commands[name] = cmd
		return nil
	})

	return commands, err
}
