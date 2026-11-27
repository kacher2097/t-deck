package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"t-deck/internal/apperr"
	"t-deck/internal/config"
	"t-deck/internal/fsutil"
	"t-deck/internal/model"
)

// ProjectStore defines the persistence operations required by Scanner (DIP).
type ProjectStore interface {
	GetProjectModel(id string) (model.Project, bool, error)
	ListAllLocalStates() (map[string]model.LocalState, error)
	SlugExists(slug string, excludeID string) (bool, error)
	UpsertProject(p model.Project, local *model.LocalState) error
	UpsertLocalState(st model.LocalState) error
	UpdateLocalStatus(projectID string, status string) error
}

// ScanOptions configures workspace scanning, recency filtering, and exclusions.
type ScanOptions struct {
	WorkspaceDirs    []string
	AppsDir          string
	RecentDays       int
	ExtraExcludeDirs []string
}

// Scanner discovers local projects, creates .tdeck-meta.json anchors, and relinks moved folders.
type Scanner struct {
	repo ProjectStore
	mu   sync.Mutex
}

// NewScanner creates a Scanner backed by the given ProjectStore.
func NewScanner(repo ProjectStore) *Scanner {
	return &Scanner{repo: repo}
}

// Scan traverses workspaceDirs and appsDir recursively without recency cutoff.
func (s *Scanner) Scan(workspaceDirs []string, appsDir string) (model.ScanReport, error) {
	return s.ScanWithOptions(ScanOptions{
		WorkspaceDirs: workspaceDirs,
		AppsDir:       appsDir,
	})
}

// ScanWithOptions traverses workspaceDirs and appsDir with optional recency and exclusion filters.
func (s *Scanner) ScanWithOptions(opts ScanOptions) (model.ScanReport, error) {
	report := model.ScanReport{
		Errors: make([]string, 0),
	}

	roots := deduplicateRoots(opts.WorkspaceDirs, opts.AppsDir)
	rootSet := make(map[string]struct{}, len(roots))
	for _, r := range roots {
		rootSet[strings.ToLower(filepath.Clean(r))] = struct{}{}
	}
	seenProjectIDs := make(map[string]struct{})
	visitedDirs := make(map[string]struct{})
	appsRootNorm := ""
	if strings.TrimSpace(opts.AppsDir) != "" {
		appsRootNorm = strings.ToLower(filepath.Clean(opts.AppsDir))
	}

	var cutoff time.Time
	if opts.RecentDays > 0 {
		cutoff = time.Now().Add(-time.Duration(opts.RecentDays) * 24 * time.Hour)
	}

	excludeSet := buildExcludeSet(opts.ExtraExcludeDirs)
	existingStates, err := s.repo.ListAllLocalStates()
	if err != nil {
		existingStates = make(map[string]model.LocalState)
	}

	if len(roots) <= 1 {
		for _, root := range roots {
			if !fsutil.DirExists(root) {
				report.Errors = append(report.Errors, fmt.Sprintf("root not accessible: %s", root))
				continue
			}
			isAppsRoot := appsRootNorm != "" && strings.ToLower(root) == appsRootNorm
			s.walkRoot(root, 0, isAppsRoot, cutoff, excludeSet, rootSet, existingStates, seenProjectIDs, visitedDirs, &report)
		}
	} else {
		var wg sync.WaitGroup
		sem := make(chan struct{}, 4)
		for _, root := range roots {
			root := root
			if !fsutil.DirExists(root) {
				s.mu.Lock()
				report.Errors = append(report.Errors, fmt.Sprintf("root not accessible: %s", root))
				s.mu.Unlock()
				continue
			}
			isAppsRoot := appsRootNorm != "" && strings.ToLower(root) == appsRootNorm
			wg.Add(1)
			go func() {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				defer apperr.Recover("scanner-worker:"+root, nil, func(scope string, recovered any, _ string) {
					s.mu.Lock()
					report.Errors = append(report.Errors, fmt.Sprintf("%s panic: %v", scope, recovered))
					s.mu.Unlock()
				})
				s.walkRoot(root, 0, isAppsRoot, cutoff, excludeSet, rootSet, existingStates, seenProjectIDs, visitedDirs, &report)
			}()
		}
		wg.Wait()
	}

	s.markMissingProjects(existingStates, seenProjectIDs, appsRootNorm, excludeSet, &report)
	return report, nil
}

