package reader

import (
	"context"
	"errors"
	"github.com/mmcdole/gofeed"
	"github.com/pquerna/cachecontrol/cacheobject"
	"log"
	"net/http"
	"net/url"
	"reader/internal/feed"
	"reader/internal/storage"
	"time"
)

var (
	ErrNotModified = errors.New("304 not modified")
)

type Reader struct {
	s                storage.Storage
	p                *gofeed.Parser
	f                []*url.URL
	c                *http.Client
	workers          uint
	retry            time.Duration
	retryNotModified time.Duration
	retryAfterError  time.Duration
}

type queuedFeed struct {
	f *feed.Feed
	d time.Duration
}

func newQueuedFeed(f *feed.Feed, d time.Duration) queuedFeed {
	return queuedFeed{f: f, d: d}
}

type cachedParsedFeed struct {
	f *gofeed.Feed
	d time.Duration
}

type Option func(*Reader)

func WithHTTPClient(client *http.Client) Option {
	return func(reader *Reader) {
		reader.c = client
	}
}

func WithWorkers(workers uint) Option {
	return func(reader *Reader) {
		if workers <= 0 {
			reader.workers = 1
			return
		}

		reader.workers = workers
	}
}

func WithRetryDuration(retry time.Duration) Option {
	return func(reader *Reader) {
		reader.retry = retry
	}
}

func WithRetryNotModifiedDuration(retry time.Duration) Option {
	return func(reader *Reader) {
		reader.retryNotModified = retry
	}
}

func WithRetryAfterErrorDuration(retry time.Duration) Option {
	return func(reader *Reader) {
		reader.retryAfterError = retry
	}
}

// NewReader will instantiate a reader with default options
// which can be overridden by a select number of option functions.
func NewReader(s storage.Storage, options ...Option) *Reader {
	r := &Reader{
		s: s,
		p: gofeed.NewParser(),
	}

	defaultOptions := []Option{
		WithWorkers(8),
		WithHTTPClient(&http.Client{}),
		WithRetryDuration(60 * time.Second),
		WithRetryNotModifiedDuration(120 * time.Second),
		WithRetryAfterErrorDuration(300 * time.Second),
	}

	for _, opt := range append(defaultOptions, options...) {
		opt(r)
	}

	return r
}

// Update will spawn a long running process which will queue
// and process a given list of feeds. It utilises timeouts so
// that we do not bombard the feed servers with requests and
// also utilises a worker pool pattern to ensure we do not
// make too many concurrent requests.
// An error channel is provided so that we can keep track
// of any errors that occur when processing feeds.
func (r *Reader) Update(ctx context.Context, feeds []*feed.Feed) <-chan error {
	errChan := make(chan error)
	queuedChan := make(chan queuedFeed, len(feeds))
	feedChan := make(chan *feed.Feed)

	// Queue initial feeds into queued feed channel with default retry timeout
	for _, f := range feeds {
		queuedChan <- newQueuedFeed(f, r.retry)
	}

	go func() {
		defer close(queuedChan)
		defer close(feedChan)
		defer close(errChan)

		// Goroutines are cheap so we can use them to wait for a given
		// request timeout before we pass the feed to the main processing
		// queue.
		for qf := range queuedChan {
			select {
			case <-ctx.Done():
				return
			default:
			}

			go func(qf queuedFeed) {
				duration := qf.f.ModifiedAt.Add(qf.d).Sub(time.Now())
				log.Printf("queuing feed %s in %s", qf.f.UUID(), duration)

				select {
				case <-ctx.Done():
					return
				case <-time.After(duration):
				}

				feedChan <- qf.f
			}(qf)
		}
	}()

	// Spawn n number of workers to allow for concurrent processing
	// of given feeds. Once a feed is processed, it is then pushed
	// to the queued feed with an appropriate delay.
	for i := uint(0); i < r.workers; i++ {
		go func() {
			for f := range feedChan {

				// if our context is closed, don't process anymore feeds
				select {
				case <-ctx.Done():
					return
				default:
				}

				cf, err := r.getFeedContent(ctx, f)
				f.ModifiedAt = time.Now()
				if err != nil {
					queuedChan <- newQueuedFeed(f, cf.d)
					errChan <- err
					continue
				}

				f, articles := r.mapParsedFeedToFeedAndArticles(cf.f, f)

				if err := r.s.Store(f, articles); err != nil {
					queuedChan <- newQueuedFeed(f, cf.d)
					errChan <- err
					continue
				}

				queuedChan <- newQueuedFeed(f, cf.d)
			}
		}()
	}

	return errChan
}

func (r *Reader) mapParsedFeedToFeedAndArticles(pf *gofeed.Feed, f *feed.Feed) (*feed.Feed, []*feed.Article) {
	f.Title = pf.Title
	f.Link = pf.Link

	articles := make([]*feed.Article, 0, len(pf.Items))

	for _, a := range pf.Items {
		published := time.Time{}
		if a.PublishedParsed != nil {
			published = *a.PublishedParsed
		}

		article := &feed.Article{
			GUID:        a.GUID,
			Published:   published,
			Title:       a.Title,
			Description: a.Description,
			Link:        a.Link,
		}

		// If an image was parsed from the article, add it here
		if a.Image != nil {
			article.Image = &feed.Image{
				Title: a.Image.Title,
				URL:   a.Image.URL,
			}

			// Otherwise try to see if a media:thumbnail tag exists with
			// a valid url attribute and use that instead.
		} else {
			if media, ok := a.Extensions["media"]; ok {
				if thumb, ok := media["thumbnail"]; ok {
					if len(thumb) > 0 {
						if url, ok := thumb[0].Attrs["url"]; ok {
							article.Image = &feed.Image{
								Title: "Thumbnail",
								URL:   url,
							}
						}
					}
				}
			}
		}

		articles = append(articles, article)
	}

	return f, articles
}

// getFeedContent will try to get content of given feed
// and pass back a cached feed with a retry timeout
// which is hopefully controlled by the feed server's
// Cache-Control response header.
func (r *Reader) getFeedContent(ctx context.Context, f *feed.Feed) (feed cachedParsedFeed, err error) {

	// Set default feed retry duration to error duration
	feed.d = r.retryAfterError

	req, err := http.NewRequestWithContext(ctx, "GET", f.FeedLink.String(), nil)
	if err != nil {
		return feed, err
	}

	// Send If-Modified-Since to allow for server to return 304 if needed
	req.Header.Set("If-Modified-Since", f.ModifiedAt.Format(time.RFC1123))

	resp, err := r.c.Do(req)
	if err != nil {
		return
	}

	if resp != nil {

		// Defer closing of response body
		defer func() {
			if ce := resp.Body.Close(); ce != nil {
				err = ce
			}
		}()
	}

	directives, err := cacheobject.ParseResponseCacheControl(resp.Header.Get("Cache-Control"))
	if err == nil {

		// If the server returns a max-age directive, respect it otherwise
		// revert to default retry timeout
		if directives.MaxAge > 0 {
			feed.d = time.Second * time.Duration(directives.MaxAge)
		} else {
			feed.d = r.retry
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusNotModified {

			// If timeout for 304 is greater than given max-age, respect
			// that value instead.
			if r.retryNotModified > feed.d {
				feed.d = r.retryNotModified
			}

			return feed, ErrNotModified
		}

		return feed, gofeed.HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
		}
	}

	pf, err := r.p.Parse(resp.Body)
	feed.f = pf

	return
}
