package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// LocalGitService implements GitService by wrapping the git CLI via os/exec.
// This is the only file in the codebase that should call os/exec for git.
type LocalGitService struct{}

// NewLocalGitService creates a new LocalGitService.
func NewLocalGitService() *LocalGitService {
	return &LocalGitService{}
}

// PreviewMerge uses "git merge-tree" (Git 2.38+) to simulate a merge without
// touching the working tree or index. This is a clean, side-effect-free preview.
func (s *LocalGitService) PreviewMerge(repoPath string, branches []string, targetBranch string) (*MergePreview, error) {
	if len(branches) == 0 {
		return &MergePreview{}, nil
	}

	var allConflicts []string
	var allClean []string

	for _, branch := range branches {
		conflicts, clean, err := s.previewOneBranch(repoPath, branch, targetBranch)
		if err != nil {
			return nil, fmt.Errorf("preview merge of %s → %s: %w", branch, targetBranch, err)
		}
		allConflicts = append(allConflicts, conflicts...)
		allClean = append(allClean, clean...)
	}

	return &MergePreview{
		HasConflicts:     len(allConflicts) > 0,
		ConflictingFiles: allConflicts,
		CleanFiles:       allClean,
	}, nil
}

// previewOneBranch runs git merge-tree for a single branch into targetBranch.
func (s *LocalGitService) previewOneBranch(repoPath, branch, targetBranch string) (conflicts, clean []string, err error) {
	// git merge-tree --write-tree returns exit 0 (no conflicts) or exit 1 (conflicts)
	// --name-only lists conflicting files
	cmd := exec.Command("git", "-C", repoPath, "merge-tree", "--no-messages",
		"--merge-base="+targetBranch, targetBranch, branch)
	out, err := cmd.Output()

	if err != nil {
		// Exit code 1 from git merge-tree means conflicts exist
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// Parse conflict output
			lines := strings.Split(string(out), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" && !strings.HasPrefix(line, "warning:") {
					conflicts = append(conflicts, line)
				}
			}
			return conflicts, nil, nil
		}
		return nil, nil, fmt.Errorf("git merge-tree: %w", err)
	}

	// No conflicts — parse clean files from the output
	// In Git 2.38+, merge-tree outputs the merged tree SHA on success
	// We use diff-tree to find which files were changed in the branch
	diffCmd := exec.Command("git", "-C", repoPath, "diff", "--name-only", targetBranch+"..."+branch)
	diffOut, err := diffCmd.Output()
	if err != nil {
		return nil, nil, fmt.Errorf("git diff --name-only: %w", err)
	}

	for _, line := range strings.Split(string(diffOut), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			clean = append(clean, line)
		}
	}

	return nil, clean, nil
}

// CreateBranch creates a new branch in the given repo.
func (s *LocalGitService) CreateBranch(repoPath string, branchName string, baseBranch string) error {
	cmd := exec.Command("git", "-C", repoPath, "checkout", "-b", branchName, baseBranch)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git checkout -b %s %s: %w\n%s", branchName, baseBranch, err, string(out))
	}
	return nil
}

// ListBranches returns branches matching the given pattern.
func (s *LocalGitService) ListBranches(repoPath string, pattern string) ([]string, error) {
	cmd := exec.Command("git", "-C", repoPath, "branch", "--list", pattern)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git branch --list %s: %w", pattern, err)
	}

	var branches []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "*"))
		line = strings.TrimSpace(line)
		if line != "" {
			branches = append(branches, line)
		}
	}

	return branches, nil
}