func (s *Scanner) walkRoot(
	dir string,
	depth int,
	isAppsRoot bool,
	cutoff time.Time,
	excludeSet map[string]struct{},
	rootSet map[string]struct{},
	existingStates map[string]model.LocalState,
	seenIDs map[string]struct{},
	visitedDirs map[string]struct{},
	report *model.ScanReport,
) {
	normDir := strings.ToLower(filepath.Clean(dir))
	if depth > 0 {
		// If a child directory is itself a configured workspace root, let that root walk it directly
		// instead of misclassifying the container root as a single project.
		if _, isRoot := rootSet[normDir]; isRoot {
			return
		}
	}
	s.mu.Lock()
	if _, already := visitedDirs[normDir]; already {
		s.mu.Unlock()
		return
	}
	visitedDirs[normDir] = struct{}{}
	report.ScannedDirs++
	s.mu.Unlock()

	if depth > 0 {
		if !isAppsRoot && isInstalledAppOrSDKDir(dir) {
			s.cleanupBogusMeta(dir)
			return
		}
		if !isAppsRoot && isGenericSubdirProjectName(dir) {
			s.cleanupBogusMeta(dir)
			return
		}
		if !isAppsRoot {
			if isContainer, _ := inspectWorkspaceContainer(dir); isContainer {
				s.cleanupBogusMeta(dir)
				// Fall through to descend into child projects instead of treating container as one project.
			} else if handled := s.inspectDir(dir, isAppsRoot, cutoff, existingStates, seenIDs, report); handled {
				return
			}
		} else if handled := s.inspectDir(dir, isAppsRoot, cutoff, existingStates, seenIDs, report); handled {
			return
		}
	}

	if depth >= config.MaxScanDepth {
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		s.mu.Lock()
		report.Errors = append(report.Errors, fmt.Sprintf("read dir %s: %v", dir, err))
		s.mu.Unlock()
		return
	}

	_, hasStrayRootPkg := inspectWorkspaceContainer(dir)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		subPath := filepath.Join(dir, name)
		if isIgnoredDir(name) || isExtraExcluded(name, subPath, excludeSet) {
			continue
		}
		if hasStrayRootPkg && isStrayContainerSubdir(name) && !fsutil.DirExists(filepath.Join(subPath, ".git")) {
			continue
		}
		s.walkRoot(subPath, depth+1, isAppsRoot, cutoff, excludeSet, rootSet, existingStates, seenIDs, visitedDirs, report)
	}
}

func (s *Scanner) inspectDir(
	dir string,
	isAppsRoot bool,
	cutoff time.Time,
	existingStates map[string]model.LocalState,
	seenIDs map[string]struct{},
	report *model.ScanReport,
) bool {
	metaPath := filepath.Join(dir, config.MetaFileName)
	if fsutil.FileExists(metaPath) {
		meta, err := ReadMetaFile(dir)
		if err != nil || strings.TrimSpace(meta.ProjectID) == "" {
			s.mu.Lock()
			report.Errors = append(report.Errors, fmt.Sprintf("invalid meta in %s: %v", dir, err))
			s.mu.Unlock()
		} else {
			detected, _ := detectProjectMarkers(dir, isAppsRoot)
			if !detected && !fsutil.FileExists(filepath.Join(dir, config.ManifestFileName)) {
				s.cleanupBogusMeta(dir)
				return false
			}
			if !cutoff.IsZero() && !isProjectModifiedSince(dir, cutoff) {
				return true
			}
			s.mu.Lock()
			report.Discovered++
			seenIDs[meta.ProjectID] = struct{}{}
			s.mu.Unlock()
			s.handleExistingMeta(dir, meta, isAppsRoot, existingStates, report)
			return true
		}
	}

	detected, info := detectProjectMarkers(dir, isAppsRoot)
	if !detected {
		return false
	}

	if !cutoff.IsZero() && !isProjectModifiedSince(dir, cutoff) {
		return true
	}

	newID := uuid.NewString()
	s.mu.Lock()
	report.Discovered++
	seenIDs[newID] = struct{}{}
	s.mu.Unlock()

	folderName := filepath.Base(dir)
	s.mu.Lock()
	slug := s.uniqueSlug(fsutil.Slugify(folderName), newID)
	s.mu.Unlock()

	hasEnv, hasEnvEx, hasBat := computeDirFlags(dir, info.batFile)
	project := model.Project{
		ID:        newID,
		Slug:      slug,
		Name:      folderName,
		Type:      info.projectType,
		Summary:   ExtractSummary(dir),
		Tags:      inferTags(info.techStack, info.projectType),
		TechStack: info.techStack,
		GitRemote: parseGitRemote(dir),
		BatFile:   info.batFile,
		ExeName:   info.exeName,
		UpdatedAt: time.Now().Unix(),
	}

	if err := WriteMetaFile(dir, model.ProjectMeta{
		ProjectID: project.ID,
		Slug:      project.Slug,
		Type:      project.Type,
	}); err != nil {
		s.mu.Lock()
		report.Errors = append(report.Errors, fmt.Sprintf("write meta %s: %v", dir, err))
		s.mu.Unlock()
	}

	s.mu.Lock()
	err := s.repo.UpsertProject(project, &model.LocalState{
		ProjectID:      project.ID,
		LocalDir:       dir,
		Status:         ComputeLocalStatus(dir, project.Type),
		HasEnv:         hasEnv,
		HasEnvExample:  hasEnvEx,
		HasBat:         hasBat,
		FlagsCheckedAt: time.Now().Unix(),
	})
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("save project %s: %v", dir, err))
		s.mu.Unlock()
		return true
	}
	s.syncManifestAndGitLocked(dir, project)
	report.Added++
	s.mu.Unlock()
	return true
}

