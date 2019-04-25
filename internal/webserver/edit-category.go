package webserver

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/awused/aw-rss/internal/structs"
	"github.com/go-chi/chi"
	log "github.com/sirupsen/logrus"
)

type editCategoryRequest struct {
	Edit structs.CategoryEdit `json:"edit"`
}

func (ws *webserver) editCategory(w http.ResponseWriter, r *http.Request) {
	var req editCategoryRequest

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mutate, err := structs.CategoryApplyEdit(req.Edit)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	f, err := ws.db.MutateCategory(int64(id), mutate)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err = json.NewEncoder(w).Encode(f); err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
