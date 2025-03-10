package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

func writeConfig(cfg *Config, t *testing.T) {
	t.Helper()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}
	err = os.WriteFile(ConfigFilePath(), data, 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}
}

func TestContextExistsAndGetContext(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	// Start with an empty config
	cfg := &Config{
		CurrentContext: "",
		Contexts:       []Context{},
	}
	writeConfig(cfg, t)

	// Should not exist
	exists, err := ContextExists("foo")
	if err != nil {
		t.Fatalf("ContextExists error: %v", err)
	}
	if exists {
		t.Fatalf("expected context 'foo' to not exist")
	}

	// Save a context and test retrieval
	ctx := Context{
		Name:           "foo",
		DockerHostType: ContextLocal,
		DockerSocket:   "/var/run/docker.sock",
		ProjectName:    "myproject",
		ProjectDir:     tempHome,
	}
	err = SaveContext(&ctx, true)
	if err != nil {
		t.Fatalf("SaveContext error: %v", err)
	}

	exists, err = ContextExists("foo")
	if err != nil {
		t.Fatalf("ContextExists error: %v", err)
	}
	if !exists {
		t.Fatalf("expected context 'foo' to exist")
	}

	gotCtx, err := GetContext("foo")
	if err != nil {
		t.Fatalf("GetContext error: %v", err)
	}
	if !contextsEqual(ctx, gotCtx) {
		t.Fatalf("expected context %+v, got %+v", ctx, gotCtx)
	}
}

func contextsEqual(a, b Context) bool {
	return a.Name == b.Name &&
		a.DockerHostType == b.DockerHostType &&
		a.DockerSocket == b.DockerSocket &&
		a.ProjectName == b.ProjectName &&
		a.Profile == b.Profile &&
		a.ProjectDir == b.ProjectDir &&
		a.SSHUser == b.SSHUser &&
		a.SSHHostname == b.SSHHostname &&
		a.SSHPort == b.SSHPort &&
		a.SSHKeyPath == b.SSHKeyPath &&
		a.Site == b.Site &&
		len(a.EnvFile) == len(b.EnvFile) &&
		a.RunSudo == b.RunSudo
}

func TestContextString(t *testing.T) {
	ctx := Context{
		Name:           "test",
		DockerHostType: ContextLocal,
		DockerSocket:   "/var/run/docker.sock",
		ProjectName:    "project",
		ProjectDir:     "/tmp",
	}
	s, err := ctx.String()
	if err != nil {
		t.Fatalf("Context.String error: %v", err)
	}
	// check that YAML output contains expected fields
	if !strings.Contains(s, "test") || !strings.Contains(s, "local") {
		t.Fatalf("unexpected context string: %s", s)
	}
}

func TestSaveContext(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	// Initialize empty config.
	cfg := &Config{
		CurrentContext: "",
		Contexts:       []Context{},
	}
	writeConfig(cfg, t)

	ctx := Context{
		Name:           "myctx",
		DockerHostType: ContextLocal,
		ProjectDir:     tempHome,
	}
	// Test adding new context.
	err := SaveContext(&ctx, true)
	if err != nil {
		t.Fatalf("SaveContext error: %v", err)
	}
	loadedCfg, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if len(loadedCfg.Contexts) != 1 {
		t.Fatalf("expected 1 context, got %d", len(loadedCfg.Contexts))
	}
	if loadedCfg.CurrentContext != "myctx" {
		t.Fatalf("expected current context 'myctx', got %s", loadedCfg.CurrentContext)
	}

	// Test updating context.
	ctx.ProjectName = "updated-project"
	err = SaveContext(&ctx, false)
	if err != nil {
		t.Fatalf("SaveContext error: %v", err)
	}
	loadedCfg, err = Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if loadedCfg.Contexts[0].ProjectName != "updated-project" {
		t.Fatalf("expected updated project name, got %s", loadedCfg.Contexts[0].ProjectName)
	}
}

