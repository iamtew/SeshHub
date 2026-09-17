package web

import (
	"database/sql"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"seshhub/internal/article"
	"seshhub/internal/auth"
	"seshhub/internal/page"
	"seshhub/internal/skater"
	"seshhub/internal/yt"
)

func formProfile(r *http.Request, existing skater.Profile) skater.Profile {
	existing.RealName = r.FormValue("display_name")
	existing.Slug = strings.TrimSpace(r.FormValue("slug"))
	existing.Bio = r.FormValue("bio")
	existing.Stance = r.FormValue("stance")
	existing.Location = r.FormValue("location")
	existing.AvatarR1, _ = strconv.Atoi(r.FormValue("avatar_r1"))
	existing.AvatarR2, _ = strconv.Atoi(r.FormValue("avatar_r2"))
	existing.AvatarR3, _ = strconv.Atoi(r.FormValue("avatar_r3"))
	existing.AvatarR4, _ = strconv.Atoi(r.FormValue("avatar_r4"))
	existing.AvatarBorderStyle = r.FormValue("avatar_border_style")
	existing.AvatarBorderColor = r.FormValue("avatar_border_color")
	existing.AvatarBorder, _ = strconv.Atoi(r.FormValue("avatar_border"))
	existing.AvatarBorderBlur, _ = strconv.Atoi(r.FormValue("avatar_border_blur"))
	if _, ok := r.PostForm["featured_video_id"]; ok {
		existing.FeaturedVideoID = r.FormValue("featured_video_id")
	}
	return existing
}

func applyPhoto(r *http.Request, p skater.Profile) (skater.Profile, error) {
	f, hdr, err := r.FormFile("avatar")
	if err == nil {
		defer f.Close()
		if hdr.Size > 0 {
			url, err := skater.SavePhoto(p.ID, f)
			if err != nil {
				return p, err
			}
			p.PhotoURL = url
			return p, nil
		}
	} else if err != http.ErrMissingFile && err != http.ErrNotMultipart {
		return p, err
	}
	if r.FormValue("avatar_reset") == "1" {
		skater.RemovePhoto(p.ID)
		p.PhotoURL = ""
	}
	return p, nil
}

func skaterView(p skater.Profile) map[string]any {
	return map[string]any{
		"P": p, "Sponsors": skater.Lines(p.Sponsors), "Tricks": skater.Lines(p.SignatureTricks),
		"Social": skater.Lines(p.SocialLinks),
	}
}

