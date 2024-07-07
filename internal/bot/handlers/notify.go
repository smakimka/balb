package handlers

import (
	"net/http"

	"github.com/go-chi/render"
	"github.com/smakimka/balb/internal/bot/storage"
	"github.com/smakimka/balb/internal/model"
)

type NotifyHandler struct {
	s storage.Storage
}

func NewNotifyHandler(s storage.Storage) NotifyHandler {
	return NotifyHandler{s: s}
}

func (h NotifyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data := &model.NotifyRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, model.Response{Msg: "wrong json"})
		return
	}

	if err := h.s.CreateBirthday(r.Context(), data); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, model.Response{Msg: "internal server error"})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, model.Response{})
}
