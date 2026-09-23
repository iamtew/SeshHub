package twitch

import (
	"context"
	"database/sql"
	"time"
)

type tracked struct {
	ID, Login, Title, Name string
	Started                string
}

func Poll(ctx context.Context, db *sql.DB, c *Client) ([]Live, error) {
	if !c.Configured() {
		return nil, nil
	}
	rows, err := db.QueryContext(ctx, `SELECT id, twitch_login, IFNULL(twitch_title,''), IFNULL(twitch_started_at,''),
		COALESCE(NULLIF(TRIM(real_name),''), skater_name)
		FROM skater_profiles WHERE twitch_login != ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []tracked
	var logins []string
	for rows.Next() {
		var t tracked
		if err := rows.Scan(&t.ID, &t.Login, &t.Title, &t.Started, &t.Name); err != nil {
			return nil, err
		}
		t.Login, _ = Normalize(t.Login)
		if t.Login == "" {
			continue
		}
		list = append(list, t)
		logins = append(logins, t.Login)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	live, err := c.Streams(ctx, logins)
	if err != nil {
		return nil, err
	}
	var went []Live
	for _, t := range list {
		s, on := live[t.Login]
		was := t.Started != ""
		if !on {
			if was {
				if err := clearLive(db, t.ID); err != nil {
					return went, err
				}
			}
			continue
		}
		if !was {
			went = append(went, Live{Name: t.Name, Login: t.Login, Title: s.Title, Started: s.Started})
		}
		if err := setLive(db, t.ID, s.Title, s.Started); err != nil {
			return went, err
		}
	}
	return went, nil
}

func setLive(db *sql.DB, id, title string, started time.Time) error {
	_, err := db.Exec(`UPDATE skater_profiles SET twitch_title=?, twitch_started_at=? WHERE id=?`,
		title, started.UTC().Format(time.RFC3339), id)
	return err
}

func clearLive(db *sql.DB, id string) error {
	_, err := db.Exec(`UPDATE skater_profiles SET twitch_title='', twitch_started_at=NULL WHERE id=?`, id)
	return err
}