func (s *Server) mediaAvatar(w http.ResponseWriter, r *http.Request) {
	path, ok := skater.PhotoFile(r.PathValue("file"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, path)
}

func (s *Server) team(w http.ResponseWriter, r *http.Request) {
	list, err := skater.ListTeam(s.db)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	data := map[string]any{"Title": "FS Team", "Path": "/team", "Skaters": list}
	if p, err := page.Get(s.db, "slug", "team"); err == nil {
		data["Title"] = p.Title
		data["HTML"] = template.HTML(article.Render(p.ContentRaw))
		if p.CSS != "" {
			data["CSS"] = template.CSS(p.CSS)
		}
	}
	if u := UserFrom(r); u != nil && u.Role == auth.RoleAdmin {
		data["EditHref"] = "/admin/pages/" + page.TeamID + "?next=/team"
	}
	s.render(w, r, "skaters_list.html", data)
}

func (s *Server) friends(w http.ResponseWriter, r *http.Request) {
	list, err := skater.ListFriends(s.db)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.render(w, r, "skaters_list.html", map[string]any{"Title": "Friends", "Path": "/friends", "Skaters": list})
}

func (s *Server) skatersAlias(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/team", http.StatusFound)
}

func (s *Server) skaterDetail(w http.ResponseWriter, r *http.Request) {
	want := r.PathValue("slug")
	p, err := skater.Get(s.db, "slug", want)
	if err == sql.ErrNoRows {
		cur, rerr := skater.CurrentSlug(s.db, want)
		if rerr == nil {
			http.Redirect(w, r, rosterURL(s.db, cur), http.StatusFound)
			return
		}
		http.NotFound(w, r)
		return
	}
	if err == nil && p.UserID == "" {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	wantPath := skater.RosterPath(p.Role)
	if !strings.HasPrefix(r.URL.Path, wantPath+"/") {
		http.Redirect(w, r, wantPath+"/"+p.Slug, http.StatusFound)
		return
	}
	u := UserFrom(r)
	name := p.PublicName()
	data := skaterView(p)
	data["Title"] = name
	data["Path"] = wantPath
	data["OGTitle"] = name
	data["OGDesc"] = article.Excerpt(p.Bio, "")
	if strings.TrimSpace(p.Bio) != "" {
		data["BioHTML"] = template.HTML(article.Render(p.Bio))
	}
	if p.AvatarURL != "" {
		data["OGImage"] = p.AvatarURL
	}
	by, _ := yt.OwnerFilters(s.db)
	if v, err := yt.Get(s.db, p.FeaturedVideoID); err == nil && yt.Match(v, by[v.ChannelID]) {
		data["Featured"] = v
	}
	clips := yt.FilterOwned(userChannelVideos(s.db, p.UserID, 0), by)
	if feat, ok := data["Featured"].(yt.Video); ok {
		var rest []yt.Video
		for _, c := range clips {
			if c.ID != feat.ID {
				rest = append(rest, c)
			}
		}
		clips = rest
	}
	n, _ := strconv.Atoi(r.URL.Query().Get("n"))
	pn, _ := strconv.Atoi(r.URL.Query().Get("p"))
	data["Clips"] = videoPager(wantPath+"/"+p.Slug, n, pn, clips, data)
	if photos, err := skater.ListByProfile(s.db, p.ID); err == nil && len(photos) > 0 {
		data["Gallery"] = photos
		data["GalleryJSON"] = galleryJSON(photos)
		data["GalleryScript"] = "gallery-json"
	}
	if u != nil {
		data["CanEdit"] = skater.CanEdit(u.Role, u.ID, p.UserID)
	}
	s.render(w, r, "skater_detail.html", data)
}

func rosterURL(db *sql.DB, slug string) string {
	p, err := skater.Get(db, "slug", slug)
	if err != nil {
		return "/team/" + slug
	}
	return skater.RosterPath(p.Role) + "/" + slug
}

func (s *Server) adminSkaters(w http.ResponseWriter, r *http.Request) {
	u := UserFrom(r)
	if u == nil || u.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	list, err := skater.List(s.db)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.render(w, r, "admin_skaters.html", map[string]any{"Title": "FS Team", "Path": "/admin/skaters", "Skaters": list})
}

func (s *Server) adminSkaterEdit(w http.ResponseWriter, r *http.Request) {
	u := UserFrom(r)
	if u == nil || u.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/skaters", http.StatusSeeOther)
		return
	}
	p, err := skater.Get(s.db, "id", r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if st := r.FormValue("status"); st != "" {
		p.Status = st
	}
	if _, err := skater.Save(s.db, p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/admin/skaters", http.StatusSeeOther)
}

func (s *Server) adminSkaterDelete(w http.ResponseWriter, r *http.Request) {
	u := UserFrom(r)
	if u == nil || u.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	_ = skater.Delete(s.db, r.PathValue("id"))
	http.Redirect(w, r, "/admin/skaters", http.StatusSeeOther)
}

func (s *Server) dashboardProfile(w http.ResponseWriter, r *http.Request) {
	u, p, ok := s.ownProfile(w, r)
	if !ok {
		return
	}
	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, skater.PhotoMax+1<<20)
		if err := r.ParseMultipartForm(skater.PhotoMax); err != nil {
			http.Error(w, "too large", http.StatusRequestEntityTooLarge)
			return
		}
		next := formProfile(r, p)
		next, err := applyPhoto(r, next)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if _, err := skater.Save(s.db, next); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, profilePath(r.FormValue("tab")), http.StatusSeeOther)
		return
	}
	s.renderSkaterProfile(w, r, u, p, nil, false)
}

