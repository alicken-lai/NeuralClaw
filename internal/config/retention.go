package config

// RetentionPolicy defines the time-to-live (in days) for different memory types.
type RetentionPolicy struct {
	RawDays            int `mapstructure:"raw_days" yaml:"raw_days"`
	DailySummaryDays   int `mapstructure:"daily_summary_days" yaml:"daily_summary_days"`
	WeeklySummaryDays  int `mapstructure:"weekly_summary_days" yaml:"weekly_summary_days"`
	MonthlySummaryDays int `mapstructure:"monthly_summary_days" yaml:"monthly_summary_days"`
	ConceptEdgesDays   int `mapstructure:"concept_edges_days" yaml:"concept_edges_days"`
	AnomaliesDays      int `mapstructure:"anomalies_days" yaml:"anomalies_days"`
}

// DefaultRetentionPolicy returns the Google-like token-efficient default policy.
func DefaultRetentionPolicy() RetentionPolicy {
	return RetentionPolicy{
		RawDays:            90,
		DailySummaryDays:   730,  // 2 years
		WeeklySummaryDays:  1825, // 5 years
		MonthlySummaryDays: 3650, // 10 years (or 1825 if tight, defaults to 3650 but user can config)
		ConceptEdgesDays:   1825, // 5 years
		AnomaliesDays:      730,  // 2 years
	}
}
