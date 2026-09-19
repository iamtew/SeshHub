package admin

import "database/sql"

type Stats struct {
	Articles, Videos, Skaters, Pending int
}

func StatsFrom(db *sql.DB) (Stats, error) {
	var s Stats
	queries := []struct {
		q string
		n *int
	}{
		{`SELECT COUNT(*) FROM articles`, &s.Articles},
		{`SELECT COUNT(*) FROM youtube_videos WHERE is_hidden = 0`, &s.Videos},
		{`SELECT COUNT(*) FROM skater_profiles p LEFT JOIN users u ON u.id = p.user_id WHERE u.role IN ('skater','admin')`, &s.Skaters},
		{`SELECT COUNT(*) FROM access_requests WHERE status = 'pending'`, &s.Pending},
	}
	for _, q := range queries {
		if err := db.QueryRow(q.q).Scan(q.n); err != nil {
			return s, err
		}
	}
	return s, nil
}
