package admin

import "database/sql"

type Stats struct {
	Articles, Videos, Skaters, Pending int
	LastSyncStatus                     string
	LastSyncAt                         string
}

func StatsFrom(db *sql.DB) (Stats, error) {
	var s Stats
	queries := []struct {
		q string
		n *int
	}{
		{`SELECT COUNT(*) FROM articles`, &s.Articles},
		{`SELECT COUNT(*) FROM youtube_videos WHERE is_hidden = 0`, &s.Videos},
		{`SELECT COUNT(*) FROM skater_profiles`, &s.Skaters},
		{`SELECT COUNT(*) FROM access_requests WHERE status = 'pending'`, &s.Pending},
	}
	for _, q := range queries {
		if err := db.QueryRow(q.q).Scan(q.n); err != nil {
			return s, err
		}
	}
	err := db.QueryRow(`SELECT status, started_at FROM sync_logs WHERE service = 'youtube' ORDER BY started_at DESC LIMIT 1`).Scan(&s.LastSyncStatus, &s.LastSyncAt)
	if err == sql.ErrNoRows {
		return s, nil
	}
	return s, err
}