func (s *Scanner) handleExistingMeta(dir string, meta model.ProjectMeta, isAppsRoot bool, existingStates map[string]model.LocalState, report *model.ScanReport) {
	s.mu.Lock()
	existing, found, err := s.repo.GetProjectModel(meta.ProjectID)
	s.mu.Unlock()
	if err != nil {
		s.mu.Lock()
		report.Errors = append(report.Errors, fmt.Sprintf("lookup project %s: %v", meta.ProjectID, err))
		s.mu.Unlock()
		return
	}

	if found {
		s.relinkExistingProject(dir, existing, isAppsRoot, existingStates, report)
		return
	}

	// Never resurrect a project that the user explicitly deleted in T-Deck.
	if delChecker, ok := s.repo.(interface {
		IsProjectDeleted(id string) (bool, error)
	}); ok {
		s.mu.Lock()
		isDel, delErr := delChecker.IsProjectDeleted(meta.ProjectID)
		s.mu.Unlock()
		if delErr == nil && isDel {
			return
		}
	}

	_, markerInfo := detectProjectMarkers(dir, isAppsRoot)
	pType := meta.Type
	if pType != config.ProjectTypeWeb && pType != config.ProjectTypeDesktop && pType != config.ProjectTypeHybrid {
		pType = markerInfo.projectType
		if pType == "" {
			pType = config.ProjectTypeWeb
		}
	}

	folderName := filepath.Base(dir)
	slugSource := strings.TrimSpace(meta.Slug)
	if slugSource == "" {
		slugSource = folderName
	}
	batFile := markerInfo.batFile
	if batFile == "" {
		batFile = config.DefaultBatFile
	}

	hasEnv, hasEnvEx, hasBat := computeDirFlags(dir, batFile)
	s.mu.Lock()
	slug := s.uniqueSlug(fsutil.Slugify(slugSource), meta.ProjectID)
	s.mu.Unlock()

	project := model.Project{
		ID:        meta.ProjectID,
		Slug:      slug,
		Name:      folderName,
		Type:      pType,
		Summary:   ExtractSummary(dir),
		Tags:      inferTags(markerInfo.techStack, pType),
		TechStack: markerInfo.techStack,
		GitRemote: parseGitRemote(dir),
		BatFile:   batFile,
		ExeName:   markerInfo.exeName,
		UpdatedAt: time.Now().Unix(),
	}

	s.mu.Lock()
	err = s.repo.UpsertProject(project, &model.LocalState{
		ProjectID:      project.ID,
		LocalDir:       dir,
		Status:         ComputeLocalStatus(dir, pType),
		HasEnv:         hasEnv,
		HasEnvExample:  hasEnvEx,
		HasBat:         hasBat,
		FlagsCheckedAt: time.Now().Unix(),
	})
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("import meta %s: %v", dir, err))
		s.mu.Unlock()
		return
	}
	s.syncManifestAndGitLocked(dir, project)
	report.Added++
	s.mu.Unlock()
}

