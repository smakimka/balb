package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/render"

	"github.com/smakimka/balb/internal/model"
	"github.com/smakimka/balb/internal/server/storage"
)

type SubscribeHandler struct {
	s storage.Storage
}

func NewSubscribeHandler(s storage.Storage) SubscribeHandler {
	return SubscribeHandler{s: s}
}

func (h SubscribeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data := &model.SubscriptionData{}
	if err := render.Bind(r, data); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, model.Response{Msg: "wrong json"})
		return
	}

	err := h.s.Subscribe(r.Context(), data)
	if err != nil {
		if errors.Is(err, storage.ErrSubscriptionAlreadyExists) {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, model.Response{Msg: "subscription already exists"})
			return
		}

		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, model.Response{Msg: "internal server error"})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, model.Response{})
}
