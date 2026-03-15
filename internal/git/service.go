package git

// MergePreview holds the result of a local git merge preview.
type MergePreview struct {
	HasConflicts     bool     `json:"has_conflicts"`
	ConflictingFiles []string `json:"conflicting_files"`
	CleanFiles       []string `json:"clean_files"`
}

// GitService abstracts git operations performed client-side.
// The server does NOT implement this interface — it has zero git dependency.
// The CLI client creates a LocalGitService for local operations.
type GitService interface {
	// PreviewMerge simulates merging branches into targetBranch without modifying
	// the working tree or index. Returns conflict information.
	PreviewMerge(repoPath string, branches []string, targetBranch string) (*MergePreview, error)

	// CreateBranch creates a new branch at baseBranch in the given repo.
	CreateBranch(repoPath string, branchName string, baseBranch string) error

	// ListBranches returns branches matching the given pattern (e.g., "othrys/*").
	ListBranches(repoPath string, pattern string) ([]string, error)
}
