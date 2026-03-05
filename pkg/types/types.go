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
	SourceURI      string `json:"source_uri,omitempty"`  // NEW: Absolute link or URI
	SourceKind     string `json:"source_kind,omitempty"` // NEW: "ocr", "pdf", etc.
	Page           *int   `json:"page,omitempty"`        // NEW: Pointer for nil-ability
	ChunkIndex     *int   `json:"chunk_index,omitempty"` // NEW: Pointer for nil-ability
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

	// Living Memory fields — memories that evolve with usage.
	AccessCount    int        `json:"access_count"`               // Times this memory was retrieved by agent/DMN
	LastAccessedAt *time.Time `json:"last_accessed_at,omitempty"` // Timestamp of last retrieval
	CausalLinks    []string   `json:"causal_links,omitempty"`     // IDs of ancestor/related memories (DAG edges)

	// Evidence Backlinks
	DerivedFrom []string `json:"derived_from,omitempty"` // IDs of parent memories (raw evidence)
	EvidenceOf  []string `json:"evidence_of,omitempty"`  // IDs of summaries this supports
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
	Explain         bool // [NEW] Enable scoring breakdown
}

// Filter allows refining memory queries.
type Filter struct {
	Type    *ItemType
	Project *string
	User    *string
	Session *string
	Tags    []string
}

// ScoreBreakdown provides a detailed look at how a memory item was ranked.
type ScoreBreakdown struct {
	VectorScore float64  `json:"vector_score"`
	BM25Score   float64  `json:"bm25_score"`
	TimeBoost   float64  `json:"time_boost"`
	RRFScore    float64  `json:"rrf_score"`
	RerankScore *float64 `json:"rerank_score,omitempty"`
	FinalScore  float64  `json:"final_score"`
	AccessBoost float64  `json:"access_boost,omitempty"` // LTP effect
	Notes       []string `json:"notes,omitempty"`
}

// ExplainedHit pairs an item with its score metadata.
type ExplainedHit struct {
	Item  MemoryItem     `json:"item"`
	Score ScoreBreakdown `json:"score"`
}

// QueryResult returns matching memory items along with any search metadata.
type QueryResult struct {
	Items         []MemoryItem   `json:"items"`
	Scores        []float32      `json:"scores"`
	TotalFound    int            `json:"total_found"`
	ExplainedHits []ExplainedHit `json:"explained_hits,omitempty"` // [NEW]
	TookMillis    int64          `json:"took_millis,omitempty"`    // [NEW]
}
