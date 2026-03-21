package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// installCompletion writes the zsh completion file to ~/.raptor/completions
// and ensures ~/.zshrc sources it. For bash, it writes to ~/.raptor/completions
// and appends to ~/.bashrc.
func installCompletion() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	completionDir := filepath.Join(home, ".raptor", "completions")
	os.MkdirAll(completionDir, 0o755)

	// Detect shell from SHELL env
	shell := filepath.Base(os.Getenv("SHELL"))

	switch shell {
	case "zsh":
		installZshCompletion(home, completionDir)
	case "bash":
		installBashCompletion(home, completionDir)
	case "fish":
		installFishCompletion(home)
	}
}

func installZshCompletion(home, completionDir string) {
	completionFile := filepath.Join(completionDir, "_raptor")
	f, err := os.Create(completionFile)
	if err != nil {
		return
	}
	defer f.Close()
	rootCmd.GenZshCompletion(f)

	// Add to .zshrc if not already there
	rcPath := filepath.Join(home, ".zshrc")
	line := fmt.Sprintf("\nfpath=(%s $fpath)\nautoload -Uz compinit && compinit\n", completionDir)
	marker := completionDir

	appendIfMissing(rcPath, marker, line)
	fmt.Println("Shell completions installed (restart your shell or run: source ~/.zshrc)")
}

func installBashCompletion(home, completionDir string) {
	completionFile := filepath.Join(completionDir, "raptor.bash")
	f, err := os.Create(completionFile)
	if err != nil {
		return
	}
	defer f.Close()
	rootCmd.GenBashCompletion(f)

	rcPath := filepath.Join(home, ".bashrc")
	line := fmt.Sprintf("\nsource %s\n", completionFile)
	marker := completionFile

	appendIfMissing(rcPath, marker, line)
	fmt.Println("Shell completions installed (restart your shell or run: source ~/.bashrc)")
}

func installFishCompletion(home string) {
	completionDir := filepath.Join(home, ".config", "fish", "completions")
	os.MkdirAll(completionDir, 0o755)

	completionFile := filepath.Join(completionDir, "raptor.fish")
	f, err := os.Create(completionFile)
	if err != nil {
		return
	}
	defer f.Close()
	rootCmd.GenFishCompletion(f, true)
	fmt.Println("Shell completions installed (restart your shell to activate)")
}

// appendIfMissing appends line to the file at path if marker is not already present.
func appendIfMissing(path, marker, line string) {
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), marker) {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(line)
}