func (s *Scanner) relinkExistingProject(dir string, existing model.Project, isAppsRoot bool, existingStates map[string]model.LocalState, report *model.ScanReport) {
	prevState, hasPrev := existingStates[existing.ID]
	hasEnv, hasEnvEx, hasBat := computeDirFlags(dir, existing.BatFile)

	s.mu.Lock()
	err := s.repo.UpsertLocalState(model.LocalState{
		ProjectID:      existing.ID,
		LocalDir:       dir,
		Status:         ComputeLocalStatus(dir, existing.Type),
		HasEnv:         hasEnv,
		HasEnvExample:  hasEnvEx,
		HasBat:         hasBat,
		FlagsCheckedAt: time.Now().Unix(),
	})
	s.mu.Unlock()
	if err != nil {
		s.mu.Lock()
		report.Errors = append(report.Errors, fmt.Sprintf("relink %s: %v", dir, err))
		s.mu.Unlock()
		return
	}

	needExeDetect := existing.Type != config.ProjectTypeWeb && (existing.ExeName == "" || !fsutil.FileExists(filepath.Join(dir, existing.ExeName)))
	needMetaDetect := needExeDetect || existing.BatFile == "" || existing.TechStack == ""
	needsUpdate := false
	if needMetaDetect {
		_, markerInfo := detectProjectMarkers(dir, isAppsRoot)
		if existing.ExeName == "" && markerInfo.exeName != "" {
			existing.ExeName = markerInfo.exeName
			needsUpdate = true
		}
		if existing.BatFile == "" && markerInfo.batFile != "" {
			existing.BatFile = markerInfo.batFile
			needsUpdate = true
		}
		if existing.TechStack == "" && markerInfo.techStack != "" {
			existing.TechStack = markerInfo.techStack
			needsUpdate = true
		}
	}
	if existing.Summary == "" {
		if sum := ExtractSummary(dir); sum != "" {
			existing.Summary = sum
			needsUpdate = true
		}
	}
	if needsUpdate {
		existing.UpdatedAt = time.Now().Unix()
		s.mu.Lock()
		_ = s.repo.UpsertProject(existing, nil)
		s.mu.Unlock()
	}

	s.mu.Lock()
	s.syncManifestAndGitLocked(dir, existing)
	if !hasPrev || !strings.EqualFold(filepath.Clean(prevState.LocalDir), filepath.Clean(dir)) || prevState.Status == config.StatusCloudOnly || prevState.Status == config.StatusMissingDir {
		report.Relinked++
	} else {
		report.Updated++
	}
	s.mu.Unlock()
}

// syncManifestAndGitLocked updates local Git metadata and syncs .tdeck.json targets when present.
// Caller must hold s.mu.
func (s *Scanner) syncManifestAndGitLocked(dir string, proj model.Project) {
	if gitUpdater, ok := s.repo.(interface {
		UpdateGitState(projectID, branch, commit string, dirty bool) error
	}); ok {
		if br, sha, dirty := ReadGitInfo(dir); br != "" || sha != "" {
			_ = gitUpdater.UpdateGitState(proj.ID, br, sha, dirty)
		}
	}

	manifest, found, err := LoadProjectManifest(dir)
	if err != nil || !found || manifest == nil {
		return
	}
	_ = ApplyProjectManifestToStore(s.repo, proj.ID, manifest)
}

