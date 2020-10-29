package reader

import (
	"context"
	"github.com/mmcdole/gofeed"
	"io/ioutil"
	"net/http"
	"net/url"
	"reader/internal/feed"
	"reader/internal/storage"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestNewReader(t *testing.T) {
	type args struct {
		s       storage.Storage
		options []Option
	}
	tests := []struct {
		name string
		args args
		want *Reader
	}{
		{
			"default reader",
			args{
				storage.NewInMemoryStorage(1),
				nil,
			},
			&Reader{
				s:                storage.NewInMemoryStorage(1),
				p:                gofeed.NewParser(),
				c:                &http.Client{},
				workers:          8,
				retry:            60 * time.Second,
				retryNotModified: 120 * time.Second,
				retryAfterError:  300 * time.Second,
			},
		},
		{
			"with custom options",
			args{
				storage.NewInMemoryStorage(3),
				[]Option{
					WithWorkers(10),
					WithRetryDuration(40 * time.Second),
					WithRetryNotModifiedDuration(80 * time.Second),
					WithRetryAfterErrorDuration(120 * time.Second),
				},
			},
			&Reader{
				s:                storage.NewInMemoryStorage(3),
				p:                gofeed.NewParser(),
				c:                &http.Client{},
				workers:          10,
				retry:            40 * time.Second,
				retryNotModified: 80 * time.Second,
				retryAfterError:  120 * time.Second,
			},
		},
		{
			"with 0 workers",
			args{
				storage.NewInMemoryStorage(1),
				[]Option{
					WithWorkers(0),
				},
			},
			&Reader{
				s:                storage.NewInMemoryStorage(1),
				p:                gofeed.NewParser(),
				c:                &http.Client{},
				workers:          1,
				retry:            60 * time.Second,
				retryNotModified: 120 * time.Second,
				retryAfterError:  300 * time.Second,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewReader(tt.args.s, tt.args.options...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewReader() = %v, want %v", got, tt.want)
			}
		})
	}
}

type rtf func(r *http.Request) *http.Response

func (f rtf) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r), nil
}

