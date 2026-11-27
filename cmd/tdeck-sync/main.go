package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"t-deck/internal/cloud"
	"t-deck/internal/config"
	"t-deck/internal/database"
	"t-deck/internal/fsutil"
	"t-deck/internal/launcher"
	"t-deck/internal/media"
	"t-deck/internal/model"
	"t-deck/internal/scanner"
)

func inferDefaultCategory(p model.Project, localDir string) string {
	if strings.TrimSpace(p.Category) != "" {
		return p.Category
	}
	lower := strings.ToLower(p.Slug + " " + p.Name + " " + p.Summary + " " + localDir)
	switch {
	case strings.Contains(lower, "t-hub") || strings.Contains(lower, "noxus") || strings.Contains(lower, "desktop-station"):
		return "T-Hub Ecosystem"
	case strings.Contains(lower, "video") ||
		strings.Contains(lower, "mvdirector") ||
		strings.Contains(lower, "ai-studio") ||
		strings.Contains(lower, "media-vault") ||
		strings.Contains(lower, "vieneu-tts") ||
		strings.Contains(lower, "omnivoice") ||
		strings.Contains(lower, "storyassetgen") ||
		strings.Contains(lower, "douyin") ||
		strings.Contains(lower, "tiktok") ||
		strings.Contains(lower, "trans-vid") ||
		strings.Contains(lower, "trans_video") ||
		strings.Contains(lower, "app-transalate") ||
		strings.Contains(lower, "audiotranslator") ||
		strings.Contains(lower, "icast") ||
		strings.Contains(lower, "movie-web"):
		return "Video & Media AI"
	case strings.Contains(lower, "git vnpay") ||
		strings.Contains(lower, "git thht") ||
		strings.Contains(lower, "git online") ||
		strings.Contains(lower, "vnpay") ||
		strings.Contains(lower, "mcv-") ||
		strings.Contains(lower, "mms-") ||
		strings.Contains(lower, "vmms-") ||
		strings.Contains(lower, "finops-") ||
		strings.Contains(lower, "spos") ||
		strings.Contains(lower, "medlatec") ||
		strings.Contains(lower, "armeria") ||
		strings.Contains(lower, "osb-") ||
		strings.Contains(lower, "reconcile-sftp") ||
		strings.Contains(lower, "spring-vault") ||
		strings.Contains(lower, "vault-common") ||
		strings.Contains(lower, "core-common") ||
		strings.Contains(lower, "core-parent") ||
		strings.Contains(lower, "merchantview") ||
		strings.Contains(lower, "merchant-service") ||
		strings.Contains(lower, "mo-be-api") ||
		strings.Contains(lower, "mo-teko") ||
		strings.Contains(lower, "omni-syn") ||
		strings.Contains(lower, "payment-api") ||
		strings.Contains(lower, "rabbitmq") ||
		strings.Contains(lower, "business-api") ||
		strings.Contains(lower, "business-system") ||
		strings.Contains(lower, "database-common") ||
		strings.Contains(lower, "http-common") ||
		strings.Contains(lower, "netty-common") ||
		strings.Contains(lower, "redis-common"):
		return "Java & Enterprise (VNPAY)"
	case strings.Contains(lower, "s3-desk") ||
		strings.Contains(lower, "kho-repo") ||
		strings.Contains(lower, "botvn") ||
		strings.Contains(lower, "muse2api") ||
		strings.Contains(lower, "crawl") ||
		strings.Contains(lower, "telegram-bot") ||
		strings.Contains(lower, "publish-post-wp") ||
		strings.Contains(lower, "tool_r2") ||
		strings.Contains(lower, "r2-converter") ||
		strings.Contains(lower, "tool_send") ||
		strings.Contains(lower, "my-tool") ||
		strings.Contains(lower, "html2image") ||
		strings.Contains(lower, "convert-html") ||
		strings.Contains(lower, "selenium") ||
		strings.Contains(lower, "test-excel") ||
		strings.Contains(lower, "upload-demo"):
		return "DevTools & Automation"
	default:
		return "Business & Web Apps"
	}
}

