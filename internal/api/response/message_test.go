package response

import (
	"github.com/bitly/go-simplejson"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithMessage(t *testing.T) {
	resp := httptest.NewRecorder()
	want := "oops"
	WithMessage(resp, http.StatusInternalServerError, want)

	jsn, err := simplejson.NewFromReader(resp.Body)
	if err != nil {
		t.Error(err)
	}

	if resp.Code != http.StatusInternalServerError {
		t.Errorf("WithMessage StatusCode want %v - got %v", http.StatusInternalServerError, resp.Code)
	}

	got := jsn.GetPath("Message").MustString()
	if want != got {
		t.Errorf("WithMessage want %v - got %v", want, got)
	}
}