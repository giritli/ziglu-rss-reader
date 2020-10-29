package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reader/internal/feed"
	"reader/internal/storage"
	"reflect"
	"testing"
	"time"
)

func Test_timeOffsetFromRequest(t *testing.T) {
	want := "2010-01-02T12:13:14"
	r, _ := http.NewRequest("GET", "?offset="+want, nil)
	tm := timeOffsetFromRequest(r)

	if got := tm.Format(OffsetTimeFormat); got != want {
		t.Errorf("timeOffsetFromRequest want %v got %v", want, got)
	}
}

func timeFromString(t *testing.T, s string) time.Time {
	if tm, err := time.Parse(OffsetTimeFormat, s); err != nil {
		t.Errorf("could not parse time: %v", err)
		return time.Time{}
	} else {
		return tm
	}
}

func newTestAPI(t *testing.T, latest uint) http.Handler {
	s := storage.NewInMemoryStorage(latest)

	err := s.Store(&feed.Feed{
		FeedLink: &url.URL{
			Scheme: "https",
			Host:   "mock.local",
		},
		ModifiedAt: time.Time{},
		Title:      "Mock Feed",
		Link:       "https://mock.local",
	}, []*feed.Article{
		{
			GUID:        "",
			Link:        "https://mock.local/article/1",
			Published:   timeFromString(t, "2010-01-01T01:01:01"),
			Title:       "Article 1",
			Description: "This is the first article",
			Image:       nil,
		},
		{
			GUID:        "",
			Link:        "https://mock.local/article/2",
			Published:   timeFromString(t, "2020-01-01T01:01:01"),
			Title:       "Article 2",
			Description: "This is the second article",
			Image:       nil,
		},
	})

	if err != nil {
		t.Errorf("error occurred creating mock storage: %v", err)
	}

	err = s.Store(&feed.Feed{
		FeedLink: &url.URL{
			Scheme: "https",
			Host:   "mock2.local",
		},
		ModifiedAt: time.Time{},
		Title:      "Mock Feed 2",
		Link:       "https://mock2.local",
	}, []*feed.Article{
		{
			GUID:        "",
			Link:        "https://mock2.local/article/1",
			Published:   timeFromString(t, "2010-01-01T01:01:02"),
			Title:       "Article 1",
			Description: "This is the first article in second feed",
			Image:       nil,
		},
		{
			GUID:        "",
			Link:        "https://mock2.local/article/2",
			Published:   timeFromString(t, "2020-01-01T01:01:02"),
			Title:       "Article 2",
			Description: "This is the second article in second feed",
			Image:       nil,
		},
	})

	if err != nil {
		t.Errorf("error occurred creating mock storage: %v", err)
	}

	return NewAPI(s)
}

