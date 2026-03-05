package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"neuralclaw/internal/observability"
)

// handleTokenDashboard renders the Token Usage Dashboard page.
func (s *Server) handleTokenDashboard(w http.ResponseWriter, r *http.Request) {
	scope := GetScopeConstraint(r.Context())

	var dailySummaries []observability.DailySummary
	var sourceSummaries []observability.SourceSummary
	var totalInput, totalOutput, totalAll int

	if observability.Tracker != nil {
		dailySummaries = observability.Tracker.GetDailySummaries(5)
		sourceSummaries = observability.Tracker.GetSourceSummaries()

		sort.Slice(dailySummaries, func(i, j int) bool {
			return dailySummaries[i].Date > dailySummaries[j].Date
		})

		for _, ds := range dailySummaries {
			totalInput += ds.InputTokens
			totalOutput += ds.OutputTokens
			totalAll += ds.TotalTokens
		}
	}

	data := struct {
		Scope           string
		DailySummaries  []observability.DailySummary
		SourceSummaries []observability.SourceSummary
		TotalInput      int
		TotalOutput     int
		TotalAll        int
	}{
		Scope:           scope,
		DailySummaries:  dailySummaries,
		SourceSummaries: sourceSummaries,
		TotalInput:      totalInput,
		TotalOutput:     totalOutput,
		TotalAll:        totalAll,
	}

	s.renderTemplate(w, "tokens.html", data)
}

// FileEntry represents a file or directory in the context browser.
type FileEntry struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	IsDir      bool   `json:"is_dir"`
	SizeBytes  int64  `json:"size_bytes"`
	SizeHuman  string `json:"size_human"`
	EstTokens  int    `json:"est_tokens"`
	ChildCount int    `json:"child_count,omitempty"`
}

// handleContextBrowser renders the file context browser page.
func (s *Server) handleContextBrowser(w http.ResponseWriter, r *http.Request) {
	scope := GetScopeConstraint(r.Context())
	baseDir, _ := os.Getwd()

	entries := scanDirectory(baseDir, 0)

	data := struct {
		Scope   string
		BaseDir string
		Files   []FileEntry
	}{
		Scope:   scope,
		BaseDir: baseDir,
		Files:   entries,
	}

	s.renderTemplate(w, "context_browser.html", data)
}

// handleContextAPI returns JSON file listing for HTMX lazy loading.
func (s *Server) handleContextAPI(w http.ResponseWriter, r *http.Request) {
	dir := r.URL.Query().Get("dir")
	if dir == "" {
		dir, _ = os.Getwd()
	}

	entries := scanDirectory(dir, 0)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

func scanDirectory(dir string, depth int) []FileEntry {
	if depth > 3 {
		return nil
	}

	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var entries []FileEntry
	for _, de := range dirEntries {
		name := de.Name()
		if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" {
			continue
		}

		fullPath := filepath.Join(dir, name)
		entry := FileEntry{
			Name:  name,
			Path:  fullPath,
			IsDir: de.IsDir(),
		}

		if de.IsDir() {
			children, _ := os.ReadDir(fullPath)
			entry.ChildCount = len(children)
		} else {
			info, err := de.Info()
			if err == nil {
				entry.SizeBytes = info.Size()
				entry.SizeHuman = humanSize(info.Size())
				entry.EstTokens = int(info.Size() / 4) // rough heuristic: ~4 bytes per token
			}
		}

		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})

	return entries
}

func humanSize(bytes int64) string {
	sizes := []string{"B", "KB", "MB", "GB"}
	i := 0
	fBytes := float64(bytes)
	for fBytes >= 1024 && i < len(sizes)-1 {
		fBytes /= 1024
		i++
	}
	if fBytes == float64(int(fBytes)) {
		return fmt.Sprintf("%d %s", int(fBytes), sizes[i])
	}
	return fmt.Sprintf("%.1f %s", fBytes, sizes[i])
}
