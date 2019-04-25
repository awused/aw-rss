package webserver

import (
	"encoding/json"
	"net/http"

	"github.com/awused/aw-rss/internal/database"
	"github.com/awused/aw-rss/internal/structs"
	log "github.com/sirupsen/logrus"
)

func (ws *webserver) addCategory(w http.ResponseWriter, r *http.Request) {
	var req database.AddCategoryRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !structs.CategoryNameRE.MatchString(req.Name) {
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

	c, err := ws.db.InsertNewCategory(req)
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
