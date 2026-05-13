package fetcher

import "time"

// Usage is the normalized snapshot returned by Fetch.
//
// Fields map to claude.ai /api/organizations/<id>/usage payload (captured
// 2026-05-13). Bucket objects in that payload can be either {utilization,
// resets_at} or JSON null; both decode to a zero-valued Bucket here, which
// renderers treat as 0% / "no reset known".
type Usage struct {
	Session Bucket `json:"five_hour"`
	Weekly  Bucket `json:"seven_day"`
	Sonnet  Bucket `json:"seven_day_sonnet"`
	Design  Bucket `json:"seven_day_omelette"`

	// FetchedAt is set by the fetcher; not part of the upstream payload.
	FetchedAt time.Time `json:"-"`
}

// Bucket is one row on the Usage page.
type Bucket struct {
	PercentUsed float64   `json:"utilization"`
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
