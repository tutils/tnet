package counter

// Counter is a cumulative metric
type Counter interface {
	Value() int64
	IncreaceRatePerSec() int64

	Add(bytes int64)
}
