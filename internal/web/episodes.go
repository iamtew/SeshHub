package web

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"seshhub/internal/episode"
	"seshhub/internal/yt"
)

type epCard struct {
	URL, Thumb, Title, Meta string
}

type epView struct {
	episode.Episode
	Cards []epCard
}

func (s *Server) episodes(w http.ResponseWriter, r *http.Request) {
	episode.EnsureCurrent(s.db)
	list, err := episode.List(s.db)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	var ids []string
	for _, e := range list {
		for _, row := range e.VisibleRows() {
			if row.Thumb() != "" {
				ids = append(ids, episode.ParseVideoID(row.URL))
			}
		}
	}
	vids := yt.GetMany(s.db, ids)
	views := make([]epView, 0, len(list))
	for _, e := range list {
		views = append(views, epView{Episode: e, Cards: episodeCards(e, vids)})
	}
	s.render(w, r, "episodes.html", map[string]any{"Title": "Episodes", "Path": "/episodes", "Episodes": views})
}

func episodeCards(e episode.Episode, vids map[string]yt.Video) []epCard {
	var out []epCard
	for _, row := range e.VisibleRows() {
		if row.Thumb() == "" {
			continue
		}
		id := episode.ParseVideoID(row.URL)
		c := epCard{URL: row.URL, Thumb: row.Thumb(), Title: row.Label}
		if v, ok := vids[id]; ok {
			if v.Title != "" {
				c.Title = v.Title
			}
			if v.Thumb != "" {
				c.Thumb = v.Thumb
			}
			c.Meta = fmt.Sprintf("%s · %d views", v.Category, v.Views)
		}
		out = append(out, c)
	}
	return out
}

func (s *Server) adminEpisodes(w http.ResponseWriter, r *http.Request) {
	if s.requireHost(w, r) == nil {
		return
	}
	episode.EnsureCurrent(s.db)
	list, err := episode.List(s.db)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.render(w, r, "admin_episodes.html", map[string]any{"Title": "Episodes", "Path": "/admin/episodes", "Episodes": list})
}

func (s *Server) adminEpisodeCreate(w http.ResponseWriter, r *http.Request) {
	if s.requireHost(w, r) == nil {
		return
	}
	if r.Method != http.MethodPost {
		s.render(w, r, "episode_form.html", map[string]any{
			"Title": "New episode", "Path": "/admin/episodes", "E": episode.Episode{}, "Action": "/admin/episodes", "New": true, "FormRows": []episode.Row{{}},
		})
		return
	}
	_ = r.ParseForm()
	n, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("number")))
	if n <= 0 {
		http.Error(w, "episode number required", http.StatusBadRequest)
		return
	}
	if _, err := episode.Get(s.db, n); err == nil {
		http.Redirect(w, r, "/admin/episodes/"+strconv.Itoa(n), http.StatusSeeOther)
		return
	}
	e := episode.Episode{Number: n, Title: strings.TrimSpace(r.FormValue("title"))}
	if err := episode.Save(s.db, e); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/admin/episodes/"+strconv.Itoa(n), http.StatusSeeOther)
}

func (s *Server) adminEpisodeEdit(w http.ResponseWriter, r *http.Request) {
	u := s.requireHost(w, r)
	if u == nil {
		return
	}
	n, err := strconv.Atoi(r.PathValue("n"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	e, err := episode.Get(s.db, n)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodPost {
		np := formEpisode(r, e.Number)
		if np.WinnerName == "" && np.WinnerVideoID != "" {
			np.WinnerName = s.winnerChannel(r.Context(), u.ID, np.ChallengePlaylistID, np.WinnerVideoID)
		}
		if err := episode.Save(s.db, np); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, "/admin/episodes/"+strconv.Itoa(n), http.StatusSeeOther)
		return
	}
	picks, _ := s.challengePicks(r.Context(), u.ID, e.ChallengePlaylistID)
	s.render(w, r, "episode_form.html", map[string]any{
		"Title": "Edit EP" + strconv.Itoa(e.Number), "Path": "/admin/episodes", "E": e, "Action": "/admin/episodes/" + strconv.Itoa(e.Number),
		"FormRows": append(append([]episode.Row{}, e.Rows...), episode.Row{}),
		"Picks":    picks,
	})
}

func (s *Server) adminEpisodeDelete(w http.ResponseWriter, r *http.Request) {
	if s.requireHost(w, r) == nil {
		return
	}
	n, _ := strconv.Atoi(r.PathValue("n"))
	_ = episode.Delete(s.db, n)
	http.Redirect(w, r, "/admin/episodes", http.StatusSeeOther)
}

func formEpisode(r *http.Request, n int) episode.Episode {
	_ = r.ParseForm()
	e := episode.Episode{
		Number:              n,
		Title:               strings.TrimSpace(r.FormValue("title")),
		Heading:             strings.TrimSpace(r.FormValue("heading")),
		SubmissionsCount:    episode.ParseCount(r.FormValue("submissions_count")),
		WinnerName:          strings.TrimSpace(r.FormValue("winner_name")),
		ChallengePlaylistID: episode.ParsePlaylistID(r.FormValue("challenge_playlist_id")),
		Note:                strings.TrimSpace(r.FormValue("note")),
	}
	if e.ChallengePlaylistID == "" {
		e.ChallengePlaylistID = strings.TrimSpace(r.FormValue("challenge_playlist_id"))
	}
	sel := episode.ParseVideoID(r.FormValue("winner_video_id"))
	paste := episode.ParseVideoID(r.FormValue("winner_url"))
	if paste != "" {
		e.WinnerVideoID = paste
	} else {
		e.WinnerVideoID = sel
	}
	kinds := r.Form["row_kind"]
	labels := r.Form["row_label"]
	urls := r.Form["row_url"]
	for i := range kinds {
		kind := strings.TrimSpace(kinds[i])
		var label, u string
		if i < len(labels) {
			label = strings.TrimSpace(labels[i])
		}
		if i < len(urls) {
			u = strings.TrimSpace(urls[i])
		}
		if kind == "" && label == "" && u == "" {
			continue
		}
		if kind == "" {
			kind = "full"
		}
		e.Rows = append(e.Rows, episode.Row{Kind: kind, Label: label, URL: u})
	}
	return e
}

func (s *Server) challengePicks(ctx context.Context, userID, playlistID string) ([]yt.PlaylistItem, error) {
	if playlistID == "" {
		return nil, nil
	}
	tok, err := yt.AccessToken(s.db, userID, s.cfg.YouTubeClientID, s.cfg.YouTubeClientSecret)
	if err != nil || tok == "" {
		return nil, err
	}
	return yt.Client{Token: tok}.PlaylistItems(ctx, playlistID)
}

func (s *Server) winnerChannel(ctx context.Context, userID, playlistID, videoID string) string {
	if picks, err := s.challengePicks(ctx, userID, playlistID); err == nil {
		if ch := yt.ChannelOf(picks, videoID); ch != "" {
			return ch
		}
	}
	return yt.OEmbedAuthor(ctx, videoID)
}
