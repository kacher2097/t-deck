package database

import (
	"path/filepath"
	"os"
	"testing"

	"t-deck/internal/config"
	"t-deck/internal/model"
)

func TestProjectRepoCRUDAndFTS5Search(t *testing.T) {
	tmp := t.TempDir()
	db, err := Open(filepath.Join(tmp, config.DatabaseFileName))
	if err != nil {
		t.Fatalf("Open sqlite failed: %v", err)
	}
	defer db.Close()

	repo := NewProjectRepo(db, filepath.Join(tmp, config.ThumbnailsDirName))

	p1 := model.Project{
		ID:        "11111111-1111-4111-8111-111111111111",
		Slug:      "audio-translator",
		Name:      "Audio Translator",
		Type:      config.ProjectTypeDesktop,
		Summary:   "Công cụ dịch video và tách phụ đề tự động bằng AI",
		Tags:      []string{"video", "ai", "go"},
		TechStack: "Go, Wails, React",
		ExeName:   "AudioTranslator.exe",
		UpdatedAt: 1000,
	}
	p2 := model.Project{
		ID:        "22222222-2222-4222-8222-222222222222",
		Slug:      "invoice-web",
		Name:      "Invoice Portal",
		Type:      config.ProjectTypeWeb,
		Summary:   "Web quản lý hóa đơn bán lẻ",
		Tags:      []string{"finance", "react"},
		TechStack: "React, Vite",
		BatFile:   config.DefaultBatFile,
		UpdatedAt: 1001,
	}

	if err := repo.UpsertProject(p1, &model.LocalState{
		ProjectID: p1.ID,
		LocalDir:  filepath.Join(tmp, "audio"),
		Status:    config.StatusReady,
	}); err != nil {
		t.Fatalf("Upsert p1 failed: %v", err)
	}
	if err := repo.UpsertProject(p2, &model.LocalState{
		ProjectID: p2.ID,
		LocalDir:  "",
		Status:    config.StatusCloudOnly,
	}); err != nil {
		t.Fatalf("Upsert p2 failed: %v", err)
	}

	// Vietnamese FTS5 search.
	res, err := repo.ListProjects(model.ProjectFilter{Query: "dịch video"})
	if err != nil {
		t.Fatalf("FTS5 search failed: %v", err)
	}
	if len(res) != 1 || res[0].ID != p1.ID {
		t.Fatalf("expected p1 from Vietnamese search, got %+v", res)
	}

	// Tag filter.
	resTag, err := repo.ListProjects(model.ProjectFilter{Tag: "#finance"})
	if err != nil {
		t.Fatalf("Tag filter failed: %v", err)
	}
	if len(resTag) != 1 || resTag[0].ID != p2.ID {
		t.Fatalf("expected p2 from tag filter, got %+v", resTag)
	}

	// Status filter.
	resStatus, err := repo.ListProjects(model.ProjectFilter{Status: config.StatusCloudOnly})
	if err != nil {
		t.Fatalf("Status filter failed: %v", err)
	}
	if len(resStatus) != 1 || resStatus[0].ID != p2.ID {
		t.Fatalf("expected p2 from status filter, got %+v", resStatus)
	}

	// Delete removes from FTS as well.
	if err := repo.DeleteProject(p1.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	afterDel, err := repo.ListProjects(model.ProjectFilter{Query: "dịch video"})
	if err != nil {
		t.Fatalf("Search after delete failed: %v", err)
	}
	if len(afterDel) != 0 {
		t.Fatalf("expected 0 results after delete, got %d", len(afterDel))
	}
}


func TestCategoriesAndFilter(t *testing.T) {
	tmp := t.TempDir()
	db, err := Open(filepath.Join(tmp, config.DatabaseFileName))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewProjectRepo(db, filepath.Join(tmp, config.ThumbnailsDirName))
	cat, err := repo.CreateCategory("Video & AI")
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}

	p := model.Project{
		ID:        "cat-proj-1",
		Slug:      "video-studio",
		Name:      "Video Studio",
		Type:      config.ProjectTypeDesktop,
		Category:  cat.Name,
		UpdatedAt: 100,
	}
	if err := repo.UpsertProject(p, &model.LocalState{
		ProjectID: p.ID,
		LocalDir:  tmp,
		Status:    config.StatusReady,
	}); err != nil {
		t.Fatal(err)
	}

	found, err := repo.ListProjects(model.ProjectFilter{Category: "Video & AI"})
	if err != nil || len(found) != 1 {
		t.Fatalf("expected 1 project in category, got %d (err=%v)", len(found), err)
	}

	if _, err := repo.RenameCategory(cat.ID, "AI Media"); err != nil {
		t.Fatalf("RenameCategory failed: %v", err)
	}
	renamedList, err := repo.ListProjects(model.ProjectFilter{Category: "AI Media"})
	if err != nil || len(renamedList) != 1 {
		t.Fatalf("expected project category updated on rename, got %d (err=%v)", len(renamedList), err)
	}

	if err := repo.DeleteCategory(cat.ID); err != nil {
		t.Fatalf("DeleteCategory failed: %v", err)
	}
	uncat, err := repo.ListProjects(model.ProjectFilter{Category: "__uncategorized__"})
	if err != nil || len(uncat) != 1 {
		t.Fatalf("expected project uncategorized after delete, got %d (err=%v)", len(uncat), err)
	}

	// Reviving soft-deleted category by recreating same name succeeds cleanly.
	revived, err := repo.CreateCategory("AI Media")
	if err != nil {
		t.Fatalf("revive category failed: %v", err)
	}
	if revived.ID != cat.ID || revived.DeletedAt != 0 {
		t.Fatalf("expected revived category %s with deleted_at=0, got %+v", cat.ID, revived)
	}
}

