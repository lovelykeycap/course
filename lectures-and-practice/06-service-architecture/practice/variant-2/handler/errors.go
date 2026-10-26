package handler

import (
	"errors"
	"net/http"

	"github.com/course-go-autumn-2026/practice6-variant2-booking/domain"
)

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		w.WriteHeader(http.StatusNotFound)
	case errors.Is(err, domain.ErrInvalidStart):
		http.Error(w, domain.ErrInvalidStart.Error(), http.StatusBadRequest)
	case errors.Is(err, domain.ErrInvalidEnd):
		http.Error(w, domain.ErrInvalidEnd.Error(), http.StatusBadRequest)
	case errors.Is(err, domain.ErrInvalidInterval):
		http.Error(w, domain.ErrInvalidInterval.Error(), http.StatusBadRequest)
	case errors.Is(err, domain.ErrAlreadyBooked):
		http.Error(w, domain.ErrAlreadyBooked.Error(), http.StatusConflict)
	case errors.Is(err, domain.ErrCancelled):
		http.Error(w, domain.ErrCancelled.Error(), http.StatusConflict)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
