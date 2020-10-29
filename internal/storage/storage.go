package storage

import (
	"errors"
	"github.com/google/uuid"
	"reader/internal/feed"
	"sort"
	"sync"
	"time"
)

type Storage interface {
	Store(feed *feed.Feed, articles []*feed.Article) error
	Feeds() ([]*feed.Feed, error)
	Latest(offset time.Time) ([]*feed.Article, error)
	LatestFromFeed(feed uuid.UUID, offset time.Time) ([]*feed.Article, error)
	Article(article uuid.UUID) (*feed.Article, error)
}

type InMemoryStorage struct {
	feeds    *sync.Map
	articles *sync.Map

	// Minimum number articles to show when viewing latest.
	// We use minimum here because of the time offset rule
	// which theoretically could be more than the number of
	// articles stated to ensure correct pagination.
	minLatest uint
}

func NewInMemoryStorage(maxLatest uint) *InMemoryStorage {
	if maxLatest == 0 {
		maxLatest = 10
	}

	return &InMemoryStorage{
		minLatest: maxLatest,
		feeds:     &sync.Map{},
		articles:  &sync.Map{},
	}
}

func (s *InMemoryStorage) Store(feed *feed.Feed, articles []*feed.Article) error {
	s.feeds.Store(feed.UUID(), feed)

	am, ok := s.articles.Load(feed.UUID())
	if !ok {
		am = &sync.Map{}
	}

	for _, a := range articles {
		am.(*sync.Map).Store(a.UUID(), a)
	}

	s.articles.Store(feed.UUID(), am)

	return nil
}

func (s *InMemoryStorage) Feeds() ([]*feed.Feed, error) {
	feeds := []*feed.Feed{}

	s.feeds.Range(func(key, value interface{}) bool {
		feeds = append(feeds, value.(*feed.Feed))
		return true
	})

	// Sort feeds alphabetically by title
	sort.Slice(feeds, func(i, j int) bool {
		return feeds[i].Title < feeds[j].Title
	})

	return feeds, nil
}

func (s *InMemoryStorage) Latest(offset time.Time) ([]*feed.Article, error) {
	articles := []*feed.Article{}

	s.articles.Range(func(key, value interface{}) bool {
		value.(*sync.Map).Range(func(key, value interface{}) bool {
			articles = append(articles, value.(*feed.Article))
			return true
		})
		return true
	})

	return s.latest(articles, offset)
}

func (s *InMemoryStorage) LatestFromFeed(id uuid.UUID, offset time.Time) ([]*feed.Article, error) {
	f, ok := s.articles.Load(id)

	if !ok {
		return nil, errors.New("feed not found")
	}

	articles := []*feed.Article{}

	f.(*sync.Map).Range(func(key, value interface{}) bool {
		articles = append(articles, value.(*feed.Article))
		return true
	})

	return s.latest(articles, offset)
}

func (s *InMemoryStorage) latest(articles []*feed.Article, offset time.Time) ([]*feed.Article, error) {

	// Sort articles by published date
	sort.Slice(articles, func(i, j int) bool {
		return articles[i].Published.After(articles[j].Published)
	})

	latestArticles := []*feed.Article{}

	for i, j := uint(0), uint(len(articles)); i < j; i++ {

		// If article was published after our offset, break out
		if !offset.After(articles[i].Published) {
			continue
		}

		latestArticles = append(latestArticles, articles[i])

		// Because our pagination is time based, we want to make sure we grab
		// all articles that continue with the same published time
		// before we take into account maximum articles.
		if i < j-1 && articles[i].Published.Equal(articles[i+1].Published) {
			continue
		}

		if uint(len(latestArticles)) > s.minLatest-1 {
			break
		}
	}

	return latestArticles, nil
}

func (s *InMemoryStorage) Article(id uuid.UUID) (*feed.Article, error) {
	var article *feed.Article

	s.articles.Range(func(key, value interface{}) bool {
		a, ok := value.(*sync.Map).Load(id)
		if ok {
			article = a.(*feed.Article)
			return false
		}
		return true
	})

	if article == nil {
		return nil, errors.New("article not found")
	}

	return article, nil
}
