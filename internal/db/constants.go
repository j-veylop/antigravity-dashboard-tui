package db

// SQL query fragments used across multiple functions
const (
	// sqlTimeFilterClause is used to filter queries by a datetime window
	sqlTimeFilterClause = "AND bucket_time >= datetime('now', ?)"
)
