import { useCallback, useEffect, useRef, useState } from "react";
import type { ProjectSortBy } from "../lib/facets";

export type QuickFilter = "all" | "has_release" | "needs_update" | "running";

export function useProjectFilters() {
  const [viewMode, setViewMode] = useState<"grid" | "matrix">(() => {
    try {
      return localStorage.getItem("tdeck_view_mode") === "matrix"
        ? "matrix"
        : "grid";
    } catch {
      return "grid";
    }
  });

  const [selectedType, setSelectedType] = useState("all");
  const [selectedCategory, setSelectedCategory] = useState("all");
  const [selectedStatus, setSelectedStatus] = useState("all");
  const [selectedTag, setSelectedTag] = useState("");
  const [selectedQuickFilter, setSelectedQuickFilter] =
    useState<QuickFilter>("all");
  const [query, setQuery] = useState("");
  const [debouncedQuery, setDebouncedQuery] = useState("");
  const [isComposing, setIsComposing] = useState(false);

  const [sortBy, setSortBy] = useState<ProjectSortBy>(() => {
    try {
      const saved = localStorage.getItem(
        "tdeck_sort_by"
      ) as ProjectSortBy | null;
      return saved || "name_asc";
    } catch {
      return "name_asc";
    }
  });

  const searchInputRef = useRef<HTMLInputElement>(null);
  const hasActiveQuery = Boolean(debouncedQuery.trim());

  const handleViewModeChange = useCallback((mode: "grid" | "matrix") => {
    setViewMode(mode);
    try {
      localStorage.setItem("tdeck_view_mode", mode);
    } catch {
      // ignore storage quota errors
    }
  }, []);

  const handleSortByChange = useCallback((nextSort: ProjectSortBy) => {
    setSortBy(nextSort);
    try {
      localStorage.setItem("tdeck_sort_by", nextSort);
    } catch {
      // ignore storage quota errors
    }
  }, []);

  const handleSearchCommit = useCallback(() => {
    const trimmed = query.trim();
    if (trimmed === "" || (trimmed.length < 2 && !trimmed.startsWith("#"))) {
      setDebouncedQuery("");
      return;
    }
    setDebouncedQuery(trimmed);
  }, [query]);

  useEffect(() => {
    if (isComposing) return;
    const trimmed = query.trim();
    if (trimmed === "" || (trimmed.length < 2 && !trimmed.startsWith("#"))) {
      setDebouncedQuery("");
      return;
    }
    const timer = setTimeout(() => {
      setDebouncedQuery(trimmed);
    }, 320);
    return () => clearTimeout(timer);
  }, [query, isComposing]);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        searchInputRef.current?.focus();
        searchInputRef.current?.select();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);

  const handleTagClick = useCallback((tag: string) => {
    setSelectedTag((prev) =>
      prev.toLowerCase() === tag.toLowerCase() ? "" : tag
    );
  }, []);

  const handleCategoryClick = useCallback((cat: string) => {
    setSelectedCategory((prev) =>
      prev.toLowerCase() === cat.toLowerCase() ? "all" : cat
    );
  }, []);

  const handleResetCategory = useCallback(() => {
    setSelectedCategory("all");
  }, []);

  const clearQuery = useCallback(() => {
    setQuery("");
    setDebouncedQuery("");
  }, []);

  return {
    viewMode,
    selectedType,
    selectedCategory,
    selectedStatus,
    selectedTag,
    selectedQuickFilter,
    query,
    debouncedQuery,
    hasActiveQuery,
    sortBy,
    searchInputRef,
    setSelectedType,
    setSelectedCategory,
    setSelectedStatus,
    setSelectedTag,
    setSelectedQuickFilter,
    setQuery,
    setIsComposing,
    handleViewModeChange,
    handleSortByChange,
    handleSearchCommit,
    handleTagClick,
    handleCategoryClick,
    handleResetCategory,
    clearQuery,
  };
}

