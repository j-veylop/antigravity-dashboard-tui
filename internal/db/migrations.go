package db

import (
	"context"
	"fmt"
)

// FixLegacyTimeFormats fixes timestamp formats in the database.
// This is required because modernc.org/sqlite does not store time.Time in a format
// compatible with SQLite's date/time functions by default.
func (db *DB) FixLegacyTimeFormats() error {
	queries := []string{
		// Fix quota_snapshots_agg bucket_time (truncate " +0000 UTC")
		`UPDATE quota_snapshots_agg 
		 SET bucket_time = SUBSTR(bucket_time, 1, 19) 
		 WHERE length(bucket_time) > 19 AND bucket_time LIKE '% UTC'`,
		
		// Fix quota_snapshots timestamp
		`UPDATE quota_snapshots 
		 SET timestamp = SUBSTR(timestamp, 1, 19) 
		 WHERE length(timestamp) > 19 AND timestamp LIKE '% UTC'`,
		
		// Fix api_calls timestamp
		`UPDATE api_calls 
		 SET timestamp = SUBSTR(timestamp, 1, 19) 
		 WHERE length(timestamp) > 19 AND timestamp LIKE '% UTC'`,
	}

	for _, query := range queries {
		if _, err := db.ExecContext(context.Background(), query); err != nil {
			return fmt.Errorf("failed to fix legacy time formats: %w", err)
		}
	}

	return nil
}