func TestV2MigrationCachedFlagsAndSoftDelete(t *testing.T) {
	tmp := t.TempDir()
	db, err := Open(filepath.Join(tmp, config.DatabaseFileName))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var ver int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&ver); err != nil {
		t.Fatal(err)
	}
	if ver < 2 {
		t.Fatalf("expected PRAGMA user_version >= 2, got %d", ver)
	}

	repo := NewProjectRepo(db, filepath.Join(tmp, config.ThumbnailsDirName))
	projDir := filepath.Join(tmp, "proj-a")
	p := model.Project{
		ID:        "proj-a-id",
		Slug:      "proj-a",
		Name:      "Proj A",
		Type:      config.ProjectTypeWeb,
		UpdatedAt: 100,
	}
	if err := repo.UpsertProject(p, &model.LocalState{
		ProjectID: p.ID,
		LocalDir:  projDir,
		Status:    config.StatusReady,
	}); err != nil {
		t.Fatal(err)
	}

	if err := repo.DeleteProject(p.ID); err != nil {
		t.Fatal(err)
	}
	active, err := repo.ListAllProjects()
	if err != nil || len(active) != 0 {
		t.Fatalf("expected 0 active projects after soft-delete, got %d (err=%v)", len(active), err)
	}
	withDel, err := repo.ListAllProjectsIncludingDeleted()
	if err != nil || len(withDel) != 1 || withDel[0].DeletedAt == 0 {
		t.Fatalf("expected 1 tombstone project, got %+v (err=%v)", withDel, err)
	}
	exists, err := repo.SlugExists("proj-a", "other-id")
	if err != nil || exists {
		t.Fatalf("expected soft-deleted slug not to block SlugExists, exists=%v err=%v", exists, err)
	}
}

