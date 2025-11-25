package decision

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPromptManager(t *testing.T) {
	// 1. Setup temporary directory for prompts
	tempDir, err := os.MkdirTemp("", "prompts_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir) // Clean up

	// 2. Create dummy prompt files
	prompt1Content := "This is prompt 1 content"
	prompt2Content := "This is prompt 2 content"

	err = os.WriteFile(filepath.Join(tempDir, "prompt1.txt"), []byte(prompt1Content), 0644)
	assert.NoError(t, err)

	err = os.WriteFile(filepath.Join(tempDir, "prompt2.txt"), []byte(prompt2Content), 0644)
	assert.NoError(t, err)

	// Create a non-txt file (should be ignored)
	err = os.WriteFile(filepath.Join(tempDir, "ignored.md"), []byte("ignored"), 0644)
	assert.NoError(t, err)

	// 3. Initialize PromptManager
	pm := NewPromptManager()

	// 4. Test LoadTemplates
	err = pm.LoadTemplates(tempDir)
	assert.NoError(t, err)

	// 5. Test GetTemplate
	// Should find prompt1
	tmpl1, err := pm.GetTemplate("prompt1")
	assert.NoError(t, err)
	assert.NotNil(t, tmpl1)
	assert.Equal(t, "prompt1", tmpl1.Name)
	assert.Equal(t, prompt1Content, tmpl1.Content)

	// Should find prompt2
	tmpl2, err := pm.GetTemplate("prompt2")
	assert.NoError(t, err)
	assert.NotNil(t, tmpl2)
	assert.Equal(t, "prompt2", tmpl2.Name)
	assert.Equal(t, prompt2Content, tmpl2.Content)

	// Should NOT find ignored file
	_, err = pm.GetTemplate("ignored")
	assert.Error(t, err)

	// Should NOT find non-existent template
	_, err = pm.GetTemplate("non_existent")
	assert.Error(t, err)

	// 6. Test GetAllTemplateNames
	names := pm.GetAllTemplateNames()
	assert.Len(t, names, 2)
	assert.Contains(t, names, "prompt1")
	assert.Contains(t, names, "prompt2")

	// 7. Test ReloadTemplates
	// Add a new file
	prompt3Content := "New prompt 3"
	err = os.WriteFile(filepath.Join(tempDir, "prompt3.txt"), []byte(prompt3Content), 0644)
	assert.NoError(t, err)

	err = pm.ReloadTemplates(tempDir)
	assert.NoError(t, err)

	// Should now find prompt3
	tmpl3, err := pm.GetTemplate("prompt3")
	assert.NoError(t, err)
	assert.Equal(t, prompt3Content, tmpl3.Content)
}

func TestPromptManager_LoadNonExistentDir(t *testing.T) {
	pm := NewPromptManager()
	err := pm.LoadTemplates("/path/to/non/existent/dir/hopefully")
	assert.Error(t, err)
}
