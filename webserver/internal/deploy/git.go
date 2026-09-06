package deploy

import (
	"fmt"
	"os"
	"os/exec"
)

// syncRepo clones url at branch into dest if it doesn't exist yet, otherwise
// fetches and hard-resets dest to origin/branch.
func syncRepo(url, branch, dest string) error {
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		return runGit("", "clone", "--branch", branch, "--single-branch", url, dest)
	} else if err != nil {
		return err
	}

	if err := runGit(dest, "fetch", "origin", branch); err != nil {
		return err
	}
	return runGit(dest, "reset", "--hard", "origin/"+branch)
}

func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %v: %w", args, err)
	}
	return nil
}
