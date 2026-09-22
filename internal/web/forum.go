package web

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"seshhub/internal/article"
	"seshhub/internal/auth"
	"seshhub/internal/forum"
	"seshhub/internal/skater"
)

func (s *Server) requireLogin(w http.ResponseWriter, r *http.Request) *auth.User {
	u := UserFrom(r)
	if u == nil {
		loc := "/login"
		if n := safeNext(r.URL.Path); n != "" {
			loc = "/login?next=" + url.QueryEscape(n)
		}
		http.Redirect(w, r, loc, http.StatusFound)
		return nil
	}
	return u
}

func (s *Server) forumIncoming(w http.ResponseWriter, r *http.Request) (body, parent string, files []forum.FileIn, remove []string, ok bool) {
	r.Body = http.MaxBytesReader(w, r.Body, int64(skater.PhotoMax)*forum.PhotoMax+1<<20)
	ct := r.Header.Get("Content-Type")
	var err error
	if strings.HasPrefix(ct, "multipart/") {
		err = r.ParseMultipartForm(int64(skater.PhotoMax) * forum.PhotoMax)
	} else {
		err = r.ParseForm()
	}
	if err != nil {
		http.Error(w, "too large", http.StatusRequestEntityTooLarge)
		return "", "", nil, nil, false
	}
	body = r.FormValue("body")
	parent = strings.TrimSpace(r.FormValue("parent_id"))
	remove = r.Form["remove_photo"]
	if r.MultipartForm != nil {
		for _, fh := range r.MultipartForm.File["photos"] {
			f, err := fh.Open()
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return "", "", nil, nil, false
			}
			raw, err := io.ReadAll(io.LimitReader(f, int64(skater.PhotoMax)+1))
			_ = f.Close()
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return "", "", nil, nil, false
			}
			files = append(files, forum.FileIn{R: bytes.NewReader(raw)})
		}
	}
	if len(files) > forum.PhotoMax {
		http.Error(w, "max 3 images", http.StatusBadRequest)
		return "", "", nil, nil, false
	}
	return body, parent, files, remove, true
}

func (s *Server) forumIndex(w http.ResponseWriter, r *http.Request) {
	u := s.requireLogin(w, r)
	if u == nil {
		return
	}
	list, err := forum.ListSections(s.db, true)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.render(w, r, "forum_index.html", map[string]any{
		"Title": "Forum", "Path": "/forum", "Sections": list, "Admin": u.Role == auth.RoleAdmin,
		"ForumJS": true,
	})
}