func TestReader_getFeedContent(t *testing.T) {
	type fields struct {
		s    storage.Storage
		opts []Option
	}
	type args struct {
		ctx context.Context
		f   *feed.Feed
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		feedHeaders  map[string][]string
		feedXML      string
		wantFeed     *feed.Feed
		wantArticles []*feed.Article
		wantTimeout  time.Duration
		wantErr      bool
	}{
		{
			"successful feed",
			fields{
				nil,
				nil,
			},
			args{
				context.Background(),
				&feed.Feed{
					FeedLink: &url.URL{
						Scheme: "http",
						Host:   "rss.local",
					},
				},
			},
			nil,
			`<?xml version="1.0" encoding="UTF-8" ?>
<rss version="2.0">
<channel>
 <title>W3Schools Home Page</title>
 <link>https://www.w3schools.com</link>
 <description>Free web building tutorials</description>
 <item>
   <title>RSS Tutorial</title>
   <link>https://www.w3schools.com/xml/xml_rss.asp</link>
   <description>New RSS tutorial on W3Schools</description>
 </item>
 <item>
   <title>XML Tutorial</title>
   <link>https://www.w3schools.com/xml</link>
   <description>New XML tutorial on W3Schools</description>
   <media:thumbnail url="http://image"></media>
 </item>
</channel>

</rss>
`,
			&feed.Feed{
				FeedLink: &url.URL{
					Scheme: "http",
					Host:   "rss.local",
				},
				Title: "W3Schools Home Page",
				Link:  "https://www.w3schools.com",
			},
			[]*feed.Article{
				{
					Published:   time.Time{},
					Title:       "RSS Tutorial",
					Description: "New RSS tutorial on W3Schools",
					Link:        "https://www.w3schools.com/xml/xml_rss.asp",
					Image:       nil,
				},
				{
					Published:   time.Time{},
					Title:       "XML Tutorial",
					Description: "New XML tutorial on W3Schools",
					Link:        "https://www.w3schools.com/xml",
					Image: &feed.Image{
						Title: "Thumbnail",
						URL:   "http://image",
					},
				},
			},
			60 * time.Second,
			false,
		},
		{
			"invalid feed",
			fields{
				nil,
				nil,
			},
			args{
				context.Background(),
				&feed.Feed{
					FeedLink: &url.URL{
						Scheme: "http",
						Host:   "rss.local",
					},
				},
			},
			nil,
			`<bad feed>`,
			&feed.Feed{},
			[]*feed.Article{},
			300 * time.Second,
			true,
		},
		{
			"custom feed timeout",
			fields{
				nil,
				nil,
			},
			args{
				context.Background(),
				&feed.Feed{
					FeedLink: &url.URL{
						Scheme: "http",
						Host:   "rss.local",
					},
				},
			},
			map[string][]string{
				"Cache-Control": {"max-age=123"},
			},
			`<?xml version="1.0" encoding="UTF-8" ?>
<rss version="2.0">
<channel>
  <title>W3Schools Home Page</title>
  <link>https://www.w3schools.com</link>
  <description>Free web building tutorials</description>
</channel>

</rss>
`,
			&feed.Feed{
				FeedLink: &url.URL{
					Scheme: "http",
					Host:   "rss.local",
				},
				Title: "W3Schools Home Page",
				Link:  "https://www.w3schools.com",
			},
			[]*feed.Article{},
			123 * time.Second,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			c := WithHTTPClient(&http.Client{
				Transport: rtf(func(r *http.Request) *http.Response {
					return &http.Response{
						StatusCode: 200,
						Body:       ioutil.NopCloser(strings.NewReader(tt.feedXML)),
						Header:     tt.feedHeaders,
					}
				}),
			})

			r := NewReader(tt.fields.s, append(tt.fields.opts, c)...)
			cpf, err := r.getFeedContent(tt.args.ctx, tt.args.f)
			if (err != nil) != tt.wantErr {
				t.Errorf("getFeedContent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			f, a := r.mapParsedFeedToFeedAndArticles(cpf.f, tt.args.f)

			if !reflect.DeepEqual(f, tt.wantFeed) {
				t.Errorf("getFeedContent() got feed = %v, want %v", cpf, tt.wantFeed)
			}

			if !reflect.DeepEqual(a, tt.wantArticles) {
				t.Errorf("getFeedContent() got articles = %v, want %v", a, tt.wantArticles)
			}

			if !reflect.DeepEqual(cpf.d, tt.wantTimeout) {
				t.Errorf("getFeedContent() got timeout = %v, want %v", cpf.d, tt.wantTimeout)
			}
		})
	}
}

func TestReader_Update_ClosedWithContext(t *testing.T) {
	ctx, cf := context.WithCancel(context.Background())
	cf()

	s := storage.NewInMemoryStorage(10)

	err := s.Store(&feed.Feed{
		FeedLink: &url.URL{
			Scheme: "https",
			Host:   "nohost.local",
		},
	}, nil)
	if err != nil {
		t.Errorf("error occurred writing feed to storage: %v", err)
	}

	feeds, err := s.Feeds()
	if err != nil {
		t.Errorf("error occurred retrieving feeds from storage: %v", err)
	}

	r := NewReader(s)
	select {
	case err, ok := <-r.Update(ctx, feeds):
		if ok {
			t.Errorf("error returned from channel: %v", err)
		}
	case <-time.After(10 * time.Millisecond):
		t.Error("timeout reached unexpectedly")
	}
}

// This test is not the greatest test ever written as it relies
// on time elapsing. The test is testing that our worker model is
// working as intended and has the ability to concurrently load
// multiple feeds at the same time.
func TestReader_Update_ConcurrentWorkers(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8" ?>
<rss version="2.0">
<channel>
 <title>W3Schools Home Page</title>
 <link>https://www.w3schools.com</link>
 <description>Free web building tutorials</description>
 <item>
   <title>RSS Tutorial</title>
   <link>https://www.w3schools.com/xml/xml_rss.asp</link>
   <description>New RSS tutorial on W3Schools</description>
 </item>
 <item>
   <title>XML Tutorial</title>
   <link>https://www.w3schools.com/xml</link>
   <description>New XML tutorial on W3Schools</description>
   <media:thumbnail url="http://image"></media>
 </item>
</channel>

</rss>
`

	// We create a fake transport which will return a valid XML response
	// after 800 milliseconds.
	c := WithHTTPClient(&http.Client{
		Transport: rtf(func(r *http.Request) *http.Response {
			<-time.After(800 * time.Millisecond)

			return &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader(xml)),
			}
		}),
	})

	s := storage.NewInMemoryStorage(10)

	// We specifically choose 2 workers. In reality we want
	// any number greater than 1 for this test.
	r := NewReader(s, c, WithWorkers(2))

	// We use a cancellable context so we can force the reader to
	// stop when we want to.
	ctx, cf := context.WithCancel(context.Background())

	// We use the reader to update 2 fake feeds. Because each
	// feed will return after at least 800ms, updated sequentially
	// this should take at least 1.6 seconds.
	r.Update(ctx, []*feed.Feed{
		{
			FeedLink:   &url.URL{
				Scheme:     "https",
				Host:       "rss.local",
			},
		},
		{
			FeedLink:   &url.URL{
				Scheme:     "https",
				Host:       "rss2.local",
			},
		},
	})

	// After one second, we will close the reader. In theory, our two
	// feeds would be processed concurrently through our worker pipeline.
	// So after 1 second, both feeds will be available.
	<-time.After(1 * time.Second)
	cf()

	f, err := s.Feeds()
	if err != nil {
		t.Errorf("error retrieving feeds: %v", err)
		return
	}

	// If the amount of feeds we have are not exactly the two we specified,
	// that points to something in our pipeline failing.
	if len(f) != 2 {
		t.Errorf("feed count incorrect want %v, got %v", 2, len(f))
	}
}