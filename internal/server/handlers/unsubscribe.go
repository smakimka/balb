package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/render"

	"github.com/smakimka/balb/internal/model"
	"github.com/smakimka/balb/internal/server/storage"
)

type UnsubscribeHandler struct {
	s storage.Storage
}

func NewUnsubscribeHandler(s storage.Storage) UnsubscribeHandler {
	return UnsubscribeHandler{s: s}
}

func (h UnsubscribeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data := &model.SubscriptionData{}
	if err := render.Bind(r, data); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, model.Response{Msg: "wrong json"})
		return
	}

	err := h.s.Unsubscribe(r.Context(), data)
	if err != nil {
		if errors.Is(err, storage.ErrSubscriptionNotFound) {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, model.Response{Msg: "subscription not found"})
			return
		}

		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, model.Response{Msg: "internal server error"})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, model.Response{})
}