func (s *Server) forumMentions(w http.ResponseWriter, r *http.Request) {
	u := s.requireLogin(w, r)
	if u == nil {
		return
	}
	list, err := forum.ListMentions(s.db, u.ID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	_ = forum.MarkMentionsRead(s.db, u.ID)
	type row struct {
		forum.MentionItem
		Href string
	}
	views := make([]row, len(list))
	for i, it := range list {
		views[i] = row{MentionItem: it, Href: forum.MentionHref(it, 24)}
	}
	s.render(w, r, "forum_mentions.html", map[string]any{
		"Title": "Mentions", "Path": "/forum/mentions", "Items": views, "ForumJS": true,
	})
}

func (s *Server) forumUsers(w http.ResponseWriter, r *http.Request) {
	u := s.requireLogin(w, r)
	if u == nil {
		return
	}
	hits, err := forum.SearchUsers(s.db, r.URL.Query().Get("q"), 8)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if hits == nil {
		hits = []forum.UserHit{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(hits)
}

func (s *Server) mediaForum(w http.ResponseWriter, r *http.Request) {
	if s.requireLogin(w, r) == nil {
		return
	}
	path, ok := forum.PhotoFile(r.PathValue("file"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	serveMedia(w, r, path)
}

func (s *Server) forumSection(w http.ResponseWriter, r *http.Request) {
	u := s.requireLogin(w, r)
	if u == nil {
		return
	}
	sec, err := forum.GetSection(s.db, r.PathValue("sectionSlug"))
	if err == sql.ErrNoRows || (err == nil && !sec.Active && u.Role != auth.RoleAdmin) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	list, err := forum.ListThreads(s.db, sec.ID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	n, _ := strconv.Atoi(r.URL.Query().Get("n"))
	p, _ := strconv.Atoi(r.URL.Query().Get("p"))
	per, page, offset, from, to := videoPage(n, p, len(list))
	shown := list
	if len(list) > 0 {
		end := offset + per
		if end > len(list) {
			end = len(list)
		}
		shown = list[offset:end]
	}
	s.render(w, r, "forum_section.html", map[string]any{
		"Title": sec.Name, "Path": "/forum", "S": sec, "Threads": shown, "Admin": u.Role == auth.RoleAdmin,
		"PagerLabel": "Thread pagination", "PagerBase": "/forum/" + sec.Slug,
		"Per": per, "Page": page, "Total": len(list), "From": from, "To": to,
		"Prev": page - 1, "Next": page + 1, "HasPrev": page > 1, "HasNext": to < len(list),
		"ForumJS": true,
	})
}

func (s *Server) forumNew(w http.ResponseWriter, r *http.Request) {
	u := s.requireLogin(w, r)
	if u == nil {
		return
	}
	sec, err := forum.GetSection(s.db, r.PathValue("sectionSlug"))
	if err == sql.ErrNoRows || (err == nil && !sec.Active && u.Role != auth.RoleAdmin) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if r.Method == http.MethodGet {
		s.render(w, r, "forum_form.html", map[string]any{
			"Title": "New thread", "Path": "/forum", "S": sec, "TitleVal": "", "Body": "", "ForumJS": true,
		})
		return
	}
	body, _, files, _, ok := s.forumIncoming(w, r)
	if !ok {
		return
	}
	th, err := forum.CreateThread(s.db, sec.ID, u.ID, r.FormValue("title"), body, files)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.announce("forum", "Forum: "+display(u)+" started "+th.Title+" "+s.publicURL("/forum/"+sec.Slug+"/"+th.Slug))
	http.Redirect(w, r, "/forum/"+sec.Slug+"/"+th.Slug, http.StatusSeeOther)
}

type forumPostView struct {
	forum.Post
	HTML      template.HTML
	CanEdit   bool
	CanDelete bool
	Latest    bool
	Quote     string
}

func (s *Server) forumThread(w http.ResponseWriter, r *http.Request) {
	u := s.requireLogin(w, r)
	if u == nil {
		return
	}
	sec, th, ok := s.forumLoad(w, r, u)
	if !ok {
		return
	}
	if r.Method == http.MethodPost {
		if !forum.CanEdit(u.ID, th.UserID) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		slug, err := forum.UpdateThreadTitle(s.db, th, r.FormValue("title"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, "/forum/"+sec.Slug+"/"+slug, http.StatusSeeOther)
		return
	}
	posts, err := forum.ListPosts(s.db, th.ID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	_ = forum.MarkRead(s.db, u.ID, th.ID)
	n, _ := strconv.Atoi(r.URL.Query().Get("n"))
	p, _ := strconv.Atoi(r.URL.Query().Get("p"))
	per, page, offset, from, to := videoPage(n, p, len(posts))
	shown := posts
	if len(posts) > 0 {
		end := offset + per
		if end > len(posts) {
			end = len(posts)
		}
		shown = posts[offset:end]
	}
	views := make([]forumPostView, len(shown))
	for i, post := range shown {
		views[i] = forumPostView{Post: post, HTML: template.HTML(post.BodyHTML), CanEdit: forum.CanEdit(u.ID, post.UserID), CanDelete: forum.CanDelete(u.Role, u.ID, post.UserID), Latest: i == len(shown)-1 && to >= len(posts), Quote: article.Excerpt(post.BodyRaw, "")}
	}
	s.render(w, r, "forum_thread.html", map[string]any{
		"Title": th.Title, "Path": "/forum", "S": sec, "T": th, "Posts": views,
		"CanEditThread": forum.CanEdit(u.ID, th.UserID),
		"PagerLabel":    "Post pagination", "PagerBase": "/forum/" + sec.Slug + "/" + th.Slug,
		"Per": per, "Page": page, "Total": len(posts), "From": from, "To": to,
		"Prev": page - 1, "Next": page + 1, "HasPrev": page > 1, "HasNext": to < len(posts),
		"ForumJS": true,
	})
}

func (s *Server) forumReply(w http.ResponseWriter, r *http.Request) {
	u := s.requireLogin(w, r)
	if u == nil {
		return
	}
	sec, th, ok := s.forumLoad(w, r, u)
	if !ok {
		return
	}
	body, parent, files, _, ok := s.forumIncoming(w, r)
	if !ok {
		return
	}
	if err := forum.Reply(s.db, th.ID, u.ID, body, parent, files); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/forum/"+sec.Slug+"/"+th.Slug+"#latest", http.StatusSeeOther)
}

func (s *Server) forumPostEdit(w http.ResponseWriter, r *http.Request) {
	u := s.requireLogin(w, r)
	if u == nil {
		return
	}
	sec, th, ok := s.forumLoad(w, r, u)
	if !ok {
		return
	}
	p, err := forum.GetPost(s.db, r.PathValue("id"))
	if err == sql.ErrNoRows || p.ThreadID != th.ID {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if !forum.CanEdit(u.ID, p.UserID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	body, _, files, remove, ok := s.forumIncoming(w, r)
	if !ok {
		return
	}
	if err := forum.UpdatePost(s.db, p.ID, body, remove, files); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/forum/"+sec.Slug+"/"+th.Slug+"#p-"+p.ID, http.StatusSeeOther)
}

func (s *Server) forumPostDelete(w http.ResponseWriter, r *http.Request) {
	u := s.requireLogin(w, r)
	if u == nil {
		return
	}
	sec, th, ok := s.forumLoad(w, r, u)
	if !ok {
		return
	}
	p, err := forum.GetPost(s.db, r.PathValue("id"))
	if err == sql.ErrNoRows || p.ThreadID != th.ID {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if !forum.CanDelete(u.Role, u.ID, p.UserID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	gone, err := forum.DeletePost(s.db, p)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if gone {
		http.Redirect(w, r, "/forum/"+sec.Slug, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/forum/"+sec.Slug+"/"+th.Slug, http.StatusSeeOther)
}

func (s *Server) forumLoad(w http.ResponseWriter, r *http.Request, u *auth.User) (forum.Section, forum.Thread, bool) {
	sec, err := forum.GetSection(s.db, r.PathValue("sectionSlug"))
	if err == sql.ErrNoRows || (err == nil && !sec.Active && u.Role != auth.RoleAdmin) {
		http.NotFound(w, r)
		return forum.Section{}, forum.Thread{}, false
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return forum.Section{}, forum.Thread{}, false
	}
	th, err := forum.GetThread(s.db, sec.ID, r.PathValue("threadSlug"))
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return forum.Section{}, forum.Thread{}, false
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return forum.Section{}, forum.Thread{}, false
	}
	return sec, th, true
}

func (s *Server) adminForum(w http.ResponseWriter, r *http.Request) {
	u := UserFrom(r)
	if u == nil || u.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if r.Method == http.MethodPost {
		var err error
		switch r.FormValue("action") {
		case "create":
			err = forum.CreateSection(s.db, r.FormValue("name"), r.FormValue("description"))
		case "save":
			err = forum.SaveSection(s.db, r.FormValue("id"), r.FormValue("name"), r.FormValue("description"), r.FormValue("is_active") == "1")
		case "up":
			err = forum.MoveSection(s.db, r.FormValue("id"), -1)
		case "down":
			err = forum.MoveSection(s.db, r.FormValue("id"), 1)
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, "/admin/forum/sections", http.StatusSeeOther)
		return
	}
	list, err := forum.ListSections(s.db, false)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.render(w, r, "admin_forum.html", map[string]any{"Title": "Forum sections", "Path": "/admin/forum/sections", "Sections": list})
}

func (s *Server) adminForumDelete(w http.ResponseWriter, r *http.Request) {
	u := UserFrom(r)
	if u == nil || u.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := forum.DeleteSection(s.db, r.PathValue("id")); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/forum/sections", http.StatusSeeOther)
}
