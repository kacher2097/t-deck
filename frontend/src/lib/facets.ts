import type { ProjectView } from "../types/project";

export type ProjectSortBy =
  | "relevance"
  | "name_asc"
  | "name_desc"
  | "updated_desc"
  | "opened_desc"
  | "status_ready"
  | "category_asc"
  | "type_asc";

export interface FacetOptions {
  type?: string;
  category?: string;
  status?: string;
  tag?: string;
  quickFilter?: "all" | "has_release" | "needs_update" | "running";
  runningIds?: Set<string>;
  preserveOrder?: boolean;
  sortBy?: ProjectSortBy;
  hasQuery?: boolean;
}

const nameCollator = new Intl.Collator("vi", {
  sensitivity: "base",
  numeric: true,
});

export function compareProjectsAlpha(a: ProjectView, b: ProjectView): number {
  const byName = nameCollator.compare(a.name || a.slug, b.name || b.slug);
  if (byName !== 0) return byName;
  return (a.slug || "").localeCompare(b.slug || "");
}

function statusRank(p: ProjectView, runningIds?: Set<string>): number {
  if (runningIds?.has(p.id)) return 0;
  switch (p.status) {
    case "ready":
      return 1;
    case "missing_env":
      return 2;
    case "cloud_only":
      return 3;
    case "missing_dir":
      return 4;
    default:
      return 5;
  }
}

function typeRank(t: string): number {
  switch (t) {
    case "desktop":
      return 0;
    case "hybrid":
      return 1;
    default:
      return 2;
  }
}

export function applyFacets(
  projects: ProjectView[],
  opts: FacetOptions
): ProjectView[] {
  const filtered = projects.filter((p) => {
    if (opts.type && opts.type !== "all" && p.type !== opts.type) {
      return false;
    }
    if (opts.category && opts.category !== "all") {
      if (opts.category === "__uncategorized__") {
        if ((p.category || "").trim() !== "") return false;
      } else if (
        (p.category || "").trim().toLowerCase() !==
        opts.category.trim().toLowerCase()
      ) {
        return false;
      }
    }
    if (opts.status && opts.status !== "all" && p.status !== opts.status) {
      return false;
    }
    if (opts.tag) {
      const target = opts.tag.toLowerCase();
      if (!(p.tags || []).some((t) => t.toLowerCase() === target)) {
        return false;
      }
    }
    if (opts.quickFilter && opts.quickFilter !== "all") {
      if (opts.quickFilter === "has_release") {
        const hasRel =
          Boolean(p.latest_version) ||
          Object.keys(p.latest_releases || {}).length > 0;
        if (!hasRel) return false;
      } else if (opts.quickFilter === "needs_update") {
        if (!p.has_update) return false;
      } else if (opts.quickFilter === "running") {
        if (!opts.runningIds?.has(p.id)) return false;
      }
    }
    return true;
  });

  const sortMode: ProjectSortBy =
    opts.sortBy || (opts.preserveOrder ? "relevance" : "name_asc");

  if (sortMode === "relevance") {
    if (opts.hasQuery ?? opts.preserveOrder) {
      return filtered;
    }
    return [...filtered].sort(compareProjectsAlpha);
  }

  return [...filtered].sort((a, b) => {
    switch (sortMode) {
      case "name_desc":
        return compareProjectsAlpha(b, a);
      case "updated_desc": {
        const diff = (b.updated_at || 0) - (a.updated_at || 0);
        return diff !== 0 ? diff : compareProjectsAlpha(a, b);
      }
      case "opened_desc": {
        const diff = (b.last_opened_at || 0) - (a.last_opened_at || 0);
        if (diff !== 0) return diff;
        const upd = (b.updated_at || 0) - (a.updated_at || 0);
        return upd !== 0 ? upd : compareProjectsAlpha(a, b);
      }
      case "status_ready": {
        const diff = statusRank(a, opts.runningIds) - statusRank(b, opts.runningIds);
        return diff !== 0 ? diff : compareProjectsAlpha(a, b);
      }
      case "category_asc": {
        const ca = (a.category || "").trim();
        const cb = (b.category || "").trim();
        if (ca === "" && cb !== "") return 1;
        if (ca !== "" && cb === "") return -1;
        const diff = nameCollator.compare(ca, cb);
        return diff !== 0 ? diff : compareProjectsAlpha(a, b);
      }
      case "type_asc": {
        const diff = typeRank(a.type) - typeRank(b.type);
        return diff !== 0 ? diff : compareProjectsAlpha(a, b);
      }
      case "name_asc":
      default:
        return compareProjectsAlpha(a, b);
    }
  });
}

export interface SidebarCounts {
  total: number;
  categoryMap: Map<string, number>;
  uncategorized: number;
  web: number;
  desktop: number;
  hybrid: number;
  ready: number;
  cloudOnly: number;
  missingEnv: number;
  missingDir: number;
  sortedTags: [string, number][];
}

export function computeSidebarCounts(
  projects: ProjectView[],
  selectedTag: string
): SidebarCounts {
  const categoryMap = new Map<string, number>();
  const tagMap = new Map<string, number>();
  let uncategorized = 0;
  let web = 0;
  let desktop = 0;
  let hybrid = 0;
  let ready = 0;
  let cloudOnly = 0;
  let missingEnv = 0;
  let missingDir = 0;

  for (const p of projects) {
    const c = (p.category || "").trim();
    if (!c) {
      uncategorized++;
    } else {
      const k = c.toLowerCase();
      categoryMap.set(k, (categoryMap.get(k) || 0) + 1);
    }

    if (p.type === "web") web++;
    else if (p.type === "desktop") desktop++;
    else if (p.type === "hybrid") hybrid++;

    if (p.status === "ready") ready++;
    else if (p.status === "cloud_only") cloudOnly++;
    else if (p.status === "missing_env") missingEnv++;
    else if (p.status === "missing_dir") missingDir++;

    for (const t of p.tags || []) {
      const clean = t.trim();
      if (!clean) continue;
      tagMap.set(clean, (tagMap.get(clean) || 0) + 1);
    }
  }

  if (
    selectedTag &&
    !Array.from(tagMap.keys()).some(
      (t) => t.toLowerCase() === selectedTag.toLowerCase()
    )
  ) {
    tagMap.set(selectedTag, 0);
  }

  const sortedTags = Array.from(tagMap.entries()).sort((a, b) => {
    if (b[1] !== a[1]) return b[1] - a[1];
    return a[0].localeCompare(b[0]);
  });

  return {
    total: projects.length,
    categoryMap,
    uncategorized,
    web,
    desktop,
    hybrid,
    ready,
    cloudOnly,
    missingEnv,
    missingDir,
    sortedTags,
  };
}