func (s *Server) profileFilter(w http.ResponseWriter, r *http.Request) {
	u, p, ok := s.ownProfile(w, r)
	if !ok {
		return
	}
	_ = r.ParseForm()
	spec := yt.FormSpec(r.Form["field"], r.Form["op"], r.Form["value"], r.Form["kind"])
	switch r.FormValue("action") {
	case "clear":
		if err := auth.SetVideoFilter(s.db, u.ID, ""); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, profilePath("youtube"), http.StatusSeeOther)
	case "save":
		if err := auth.SetVideoFilter(s.db, u.ID, yt.EncodeSpec(spec)); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, profilePath("youtube"), http.StatusSeeOther)
	case "add":
		if len(spec.Rules) < yt.MaxRules {
			spec.Rules = append(spec.Rules, yt.Rule{})
		}
		s.renderSkaterProfile(w, r, u, p, &spec, false)
	default:
		s.renderSkaterProfile(w, r, u, p, &spec, true)
	}
}

type clipTest struct {
	yt.Video
	Keep bool
}

func (s *Server) ownProfile(w http.ResponseWriter, r *http.Request) (*auth.User, skater.Profile, bool) {
	u := UserFrom(r)
	if u == nil || !auth.HasPublicRoster(u.Role) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return nil, skater.Profile{}, false
	}
	p, err := skater.Get(s.db, "user_id", u.ID)
	if err == sql.ErrNoRows {
		name := u.Username
		if name == "" {
			name = u.DisplayName
		}
		p, err = skater.EnsureForUser(s.db, u.ID, name)
	}
	if err == sql.ErrNoRows {
		http.Error(w, "no profile linked", http.StatusNotFound)
		return nil, skater.Profile{}, false
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return nil, skater.Profile{}, false
	}
	return u, p, true
}

func (s *Server) renderSkaterProfile(w http.ResponseWriter, r *http.Request, u *auth.User, p skater.Profile, spec *yt.Spec, test bool) {
	var sspec yt.Spec
	if spec != nil {
		sspec = *spec
	} else {
		raw, _ := auth.GetVideoFilter(s.db, u.ID)
		sspec = yt.ParseSpec(raw)
	}
	sspec.Rules = yt.WithBlank(sspec.Rules)
	kindOn := map[string]bool{}
	for _, k := range sspec.Kinds {
		kindOn[k] = true
	}
	vids := userChannelVideos(s.db, u.ID, 50)
	defName := providerName(u)
	tab := "photo"
	if spec != nil {
		tab = "youtube"
	}
	data := map[string]any{
		"Title": "Profile", "Path": "/dashboard/profile", "P": p, "Action": "/dashboard/profile", "BioEdit": true,
		"Videos": vids, "FilterRows": sspec.Rules, "KindOn": kindOn, "Test": test, "Tab": tab,
		"DefaultName": defName, "DefaultSlug": skater.Slugify(defName), "SlugPrefix": skater.RosterPath(u.Role) + "/",
	}
	if test {
		check := yt.NormalizeSpec(sspec)
		var keep, hide []clipTest
		for _, v := range vids {
			item := clipTest{Video: v, Keep: yt.Match(v, check)}
			if item.Keep {
				keep = append(keep, item)
			} else {
				hide = append(hide, item)
			}
		}
		data["TestClips"] = append(keep, hide...)
	}
	s.render(w, r, "skater_form.html", data)
}

func profilePath(tab string) string {
	switch tab {
	case "profile", "youtube":
		return "/dashboard/profile#" + tab
	default:
		return "/dashboard/profile#photo"
	}
}

func providerName(u *auth.User) string {
	if u.DiscordID != "" {
		if s := strings.TrimSpace(u.DisplayName); s != "" {
			return s
		}
		return strings.TrimSpace(u.Username)
	}
	if s := strings.TrimSpace(u.YouTubeChannelTitle); s != "" {
		return s
	}
	if s := strings.TrimSpace(u.DisplayName); s != "" {
		return s
	}
	return strings.TrimSpace(u.Username)
}