func TestV3MigrationTargetsAndReleases(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, config.DatabaseFileName)
	db, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var ver int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&ver); err != nil {
		t.Fatal(err)
	}
	if ver != 3 {
		t.Fatalf("expected PRAGMA user_version == 3, got %d", ver)
	}

	repo := NewProjectRepo(db, filepath.Join(tmp, config.ThumbnailsDirName))
	p := model.Project{
		ID:        "hybrid-1",
		Slug:      "hybrid-app",
		Name:      "Hybrid App",
		Type:      config.ProjectTypeWeb,
		UpdatedAt: 100,
	}
	if err := repo.UpsertProject(p, &model.LocalState{
		ProjectID: p.ID,
		LocalDir:  tmp,
		Status:    config.StatusReady,
	}); err != nil {
		t.Fatal(err)
	}

	webTgt, err := repo.UpsertTarget(model.ProjectTarget{
		ProjectID: p.ID,
		Kind:      config.ProjectTypeWeb,
		Name:      "Portal",
		ProdURL:   "https://portal.example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	deskTgt, err := repo.UpsertTarget(model.ProjectTarget{
		ProjectID:   p.ID,
		Kind:        config.ProjectTypeDesktop,
		Name:        "Client",
		LaunchEntry: "Client.exe",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Having both web and desktop targets automatically promotes parent project to "hybrid".
	view, err := repo.GetProject(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if view.Type != config.ProjectTypeHybrid || len(view.Targets) != 2 || view.LiveURL != "https://portal.example.com" {
		t.Fatalf("expected hybrid project with 2 targets and LiveURL, got %+v", view)
	}

	if _, err := repo.UpsertRelease(model.Release{
		ProjectID:   p.ID,
		TargetID:    deskTgt.ID,
		Version:     "v1.5.0",
		Channel:     config.ReleaseChannelStable,
		R2Key:       "releases/hybrid-app/desktop/v1.5.0/app.zip",
		PublishedAt: 500,
	}); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateInstalledRelease(p.ID, "v1.4.0", ""); err != nil {
		t.Fatal(err)
	}

	viewAfter, err := repo.GetProject(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !viewAfter.HasUpdate || viewAfter.InstalledVersion != "v1.4.0" {
		t.Fatalf("expected HasUpdate=true and InstalledVersion=v1.4.0, got %+v", viewAfter)
	}
	_ = webTgt
}

func TestRealAppDataDBCopyMigrationOptIn(t *testing.T) {
	realCopy := os.Getenv("TDECK_VERIFY_REAL_DB_COPY")
	if realCopy == "" {
		t.Skip("set TDECK_VERIFY_REAL_DB_COPY to run copied AppData migration test")
	}
	db, err := Open(realCopy)
	if err != nil {
		t.Fatalf("Open copied real DB failed: %v", err)
	}
	defer db.Close()

	var userVer int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&userVer); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(realCopy + ".v2.bak"); err != nil {
		t.Fatalf("expected .v2.bak backup file created: %v", err)
	}
	repo := NewProjectRepo(db, filepath.Join(filepath.Dir(realCopy), "thumbnails"))
	projects, err := repo.ListProjects(model.ProjectFilter{})
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}
	cats, err := repo.ListCategories()
	if err != nil {
		t.Fatalf("ListCategories failed: %v", err)
	}
	var targetCount, relCount int
	_ = db.QueryRow(`SELECT COUNT(1) FROM project_targets WHERE COALESCE(deleted_at, 0) = 0`).Scan(&targetCount)
	_ = db.QueryRow(`SELECT COUNT(1) FROM releases WHERE COALESCE(deleted_at, 0) = 0`).Scan(&relCount)

	ftsRes, err := repo.ListProjects(model.ProjectFilter{Query: "app"})
	if err != nil {
		t.Fatalf("FTS search after migration failed: %v", err)
	}
	t.Logf("REAL DB MIGRATION VERIFIED: user_version=%d projects=%d categories=%d targets=%d releases=%d fts_app=%d",
		userVer, len(projects), len(cats), targetCount, relCount, len(ftsRes))
}

func TestListProjectsDefaultAlphaOrder(t *testing.T) {
	tmp := t.TempDir()
	db, err := Open(filepath.Join(tmp, config.DatabaseFileName))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewProjectRepo(db, filepath.Join(tmp, config.ThumbnailsDirName))
	items := []model.Project{
		{ID: "p-z", Slug: "zebra", Name: "Zebra Tool", Type: config.ProjectTypeDesktop, UpdatedAt: 300},
		{ID: "p-a", Slug: "alpha", Name: "alpha app", Type: config.ProjectTypeWeb, UpdatedAt: 100},
		{ID: "p-m", Slug: "mid", Name: "Mid Studio", Type: config.ProjectTypeHybrid, UpdatedAt: 200},
	}
	for _, p := range items {
		if err := repo.UpsertProject(p, &model.LocalState{
			ProjectID: p.ID,
			LocalDir:  tmp,
			Status:    config.StatusReady,
		}); err != nil {
			t.Fatal(err)
		}
	}

	// Launching "Zebra Tool" updates last_opened_at, but default listing must stay A-Z.
	if err := repo.TouchLastOpened("p-z"); err != nil {
		t.Fatal(err)
	}

	list, err := repo.ListProjects(model.ProjectFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 projects, got %d", len(list))
	}
	want := []string{"p-a", "p-m", "p-z"}
	for i, id := range want {
		if list[i].ID != id {
			t.Fatalf("position %d: expected %s, got %s (full order: %s, %s, %s)",
				i, id, list[i].ID, list[0].Name, list[1].Name, list[2].Name)
		}
	}
}

func TestBulkDeleteProjectsAndTombstoneCheck(t *testing.T) {
	tmp := t.TempDir()
	db, err := Open(filepath.Join(tmp, config.DatabaseFileName))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewProjectRepo(db, filepath.Join(tmp, config.ThumbnailsDirName))
	for _, p := range []model.Project{
		{ID: "b-1", Slug: "b1", Name: "B1", Type: config.ProjectTypeWeb, UpdatedAt: 100},
		{ID: "b-2", Slug: "b2", Name: "B2", Type: config.ProjectTypeDesktop, UpdatedAt: 101},
		{ID: "b-3", Slug: "b3", Name: "B3", Type: config.ProjectTypeHybrid, UpdatedAt: 102},
	} {
		if err := repo.UpsertProject(p, &model.LocalState{
			ProjectID: p.ID,
			LocalDir:  tmp,
			Status:    config.StatusReady,
		}); err != nil {
			t.Fatal(err)
		}
	}

	deleted, err := repo.DeleteProjects([]string{"b-1", "b-2", "b-1"})
	if err != nil {
		t.Fatalf("DeleteProjects failed: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("expected 2 deleted, got %d", deleted)
	}

	isDel, err := repo.IsProjectDeleted("b-1")
	if err != nil || !isDel {
		t.Fatalf("expected b-1 to be deleted tombstone, got isDel=%v err=%v", isDel, err)
	}
	isDel3, err := repo.IsProjectDeleted("b-3")
	if err != nil || isDel3 {
		t.Fatalf("expected b-3 to remain active, got isDel=%v err=%v", isDel3, err)
	}

	remaining, err := repo.ListProjects(model.ProjectFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 1 || remaining[0].ID != "b-3" {
		t.Fatalf("expected only b-3 remaining, got %+v", remaining)
	}
}

func TestSearchAndTagFiltersResistSQLAndWildcardInjection(t *testing.T) {
	tmp := t.TempDir()
	db, err := Open(filepath.Join(tmp, config.DatabaseFileName))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewProjectRepo(db, filepath.Join(tmp, config.ThumbnailsDirName))
	projects := []model.Project{
		{
			ID:        "sec-1",
			Slug:      "normal-app",
			Name:      "Normal App",
			Type:      config.ProjectTypeWeb,
			Category:  "Web",
			Summary:   "Standard web dashboard",
			Tags:      []string{"node_js", "react"},
			TechStack: "React",
			UpdatedAt: 100,
		},
		{
			ID:        "sec-2",
			Slug:      "percent-app",
			Name:      "100% Pure Tool",
			Type:      config.ProjectTypeDesktop,
			Category:  "Tools",
			Summary:   "Special chars % and _",
			Tags:      []string{"100%", "c++"},
			TechStack: "Go",
			UpdatedAt: 101,
		},
	}
	for _, p := range projects {
		if err := repo.UpsertProject(p, &model.LocalState{
			ProjectID: p.ID,
			LocalDir:  tmp,
			Status:    config.StatusReady,
		}); err != nil {
			t.Fatal(err)
		}
	}

	// 1. Bare '%' must NOT act as wildcard matching all rows (only matches literal '%' in sec-2).
	resPct, err := repo.ListProjects(model.ProjectFilter{Query: "%"})
	if err != nil {
		t.Fatalf("query %% failed: %v", err)
	}
	if len(resPct) != 1 || resPct[0].ID != "sec-2" {
		t.Fatalf("expected literal %% to match only sec-2, got %+v", resPct)
	}

	// 2. Classic SQL injection payloads must return 0 rows and leave DB intact.
	for _, payload := range []string{
		`' OR '1'='1`,
		`"; DROP TABLE projects; --`,
		`%' OR 1=1 --`,
		`") OR 1=1 --`,
	} {
		resQ, err := repo.ListProjects(model.ProjectFilter{Query: payload})
		if err != nil {
			t.Fatalf("query payload %q errored: %v", payload, err)
		}
		if len(resQ) != 0 {
			t.Fatalf("query payload %q leaked rows: %+v", payload, resQ)
		}
		resT, err := repo.ListProjects(model.ProjectFilter{Tag: payload})
		if err != nil {
			t.Fatalf("tag payload %q errored: %v", payload, err)
		}
		if len(resT) != 0 {
			t.Fatalf("tag payload %q leaked rows: %+v", payload, resT)
		}
	}

	// 3. Tag filter with '_' and '%' must match exact tag elements only, not wildcards.
	resUnder, err := repo.ListProjects(model.ProjectFilter{Tag: "node_js"})
	if err != nil || len(resUnder) != 1 || resUnder[0].ID != "sec-1" {
		t.Fatalf("expected exact tag node_js to match sec-1, got %+v (err=%v)", resUnder, err)
	}
	resWildTag, err := repo.ListProjects(model.ProjectFilter{Tag: "%"})
	if err != nil || len(resWildTag) != 0 {
		t.Fatalf("expected tag %% not to act as wildcard, got %+v (err=%v)", resWildTag, err)
	}
	resLiteralPctTag, err := repo.ListProjects(model.ProjectFilter{Tag: "100%"})
	if err != nil || len(resLiteralPctTag) != 1 || resLiteralPctTag[0].ID != "sec-2" {
		t.Fatalf("expected exact tag 100%% to match sec-2, got %+v (err=%v)", resLiteralPctTag, err)
	}

	// 4. tableColumns rejects non-allowlisted table names.
	if _, err := tableColumns(db, "projects; DROP TABLE projects"); err == nil {
		t.Fatalf("expected tableColumns to reject untrusted table name")
	}

	// 5. Verify projects table is still completely intact.
	all, err := repo.ListProjects(model.ProjectFilter{})
	if err != nil || len(all) != 2 {
		t.Fatalf("expected 2 intact projects after injection tests, got %d (err=%v)", len(all), err)
	}
}

func TestSearchSensitivityAndNamePriority(t *testing.T) {
	tmp := t.TempDir()
	db, err := Open(filepath.Join(tmp, config.DatabaseFileName))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewProjectRepo(db, filepath.Join(tmp, config.ThumbnailsDirName))

	// A project whose summary mentions "golang" and "invoice" vs one actually named "Go Tool".
	pSummary := model.Project{
		ID:        "rank-summary",
		Slug:      "alpha-portal",
		Name:      "Alpha Portal",
		Type:      config.ProjectTypeWeb,
		Summary:   "Built with golang and internal invoicing pipeline.",
		UpdatedAt: 100,
	}
	pName := model.Project{
		ID:        "rank-name",
		Slug:      "go-tool",
		Name:      "Go Tool",
		Type:      config.ProjectTypeWeb,
		Summary:   "CLI utility",
		Tags:      []string{"go"},
		TechStack: "Go",
		UpdatedAt: 101,
	}
	for _, p := range []model.Project{pSummary, pName} {
		if err := repo.UpsertProject(p, &model.LocalState{
			ProjectID: p.ID,
			LocalDir:  tmp,
			Status:    config.StatusReady,
		}); err != nil {
			t.Fatal(err)
		}
	}

	// Searching "go" (2 chars) must NOT prefix-match "golang" inside summary.
	resGo, err := repo.ListProjects(model.ProjectFilter{Query: "go"})
	if err != nil {
		t.Fatal(err)
	}
	if len(resGo) != 1 || resGo[0].ID != "rank-name" {
		t.Fatalf("expected exact 2-char token 'go' to match only rank-name, got %+v", resGo)
	}

	// Single-char FTS query returns empty so it falls back to substring search without wildcard explosion.
	if q := buildFTSQuery("t"); q != "" {
		t.Fatalf("expected empty FTS query for 1-char input, got %q", q)
	}
	if q := buildFTSQuery("t-deck"); q != `"deck"*` {
		t.Fatalf("expected 1-char prefix dropped in 't-deck', got %q", q)
	}
	if q := buildFTSQuery("go"); q != `"go"` {
		t.Fatalf("expected exact token match for 2-char input 'go', got %q", q)
	}
}
