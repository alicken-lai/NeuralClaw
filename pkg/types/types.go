package types

import (
	"time"

	"neuralclaw/internal/config"
)

// ScopeLevel indicates the isolated environment context.
type ScopeLevel string

const (
	ScopeGlobal  ScopeLevel = "global"
	ScopeProject ScopeLevel = "project"
	ScopeUser    ScopeLevel = "user"
	ScopeSession ScopeLevel = "session"
)

// ItemType defines what kind of memory this is to prevent DMN summaries
// from polluting raw memories unless intentionally queried together.
type ItemType string

const (
	ItemTypeRaw            ItemType = "raw"
	ItemTypeDailySummary   ItemType = "daily_summary"
	ItemTypeWeeklySummary  ItemType = "weekly_summary"
	ItemTypeMonthlySummary ItemType = "monthly_summary"
	ItemTypeConceptEdges   ItemType = "concept_edges"
	ItemTypeAnomalies      ItemType = "anomalies"
)

// Provenance tracks the origin of the memory item.
type Provenance struct {
	SourceFilePath string `json:"source_file_path,omitempty"`
	Page           int    `json:"page,omitempty"`
	Hash           string `json:"hash,omitempty"`
	ToolVersion    string `json:"tool_version,omitempty"`
}

// MemoryItem is the core structure for storing and retrieving facts/experiences.
type MemoryItem struct {
	ID      string   `json:"id"`
	Type    ItemType `json:"type"`  // Used to differentiate raw vs summarized DMN items
	Scope   string   `json:"scope"` // Format could be "global", "project:xyz", etc.
	Project string   `json:"project"`
	User    string   `json:"user"`
	Session string   `json:"session"`

	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	SourceTime time.Time `json:"source_time"` // Time-Aware feature: original doc time

	Modality string   `json:"modality"` // e.g. text, ocr, pdf, image, chat, log
	Tags     []string `json:"tags"`

	// LanceDB Pro Compatible Fields
	Text       string    `json:"text"`       // Main textual content of the memory
	Category   string    `json:"category"`   // General classification
	Importance float32   `json:"importance"` // 0.0 to 1.0 importance scale
	Timestamp  int64     `json:"timestamp"`  // Unix seconds
	Vector     []float32 `json:"vector"`     // Embedding payload
	Metadata   string    `json:"metadata"`   // JSON string of arbitrary metadata
	BM25Text   string    `json:"bm25_text"`  // Payload for keyword indexing

	TimeDecayPolicy string `json:"time_decay_policy"`
	TTLDays         *int   `json:"ttl_days"` // optional per-item override

	Provenance Provenance `json:"provenance"`
}

// EffectiveTTLDays returns the configured TTL for the memory type, or the item-level override if set.
func (m MemoryItem) EffectiveTTLDays(policy config.RetentionPolicy) int {
	if m.TTLDays != nil {
		return *m.TTLDays
	}

	switch m.Type {
	case ItemTypeRaw:
		return policy.RawDays
	case ItemTypeDailySummary:
		return policy.DailySummaryDays
	case ItemTypeWeeklySummary:
		return policy.WeeklySummaryDays
	case ItemTypeMonthlySummary:
		return policy.MonthlySummaryDays
	case ItemTypeConceptEdges:
		return policy.ConceptEdgesDays
	case ItemTypeAnomalies:
		return policy.AnomaliesDays
	}

	return policy.RawDays // safety fallback
}

// Query structure for hybrid searches.
type Query struct {
	Text            string
	Vector          []float32
	TopK            int
	Scope           string
	TimeWindowStart *time.Time
	TimeWindowEnd   *time.Time
	Filter          Filter
}

// Filter allows refining memory queries.
type Filter struct {
	Type    *ItemType
	Project *string
	User    *string
	Session *string
	Tags    []string
}

// QueryResult returns matching memory items along with any search metadata.
type QueryResult struct {
	Items      []MemoryItem
	Scores     []float32
	TotalFound int
}