func TestAPI_Endpoints(t *testing.T) {
	tests := []struct {
		name    string
		request *http.Request
		latest  uint
		code    int
		body    interface{}
	}{
		{
			"getting feeds",
			&http.Request{
				Method: "GET",
				URL: &url.URL{
					Path: "/feeds",
				},
			},
			2,
			http.StatusOK,
			[]*feed.Feed{
				{
					FeedLink:   &url.URL{
						Scheme: "https",
						Host: "mock.local",
					},
					ModifiedAt: time.Time{},
					Title:      "Mock Feed",
					Link:       "https://mock.local",
				},
				{
					FeedLink:   &url.URL{
						Scheme: "https",
						Host: "mock2.local",
					},
					ModifiedAt: time.Time{},
					Title:      "Mock Feed 2",
					Link:       "https://mock2.local",
				},
			},
		},
		{
			"getting latest",
			&http.Request{
				Method: "GET",
				URL: &url.URL{
					Path: "/latest",
				},
			},
			4,
			http.StatusOK,
			[]*feed.Article{
				{
					GUID:        "",
					Link:        "https://mock2.local/article/2",
					Published:   timeFromString(t, "2020-01-01T01:01:02"),
					Title:       "Article 2",
					Description: "This is the second article in second feed",
					Image:       nil,
				},
				{
					GUID:        "",
					Link:        "https://mock.local/article/2",
					Published:   timeFromString(t, "2020-01-01T01:01:01"),
					Title:       "Article 2",
					Description: "This is the second article",
					Image:       nil,
				},
				{
					GUID:        "",
					Link:        "https://mock2.local/article/1",
					Published:   timeFromString(t, "2010-01-01T01:01:02"),
					Title:       "Article 1",
					Description: "This is the first article in second feed",
					Image:       nil,
				},
				{
					GUID:        "",
					Link:        "https://mock.local/article/1",
					Published:   timeFromString(t, "2010-01-01T01:01:01"),
					Title:       "Article 1",
					Description: "This is the first article",
					Image:       nil,
				},
			},
		},
		{
			"getting latest with offset",
			&http.Request{
				Method: "GET",
				URL: &url.URL{
					Path:     "/latest",
					RawQuery: "offset=2015-01-01T01:01:01",
				},
			},
			4,
			http.StatusOK,
			[]*feed.Article{
				{
					GUID:        "",
					Link:        "https://mock2.local/article/1",
					Published:   timeFromString(t, "2010-01-01T01:01:02"),
					Title:       "Article 1",
					Description: "This is the first article in second feed",
					Image:       nil,
				},
				{
					GUID:        "",
					Link:        "https://mock.local/article/1",
					Published:   timeFromString(t, "2010-01-01T01:01:01"),
					Title:       "Article 1",
					Description: "This is the first article",
					Image:       nil,
				},
			},
		},
		{
			"getting latest with offset on exact time",
			&http.Request{
				Method: "GET",
				URL: &url.URL{
					Path:     "/latest",
					RawQuery: "offset=2010-01-01T01:01:02",
				},
			},
			4,
			http.StatusOK,
			[]*feed.Article{
				{
					GUID:        "",
					Link:        "https://mock.local/article/1",
					Published:   timeFromString(t, "2010-01-01T01:01:01"),
					Title:       "Article 1",
					Description: "This is the first article",
					Image:       nil,
				},
			},
		},
		{
			"getting latest with 1 article",
			&http.Request{
				Method: "GET",
				URL: &url.URL{
					Path: "/latest",
				},
			},
			1,
			http.StatusOK,
			[]*feed.Article{
				{
					GUID:        "",
					Link:        "https://mock2.local/article/2",
					Published:   timeFromString(t, "2020-01-01T01:01:02"),
					Title:       "Article 2",
					Description: "This is the second article in second feed",
					Image:       nil,
				},
			},
		},
		{
			"getting latest from specific feed",
			&http.Request{
				Method: "GET",
				URL: &url.URL{
					Path: "/latest/" + feed.UUIDFromString("https://mock2.local").String(),
				},
			},
			4,
			http.StatusOK,
			[]*feed.Article{
				{
					GUID:        "",
					Link:        "https://mock2.local/article/2",
					Published:   timeFromString(t, "2020-01-01T01:01:02"),
					Title:       "Article 2",
					Description: "This is the second article in second feed",
					Image:       nil,
				},
				{
					GUID:        "",
					Link:        "https://mock2.local/article/1",
					Published:   timeFromString(t, "2010-01-01T01:01:02"),
					Title:       "Article 1",
					Description: "This is the first article in second feed",
					Image:       nil,
				},
			},
		},
		{
			"getting specific article",
			&http.Request{
				Method: "GET",
				URL: &url.URL{
					Path: "/article/" + feed.UUIDFromString("https://mock2.local/article/2").String(),
				},
			},
			4,
			http.StatusOK,
			&feed.Article{
				GUID:        "",
				Link:        "https://mock2.local/article/2",
				Published:   timeFromString(t, "2020-01-01T01:01:02"),
				Title:       "Article 2",
				Description: "This is the second article in second feed",
				Image:       nil,
			},
		},
		{
			"getting non-existant article",
			&http.Request{
				Method: "GET",
				URL: &url.URL{
					Path: "/article/" + feed.UUIDFromString("oops").String(),
				},
			},
			4,
			http.StatusInternalServerError,
			map[string]string{
				"Message": "could not retrieve article",
			},
		},
		{
			"getting latest from non-existant feed",
			&http.Request{
				Method: "GET",
				URL: &url.URL{
					Path: "/latest/" + feed.UUIDFromString("oops").String(),
				},
			},
			4,
			http.StatusInternalServerError,
			map[string]string{
				"Message": "could not retrieve latest articles from feed",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestAPI(t, tt.latest)
			resp := httptest.NewRecorder()
			h.ServeHTTP(resp, tt.request)

			if resp.Code != tt.code {
				t.Errorf("StatusCode want %v got %v", tt.code, resp.Code)
			}

			var b interface{}
			if err := json.NewDecoder(resp.Body).Decode(&b); err != nil {
				t.Errorf("could not decode response body: %v", err)
			}

			var wb interface{}
			jsn, err := json.Marshal(tt.body)
			if err != nil {
				t.Errorf("could not marshal want body: %v", err)
			}

			if err := json.Unmarshal(jsn, &wb); err != nil {
				t.Errorf("could not unmarshal want body: %v", err)
			}

			if !reflect.DeepEqual(wb, b) {
				t.Errorf("Response Body want %v got %v", wb, b)
			}
		})
	}
}
