package response

import (
	"encoding/json"
	"net/http"
)

func WithMessage(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(struct {
		Message string
	}{
		Message: message,
	})
}
