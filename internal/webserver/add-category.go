package webserver

import (
	"encoding/json"
	"net/http"
	"regexp"

	log "github.com/sirupsen/logrus"
)

type addCategoryRequest struct {
	Name  string `json:"name"`
	Title string `json:"title"`
}

var categoryNameRE = regexp.MustCompile(`^[a-z][a-z0-9-]+$`)

func (ws *webserver) addCategory(w http.ResponseWriter, r *http.Request) {
	var req addCategoryRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !categoryNameRE.MatchString(req.Name) {
		m := "Tried to create category with invalid name [" + req.Name + "]"
		log.Error(m)
		http.Error(w, m, http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		m := "Tried to create category with empty title"
		log.Error(m)
		http.Error(w, m, http.StatusBadRequest)
		return
	}

	c, err := ws.db.InsertNewCategory(req.Name, req.Title)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err = json.NewEncoder(w).Encode(c); err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