func TestCurrentContext(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	// Create a config with one context and set it as current.
	cfg := &Config{
		CurrentContext: "default",
		Contexts: []Context{
			{
				Name:           "default",
				DockerHostType: ContextLocal,
				ProjectDir:     tempHome,
			},
		},
	}
	writeConfig(cfg, t)

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.String("context", "default", "test flag")
	ctx, err := CurrentContext(fs)
	if err != nil {
		t.Fatalf("CurrentContext error: %v", err)
	}
	if ctx.Name != "default" {
		t.Fatalf("expected context name 'default', got %s", ctx.Name)
	}

	// Change flag value to a non-existent context.
	err = fs.Set("context", "nonexistent")
	if err != nil {
		t.Fatalf("error setting context flag: %v", err)
	}

	_, err = CurrentContext(fs)
	if err == nil {
		t.Fatalf("expected error for non-existent context")
	}
}

func TestReadSmallFileLocal(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	// Create a temporary file.
	filePath := filepath.Join(tempHome, "test.txt")
	content := "small file content"
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	ctx := &Context{
		DockerHostType: ContextLocal,
	}
	readContent := ctx.ReadSmallFile(filePath)
	if readContent != content {
		t.Fatalf("expected %q, got %q", content, readContent)
	}
}

func TestDialSSHError(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	// Set SSHKeyPath to a non-existent file to force an error.
	ctx := &Context{
		Name:        "ssh-test",
		SSHUser:     "user",
		SSHHostname: "localhost",
		SSHPort:     22,
		SSHKeyPath:  filepath.Join(tempHome, "nonexistent_key"),
	}
	_, err := ctx.DialSSH()
	if err == nil {
		t.Fatalf("expected error from DialSSH with bad SSHKeyPath")
	}
	if !strings.Contains(err.Error(), "error reading SSH key") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProjectDirExistsLocal(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	// Create a temporary directory.
	projectDir := filepath.Join(tempHome, "project")
	err := os.Mkdir(projectDir, 0755)
	if err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}
	ctx := &Context{
		DockerHostType: ContextLocal,
		ProjectDir:     projectDir,
	}
	exists, err := ctx.ProjectDirExists()
	if err != nil {
		t.Fatalf("ProjectDirExists error: %v", err)
	}
	if !exists {
		t.Fatalf("expected project dir %s to exist", projectDir)
	}

	// Test non-existent directory.
	ctx.ProjectDir = filepath.Join(tempHome, "nonexistent")
	exists, err = ctx.ProjectDirExists()
	if err != nil {
		t.Fatalf("ProjectDirExists error: %v", err)
	}
	if exists {
		t.Fatalf("expected project dir %s to not exist", ctx.ProjectDir)
	}
}

func TestUploadFileError(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	// For this test, we simulate an error by forcing DialSSH to fail.
	ctx := &Context{
		SSHUser:     "user",
		SSHHostname: "localhost",
		SSHPort:     22,
		// Point to a non-existent key to force DialSSH error.
		SSHKeyPath: filepath.Join(tempHome, "nonexistent_key"),
	}
	// Create a temporary local file.
	sourcePath := filepath.Join(tempHome, "source.txt")
	err := os.WriteFile(sourcePath, []byte("content"), 0644)
	if err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}
	err = ctx.UploadFile(sourcePath, "/remote/path/dest.txt")
	if err == nil {
		t.Fatalf("expected error from UploadFile due to SSH dial failure")
	}
}

func TestVerifyRemoteInputExistingConfig(t *testing.T) {
	// Simulate empty input for SSH hostname prompt.
	input := "\n"
	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	_, err = inW.WriteString(input)
	if err != nil {
		t.Fatalf("failed to write input: %v", err)
	}
	inW.Close()
	origStdin := os.Stdin
	defer func() { os.Stdin = origStdin }()
	os.Stdin = inR

	original := Context{
		SSHHostname: "foo.example.com.dev",
		SSHUser:     "bar",
		SSHPort:     123,
		SSHKeyPath:  "/assuming/we/already/checked",
		Profile:     "prod",
		ProjectName: "baz",
	}
	cc := original

	err = cc.VerifyRemoteInput(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !contextsEqual(cc, original) {
		t.Fatalf("expected context %+v, got %+v", original, cc)
	}
}
