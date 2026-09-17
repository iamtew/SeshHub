package web

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"

	"seshhub/internal/skater"
)

func galleryJSON(photos []skater.Photo) template.JS {
	if len(photos) == 0 {
		return "[]"
	}
	b, err := json.Marshal(photos)
	if err != nil {
		return "[]"
	}
	return template.JS(b)
}

func (s *Server) mediaGallery(w http.ResponseWriter, r *http.Request) {
	path, ok := skater.GalleryFile(r.PathValue("file"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	serveMedia(w, r, path)
}

func (s *Server) photos(w http.ResponseWriter, r *http.Request) {
	list, err := skater.ListPublicGallery(s.db)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	n, _ := strconv.Atoi(r.URL.Query().Get("n"))
	p, _ := strconv.Atoi(r.URL.Query().Get("p"))
	data := map[string]any{
		"Title": "Photos", "Path": "/photos", "PagerLabel": "Photos pagination",
		"Gallery": list, "GalleryJSON": galleryJSON(list), "GalleryScript": "gallery-json",
		"PhotosLive": true,
	}
	per, page, offset, from, to := videoPage(n, p, len(list))
	var pageItems []skater.Photo
	if len(list) > 0 {
		end := offset + per
		if end > len(list) {
			end = len(list)
		}
		pageItems = list[offset:end]
	}
	data["Photos"] = pageItems
	data["PagerBase"] = "/photos"
	data["Per"], data["Page"], data["Total"] = per, page, len(list)
	data["From"], data["To"] = from, to
	data["Prev"], data["Next"] = page-1, page+1
	data["HasPrev"], data["HasNext"] = page > 1, to < len(list)
	s.render(w, r, "photos_list.html", data)
}

func (s *Server) dashboardGallery(w http.ResponseWriter, r *http.Request) {
	_, p, ok := s.ownProfile(w, r)
	if !ok {
		return
	}
	photos, err := skater.ListByProfile(s.db, p.ID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.render(w, r, "gallery_form.html", map[string]any{"Title": "Gallery", "Path": "/dashboard/gallery", "Gallery": photos})
}

func (s *Server) dashboardGalleryAdd(w http.ResponseWriter, r *http.Request) {
	_, p, ok := s.ownProfile(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, skater.PhotoMax+1<<20)
	if err := r.ParseMultipartForm(skater.PhotoMax); err != nil {
		http.Error(w, "too large", http.StatusRequestEntityTooLarge)
		return
	}
	f, hdr, err := r.FormFile("photo")
	if err != nil || hdr.Size == 0 {
		http.Error(w, "need a photo", http.StatusBadRequest)
		return
	}
	defer f.Close()
	if _, err := skater.AddGallery(s.db, p.ID, f); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/dashboard/gallery", http.StatusSeeOther)
}

func (s *Server) dashboardGalleryDelete(w http.ResponseWriter, r *http.Request) {
	_, p, ok := s.ownProfile(w, r)
	if !ok {
		return
	}
	if err := skater.DeleteGallery(s.db, p.ID, r.PathValue("id")); err != nil {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/dashboard/gallery", http.StatusSeeOther)
}

func (s *Server) dashboardGalleryOrder(w http.ResponseWriter, r *http.Request) {
	_, p, ok := s.ownProfile(w, r)
	if !ok {
		return
	}
	_ = r.ParseForm()
	if err := skater.ReorderGallery(s.db, p.ID, r.Form["id"]); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