func main() {
	dryRun := flag.Bool("dry-run", false, "Scan and report without uploading to R2")
	slugFilter := flag.String("slug", "", "Filter to a single project slug")
	targetFilter := flag.String("target", "all", "Target kind to publish: all | web | desktop")
	channel := flag.String("channel", config.ReleaseChannelStable, "Release channel: stable | beta | canary")
	publishDesktop := flag.Bool("publish-desktop", true, "Package and publish desktop targets to R2")
	publishWeb := flag.Bool("publish-web", false, "Package and publish web static targets to R2")
	forcePublish := flag.Bool("force-publish", false, "Re-publish release even if version matches")
	runBuild := flag.Bool("build", false, "Execute target build_cmd before publishing")
	genThumbs := flag.Bool("gen-thumbs", true, "Generate 1280x720 thumbnails for projects lacking one")
	forceThumbs := flag.Bool("force-thumbs", false, "Regenerate thumbnails even if one already exists")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	mgr, err := config.NewSettingsManager("")
	if err != nil {
		log.Fatalf("init settings manager: %v", err)
	}
	settings, err := mgr.Load()
	if err != nil {
		log.Fatalf("load settings: %v", err)
	}

	db, err := database.Open(mgr.DBPath())
	if err != nil {
		log.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	repo := database.NewProjectRepo(db, mgr.ThumbnailsDir())
	sc := scanner.NewScanner(repo)

	scanRep, err := sc.ScanWithOptions(scanner.ScanOptions{
		WorkspaceDirs:    settings.WorkspaceDirs,
		AppsDir:          settings.AppsDir,
		RecentDays:       settings.ScanRecentDays,
		ExtraExcludeDirs: settings.ScanExcludeDirs,
	})
	if err != nil {
		log.Fatalf("scan workspaces: %v", err)
	}
	fmt.Printf("Scan complete: scanned=%d discovered=%d added=%d relinked=%d updated=%d missing=%d\n",
		scanRep.ScannedDirs, scanRep.Discovered, scanRep.Added, scanRep.Relinked, scanRep.Updated, scanRep.Missing)

	views, err := repo.ListProjects(model.ProjectFilter{})
	if err != nil {
		log.Fatalf("list projects: %v", err)
	}

	var thumbQueue []model.Project
	localDirByID := make(map[string]string, len(views))
	for _, v := range views {
		if *slugFilter != "" && !strings.EqualFold(v.Slug, *slugFilter) {
			continue
		}
		localDirByID[v.ID] = v.LocalDir
		proj, found, err := repo.GetProjectModel(v.ID)
		if err != nil || !found {
			continue
		}
		changed := false

		if strings.TrimSpace(proj.Summary) == "" {
			proj.Summary = media.DefaultSummaryForProject(proj)
			changed = true
		}
		if strings.TrimSpace(proj.Category) == "" {
			proj.Category = inferDefaultCategory(proj, v.LocalDir)
			_, _ = repo.CreateCategory(proj.Category)
			changed = true
		}
		if changed {
			proj.UpdatedAt = time.Now().Unix()
			_ = repo.UpsertProject(proj, nil)
		}

		if *genThumbs && (*forceThumbs || v.ThumbnailURL == "" || (v.IconURL == "" && v.LocalDir != "" && media.HasProjectBrandLogo(v.LocalDir))) {
			thumbQueue = append(thumbQueue, proj)
		}
	}

	if len(thumbQueue) > 0 {
		generateThumbnailsConcurrent(repo, mgr.ThumbnailsDir(), thumbQueue, localDirByID)
	}

	views, err = repo.ListProjects(model.ProjectFilter{})
	if err != nil {
		log.Fatalf("reload projects: %v", err)
	}

	if *dryRun {
		printSummary(views, *slugFilter)
		return
	}

	r2Client, err := cloud.NewR2Client(ctx, settings.R2)
	if err != nil {
		log.Fatalf("init R2 client: %v", err)
	}
	if err := r2Client.TestConnection(ctx); err != nil {
		log.Fatalf("R2 connection check failed: %v", err)
	}
	fmt.Printf("R2 connected: bucket=%s account=%s\n", settings.R2.BucketName, settings.R2.AccountID)

	for _, v := range views {
		if *slugFilter != "" && !strings.EqualFold(v.Slug, *slugFilter) {
			continue
		}
		if v.LocalDir == "" {
			continue
		}
		manifest, _, _ := scanner.LoadProjectManifest(v.LocalDir)
		var vSrc *scanner.ManifestVersionSource
		if manifest != nil {
			vSrc = manifest.VersionSource
		}
		resolvedVersion := scanner.ResolveProjectVersion(v.LocalDir, vSrc)

		for _, tgt := range v.Targets {
			if *targetFilter != "all" && !strings.EqualFold(tgt.Kind, *targetFilter) {
				continue
			}
			if tgt.Kind == config.ProjectTypeDesktop && !*publishDesktop {
				continue
			}
			if tgt.Kind == config.ProjectTypeWeb && !*publishWeb {
				continue
			}

			latestRel, hasLatest := v.LatestReleases[tgt.ID]
			if !*forcePublish && hasLatest && latestRel.Version == resolvedVersion {
				fmt.Printf("Skip publish %s [%s] (%s already published)\n", v.Slug, tgt.Name, resolvedVersion)
				continue
			}

			workDir, subErr := fsutil.ResolveSafeSubpath(v.LocalDir, tgt.Subpath)
			if subErr != nil {
				fmt.Printf("Skip publish %s [%s]: %v\n", v.Slug, tgt.Name, subErr)
				continue
			}
			if *runBuild && strings.TrimSpace(tgt.BuildCmd) != "" {
				fmt.Printf("Building %s [%s]: %s\n", v.Slug, tgt.Name, tgt.BuildCmd)
				if err := launcher.RunBuildCommand(ctx, workDir, tgt.BuildCmd, func(stream, line string) {
					fmt.Printf("  [%s] %s\n", stream, line)
				}); err != nil {
					log.Fatalf("build %s [%s] failed: %v", v.Slug, tgt.Name, err)
				}
			}

			artifactPath, err := resolveTargetArtifact(v.LocalDir, tgt)
			if err != nil {
				fmt.Printf("Skip publish %s [%s]: %v\n", v.Slug, tgt.Name, err)
				continue
			}

			notes, _ := launcher.SuggestChangelog(v.LocalDir)
			assetKind := "portable_zip"
			if tgt.Kind == config.ProjectTypeWeb {
				assetKind = "static_zip"
			} else if tgt.Distribution == "installer" {
				assetKind = "installer_exe"
			}

			fmt.Printf("Publishing %s [%s] (%s) from %s...\n", v.Slug, tgt.Name, resolvedVersion, artifactPath)
			rel, err := cloud.PublishRelease(ctx, r2Client, repo, settings, mgr.ThumbnailsDir(), model.PublishReleaseRequest{
				ProjectID:  v.ID,
				TargetID:   tgt.ID,
				SourcePath: artifactPath,
				Version:    resolvedVersion,
				Channel:    *channel,
				AssetKind:  assetKind,
				Changelog:  notes,
			}, nil)
			if err != nil {
				log.Fatalf("publish %s [%s] failed: %v", v.Slug, tgt.Name, err)
			}
			fmt.Printf("  -> Published %s to %s (%d bytes)\n", rel.Version, rel.R2Key, rel.FileSize)
		}
	}

	syncRep, err := cloud.SyncCatalogWithR2(ctx, r2Client, repo, mgr.ThumbnailsDir())
	if err != nil {
		log.Fatalf("sync catalog failed: %v", err)
	}
	fmt.Printf("Catalog synced: pulled=%d pushed=%d updated_local=%d updated_remote=%d thumbs_up=%d thumbs_down=%d\n",
		syncRep.Pulled, syncRep.Pushed, syncRep.UpdatedLocal, syncRep.UpdatedRemote, syncRep.ThumbnailsUp, syncRep.ThumbnailsDown)

	finalViews, err := repo.ListProjects(model.ProjectFilter{})
	if err != nil {
		log.Fatalf("list final projects: %v", err)
	}
	printSummary(finalViews, *slugFilter)
}

func resolveTargetArtifact(localDir string, tgt model.ProjectTarget) (string, error) {
	baseDir, err := fsutil.ResolveSafeSubpath(localDir, tgt.Subpath)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(tgt.ArtifactGlob) != "" {
		return launcher.ResolveArtifactGlob(baseDir, tgt.ArtifactGlob)
	}
	if strings.TrimSpace(tgt.LaunchEntry) != "" {
		if candidate, err := fsutil.ResolveSafeSubpath(localDir, tgt.LaunchEntry); err == nil {
			if _, statErr := os.Stat(candidate); statErr == nil {
				return candidate, nil
			}
		}
		if candidateSub, err := fsutil.ResolveSafeSubpath(baseDir, tgt.LaunchEntry); err == nil {
			if _, statErr := os.Stat(candidateSub); statErr == nil {
				return candidateSub, nil
			}
		}
	}
	return "", fmt.Errorf("no artifact_glob or launch_entry found")
}

func printSummary(views []model.ProjectView, slugFilter string) {
	count := 0
	fmt.Printf("\nT-Deck Catalog Summary:\n")
	for _, v := range views {
		if slugFilter != "" && !strings.EqualFold(v.Slug, slugFilter) {
			continue
		}
		count++
		ver := v.LatestVersion
		if ver == "" {
			ver = scanner.ResolveProjectVersion(v.LocalDir, nil)
		}
		fmt.Printf("- %-34s | %-7s | %-21s | targets=%d | ver=%-8s | entry=%-35s | thumb=%v\n",
			v.Slug, v.Type, emptyDash(v.Category), len(v.Targets), ver, emptyDash(firstNonEmpty(v.ExeName, v.BatFile)), v.ThumbnailURL != "")
	}
	fmt.Printf("Total matched projects: %d\n", count)
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func emptyDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

func generateThumbnailsConcurrent(repo *database.ProjectRepo, thumbsDir string, projects []model.Project, localDirByID map[string]string) {
	if err := os.MkdirAll(thumbsDir, 0o755); err != nil {
		fmt.Printf("  warning: create thumbnails dir failed: %v\n", err)
		return
	}

	workers := runtime.NumCPU()
	if workers > 8 {
		workers = 8
	}
	if workers < 1 {
		workers = 1
	}

	type thumbResult struct {
		proj    model.Project
		hasIcon bool
		err     error
	}

	jobs := make(chan model.Project, len(projects))
	results := make(chan thumbResult, len(projects))
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for proj := range jobs {
				brandLogo := media.DiscoverProjectBrandLogo(localDirByID[proj.ID])
				raw, err := media.RenderProjectThumbnailPNGWithLogo(proj, brandLogo)
				if err == nil {
					thumbPath := filepath.Join(thumbsDir, proj.ID+".png")
					err = os.WriteFile(thumbPath, raw, 0o644)
				}
				iconPath := filepath.Join(thumbsDir, proj.ID+".icon.png")
				hasIcon := false
				if err == nil {
					if brandLogo != nil {
						if iconBytes, iErr := media.EncodeBrandIconPNG(brandLogo, 128); iErr == nil {
							if os.WriteFile(iconPath, iconBytes, 0o644) == nil {
								hasIcon = true
							}
						}
					} else {
						_ = os.Remove(iconPath)
					}
				}
				results <- thumbResult{proj: proj, hasIcon: hasIcon, err: err}
			}
		}()
	}

	for _, p := range projects {
		jobs <- p
	}
	close(jobs)
	wg.Wait()
	close(results)

	now := time.Now().Unix()
	okCount := 0
	failCount := 0
	for res := range results {
		if res.err != nil {
			failCount++
			fmt.Printf("  warning: generate thumbnail for %s failed: %v\n", res.proj.Slug, res.err)
			continue
		}
		ts := now
		if ts <= res.proj.ThumbnailUpdatedAt {
			ts = res.proj.ThumbnailUpdatedAt + 1
		}
		if err := repo.UpdateThumbnailTimestamp(res.proj.ID, ts); err != nil {
			failCount++
			fmt.Printf("  warning: update timestamp for %s failed: %v\n", res.proj.Slug, err)
			continue
		}
		repo.MarkThumbnailExists(res.proj.ID, true)
		repo.MarkThumbnailExists(res.proj.ID+".icon", res.hasIcon)
		okCount++
	}
	fmt.Printf("Thumbnails generated: ok=%d failed=%d (workers=%d)\n", okCount, failCount, workers)
}