// ApplyProjectManifestToStore syncs categories, tags, and targets from a parsed .tdeck.json manifest into store.
func ApplyProjectManifestToStore(store any, projectID string, manifest *ProjectManifest) error {
	if manifest == nil {
		return nil
	}
	if tRepo, ok := store.(interface {
		ListProjectTargets(projectID string) ([]model.ProjectTarget, error)
		UpsertTarget(t model.ProjectTarget) (model.ProjectTarget, error)
	}); ok {
		existing, _ := tRepo.ListProjectTargets(projectID)
		byKey := make(map[string]model.ProjectTarget, len(existing))
		for _, t := range existing {
			byKey[strings.ToLower(t.Kind+"|"+t.Name)] = t
		}
		for i, mt := range manifest.Targets {
			kind := strings.ToLower(strings.TrimSpace(mt.Kind))
			if kind != config.ProjectTypeWeb && kind != config.ProjectTypeDesktop {
				continue
			}
			name := strings.TrimSpace(mt.Name)
			if name == "" {
				if kind == config.ProjectTypeWeb {
					name = "Web App"
				} else {
					name = "Desktop App"
				}
			}
			prev := byKey[strings.ToLower(kind+"|"+name)]
			tgt := model.ProjectTarget{
				ID:              prev.ID,
				ProjectID:       projectID,
				Kind:            kind,
				Name:            name,
				Platform:        mt.Platform,
				Distribution:    mt.Distribution,
				Subpath:         mt.Subpath,
				BuildCmd:        mt.BuildCmd,
				ArtifactGlob:    mt.ArtifactGlob,
				LaunchEntry:     mt.LaunchEntry,
				ProdURL:         mt.ProdURL,
				StagingURL:      mt.StagingURL,
				HealthURL:       mt.HealthURL,
				VersionURL:      mt.VersionURL,
				HostingProvider: mt.HostingProvider,
				SortOrder:       i,
				UpdatedAt:       time.Now().Unix(),
			}
			if _, err := tRepo.UpsertTarget(tgt); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Scanner) markMissingProjects(
	states map[string]model.LocalState,
	seenIDs map[string]struct{},
	appsRootNorm string,
	excludeSet map[string]struct{},
	report *model.ScanReport,
) {
	for pid, st := range states {
		if _, ok := seenIDs[pid]; ok {
			continue
		}
		if st.Status == config.StatusCloudOnly || st.LocalDir == "" {
			continue
		}
		if !fsutil.DirExists(st.LocalDir) {
			if err := s.repo.UpdateLocalStatus(pid, config.StatusMissingDir); err == nil {
				report.Missing++
			}
			continue
		}
		if isFalsePositiveLocalDir(st.LocalDir, appsRootNorm, excludeSet) {
			_ = os.Remove(filepath.Join(st.LocalDir, config.MetaFileName))
			s.purgeFalsePositive(pid)
		}
	}
}

func (s *Scanner) cleanupBogusMeta(dir string) {
	metaPath := filepath.Join(dir, config.MetaFileName)
	if !fsutil.FileExists(metaPath) {
		return
	}
	if meta, err := ReadMetaFile(dir); err == nil && strings.TrimSpace(meta.ProjectID) != "" {
		s.purgeFalsePositive(meta.ProjectID)
	}
	_ = os.Remove(metaPath)
}

func (s *Scanner) purgeFalsePositive(projectID string) {
	if purger, ok := s.repo.(interface {
		PurgeFalsePositiveProject(id string) error
	}); ok {
		s.mu.Lock()
		_ = purger.PurgeFalsePositiveProject(projectID)
		s.mu.Unlock()
	}
}

func inspectWorkspaceContainer(dir string) (isContainer bool, hasStrayPkg bool) {
	if fsutil.DirExists(filepath.Join(dir, ".git")) {
		return false, false
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, false
	}
	rootPkgName := ""
	if pkg := readRootPackageJSON(dir); pkg != nil {
		if n, ok := pkg["name"].(string); ok {
			rootPkgName = strings.TrimSpace(n)
		}
	}
	gitChildren := 0
	hasDuplicateChild := false
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if isIgnoredDir(name) || isNestedStackSubdir(name) {
			continue
		}
		subDir := filepath.Join(dir, name)
		hasGit := fsutil.DirExists(filepath.Join(subDir, ".git"))
		if hasGit {
			gitChildren++
		}
		if rootPkgName != "" && strings.EqualFold(name, rootPkgName) &&
			(hasGit || fsutil.FileExists(filepath.Join(subDir, "package.json"))) {
			hasDuplicateChild = true
		}
	}
	return gitChildren >= 4 || hasDuplicateChild, hasDuplicateChild && rootPkgName != ""
}

func isStrayContainerSubdir(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "backend", "src", "public", "dist", "data", "scripts", "node_modules":
		return true
	}
	return false
}

func isGenericSubdirProjectName(dir string) bool {
	if fsutil.DirExists(filepath.Join(dir, ".git")) || fsutil.FileExists(filepath.Join(dir, config.ManifestFileName)) {
		return false
	}
	switch strings.ToLower(filepath.Base(dir)) {
	case "src", "lib", "docs", "assets", "templates", "scripts", "test", "tests",
		"examples", "samples", "fixtures", "companion-extension", "nested-extension":
		return true
	}
	return false
}

