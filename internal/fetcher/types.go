package fetcher

import "time"

// Usage is the normalized snapshot returned by Fetch.
//
// The fields mirror the four buckets visible on claude.ai Settings -> Usage.
// If Anthropic adds or renames a bucket, extend this struct and the JSON
// shape in fetcher.go in one place.
type Usage struct {
	Session Bucket `json:"current_session"`
	Weekly  Bucket `json:"weekly_all_models"`
	Sonnet  Bucket `json:"weekly_sonnet"`
	Design  Bucket `json:"weekly_design"`

	// FetchedAt is set by the fetcher; not part of the upstream payload.
	FetchedAt time.Time `json:"-"`
}

// Bucket is one row on the Usage page.
type Bucket struct {
	PercentUsed float64   `json:"percent_used"`
	ResetsAt    time.Time `json:"resets_at"`
}

// Buckets returns the four buckets in display order. Stable for iteration.
func (u Usage) Buckets() []NamedBucket {
	return []NamedBucket{
		{Name: "Session", Bucket: u.Session},
		{Name: "Weekly", Bucket: u.Weekly},
		{Name: "Sonnet", Bucket: u.Sonnet},
		{Name: "Design", Bucket: u.Design},
	}
}

// NamedBucket pairs a Bucket with its display label.
type NamedBucket struct {
	Name string
	Bucket
}
