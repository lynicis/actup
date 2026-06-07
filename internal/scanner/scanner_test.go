package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverWorkflows(t *testing.T) {
	tmpDir := t.TempDir()

	workflowDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		t.Fatalf("failed to create workflow dir: %v", err)
	}

	workflows := []struct {
		name    string
		content string
	}{
		{"ci.yml", "name: CI\non: push\njobs:\n  build:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v3\n"},
		{"test.yml", "name: Test\non: pull_request\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/setup-go@v4\n"},
	}

	for _, wf := range workflows {
		path := filepath.Join(workflowDir, wf.name)
		if err := os.WriteFile(path, []byte(wf.content), 0644); err != nil {
			t.Fatalf("failed to write workflow %s: %v", wf.name, err)
		}
	}

	files, err := DiscoverWorkflows(t.Context(), []string{workflowDir})
	if err != nil {
		t.Fatalf("DiscoverWorkflows failed: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("expected 2 workflows, got %d", len(files))
	}
}

func TestIsWorkflowFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"test.yml", true},
		{"test.yaml", true},
		{"TEST.YML", true},
		{"test.YAML", true},
		{"test.txt", false},
		{"test.json", false},
		{"test", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isWorkflowFile(tt.path)
			if result != tt.expected {
				t.Errorf("isWorkflowFile(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestDiscoverWorkflowsFilePath(t *testing.T) {
	tmpDir := t.TempDir()

	workflowPath := filepath.Join(tmpDir, "workflow.yml")
	content := "name: Test\non: push\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v3\n"

	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	files, err := DiscoverWorkflows(t.Context(), []string{workflowPath})
	if err != nil {
		t.Fatalf("DiscoverWorkflows failed: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("expected 1 workflow, got %d", len(files))
	}

	if files[0] != workflowPath {
		t.Errorf("expected %s, got %s", workflowPath, files[0])
	}
}

func TestDiscoverWorkflowsEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := DiscoverWorkflows(t.Context(), []string{filepath.Join(tmpDir, "nonexistent")})
	if err != nil {
		t.Fatalf("DiscoverWorkflows should not fail for nonexistent path: %v", err)
	}
}