func isFalsePositiveLocalDir(dir string, appsRootNorm string, excludeSet map[string]struct{}) bool {
	cleaned := filepath.Clean(dir)
	norm := strings.ToLower(cleaned)
	isUnderApps := appsRootNorm != "" && strings.HasPrefix(norm, appsRootNorm+string(filepath.Separator))

	// Check every path segment against IgnoredDirs / extra exclusions.
	relCheck := cleaned
	for len(relCheck) > len(filepath.VolumeName(relCheck))+1 {
		base := filepath.Base(relCheck)
		if isIgnoredDir(base) || isExtraExcluded(base, relCheck, excludeSet) {
			return true
		}
		parent := filepath.Dir(relCheck)
		if parent == relCheck {
			break
		}
		relCheck = parent
	}

	if !isUnderApps {
		if isInstalledAppOrSDKDir(cleaned) || isGenericSubdirProjectName(cleaned) {
			return true
		}
		if isContainer, _ := inspectWorkspaceContainer(cleaned); isContainer {
			return true
		}
		parent := filepath.Dir(cleaned)
		if parent != cleaned {
			if _, hasStrayPkg := inspectWorkspaceContainer(parent); hasStrayPkg &&
				isStrayContainerSubdir(filepath.Base(cleaned)) &&
				!fsutil.DirExists(filepath.Join(cleaned, ".git")) {
				return true
			}
		}
	}

	detected, _ := detectProjectMarkers(cleaned, isUnderApps)
	return !detected && !fsutil.FileExists(filepath.Join(cleaned, config.ManifestFileName))
}

func (s *Scanner) uniqueSlug(base string, excludeID string) string {
	if base == "" {
		base = "project"
	}
	candidate := base
	for i := 2; i < 1000; i++ {
		exists, err := s.repo.SlugExists(candidate, excludeID)
		if err != nil || !exists {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
	return fmt.Sprintf("%s-%s", base, excludeID[:8])
}

func deduplicateRoots(workspaceDirs []string, appsDir string) []string {
	roots := make([]string, 0, len(workspaceDirs)+1)
	seen := make(map[string]struct{})
	add := func(dir string) {
		trimmed := strings.TrimSpace(dir)
		if trimmed == "" {
			return
		}
		cleaned := filepath.Clean(trimmed)
		key := strings.ToLower(cleaned)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		roots = append(roots, cleaned)
	}
	for _, d := range workspaceDirs {
		add(d)
	}
	add(appsDir)
	return roots
}

func buildExcludeSet(dirs []string) map[string]struct{} {
	set := make(map[string]struct{}, len(dirs))
	for _, d := range dirs {
		if clean := strings.ToLower(strings.TrimSpace(d)); clean != "" {
			set[clean] = struct{}{}
		}
	}
	return set
}

func isProjectModifiedSince(root string, cutoff time.Time) bool {
	// Fast-path: check common top-level anchor files (not root directory itself, which updates on dir creation)
	// before walking the directory tree.
	fastPaths := []string{
		filepath.Join(root, ".git", "index"),
		filepath.Join(root, ".git", "HEAD"),
		filepath.Join(root, "package.json"),
		filepath.Join(root, "go.mod"),
		filepath.Join(root, config.WailsConfig),
		filepath.Join(root, config.ReadmeFile),
		filepath.Join(root, config.DefaultBatFile),
	}
	for _, p := range fastPaths {
		if info, err := os.Stat(p); err == nil && !info.ModTime().Before(cutoff) {
			return true
		}
	}

	modified := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != root && isIgnoredDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(d.Name(), config.MetaFileName) {
			return nil
		}
		if info, err := d.Info(); err == nil && !info.ModTime().Before(cutoff) {
			modified = true
			return filepath.SkipAll
		}
		return nil
	})
	return modified
}

func computeDirFlags(dir, batFile string) (hasEnv, hasEnvEx, hasBat bool) {
	hasEnv = fsutil.FileExists(filepath.Join(dir, config.EnvFile))
	hasEnvEx = fsutil.FileExists(filepath.Join(dir, config.EnvExampleFile))
	batName := strings.TrimSpace(batFile)
	if batName == "" {
		batName = config.DefaultBatFile
	}
	hasBat = fsutil.FileExists(filepath.Join(dir, batName))
	return hasEnv, hasEnvEx, hasBat
}

func isIgnoredDir(name string) bool {
	lower := strings.ToLower(name)
	for _, ignored := range config.IgnoredDirs {
		if lower == strings.ToLower(ignored) {
			return true
		}
	}
	return strings.HasPrefix(name, ".")
}

func isExtraExcluded(name, fullPath string, excludeSet map[string]struct{}) bool {
	if len(excludeSet) == 0 {
		return false
	}
	if _, ok := excludeSet[strings.ToLower(name)]; ok {
		return true
	}
	_, ok := excludeSet[strings.ToLower(filepath.Clean(fullPath))]
	return ok
}
