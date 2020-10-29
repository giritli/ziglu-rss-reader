package storage

import (
	"github.com/google/uuid"
	"reader/internal/feed"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestInMemoryStorage_Article(t *testing.T) {
	type fields struct {
		feeds     *sync.Map
		articles  *sync.Map
		minLatest uint
	}
	type args struct {
		id uuid.UUID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *feed.Article
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InMemoryStorage{
				feeds:     tt.fields.feeds,
				articles:  tt.fields.articles,
				minLatest: tt.fields.minLatest,
			}
			got, err := s.Article(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Article() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Article() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInMemoryStorage_Feeds(t *testing.T) {
	type fields struct {
		feeds     *sync.Map
		articles  *sync.Map
		minLatest uint
	}
	tests := []struct {
		name    string
		fields  fields
		want    []*feed.Feed
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InMemoryStorage{
				feeds:     tt.fields.feeds,
				articles:  tt.fields.articles,
				minLatest: tt.fields.minLatest,
			}
			got, err := s.Feeds()
			if (err != nil) != tt.wantErr {
				t.Errorf("Feeds() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Feeds() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInMemoryStorage_Latest(t *testing.T) {
	type fields struct {
		feeds     *sync.Map
		articles  *sync.Map
		minLatest uint
	}
	type args struct {
		offset time.Time
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*feed.Article
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InMemoryStorage{
				feeds:     tt.fields.feeds,
				articles:  tt.fields.articles,
				minLatest: tt.fields.minLatest,
			}
			got, err := s.Latest(tt.args.offset)
			if (err != nil) != tt.wantErr {
				t.Errorf("Latest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Latest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInMemoryStorage_LatestFromFeed(t *testing.T) {
	type fields struct {
		feeds     *sync.Map
		articles  *sync.Map
		minLatest uint
	}
	type args struct {
		id     uuid.UUID
		offset time.Time
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*feed.Article
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InMemoryStorage{
				feeds:     tt.fields.feeds,
				articles:  tt.fields.articles,
				minLatest: tt.fields.minLatest,
			}
			got, err := s.LatestFromFeed(tt.args.id, tt.args.offset)
			if (err != nil) != tt.wantErr {
				t.Errorf("LatestFromFeed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LatestFromFeed() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInMemoryStorage_Store(t *testing.T) {
	type fields struct {
		feeds     *sync.Map
		articles  *sync.Map
		minLatest uint
	}
	type args struct {
		feed     *feed.Feed
		articles []*feed.Article
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InMemoryStorage{
				feeds:     tt.fields.feeds,
				articles:  tt.fields.articles,
				minLatest: tt.fields.minLatest,
			}
			if err := s.Store(tt.args.feed, tt.args.articles); (err != nil) != tt.wantErr {
				t.Errorf("Store() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInMemoryStorage_latest(t *testing.T) {
	type fields struct {
		feeds     *sync.Map
		articles  *sync.Map
		minLatest uint
	}
	type args struct {
		articles []*feed.Article
		offset   time.Time
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*feed.Article
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InMemoryStorage{
				feeds:     tt.fields.feeds,
				articles:  tt.fields.articles,
				minLatest: tt.fields.minLatest,
			}
			got, err := s.latest(tt.args.articles, tt.args.offset)
			if (err != nil) != tt.wantErr {
				t.Errorf("latest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("latest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewInMemoryStorage(t *testing.T) {
	type args struct {
		maxLatest uint
	}
	tests := []struct {
		name string
		args args
		want *InMemoryStorage
	}{
		{
			"latest limit",
			args{0},
			&InMemoryStorage{
				feeds:     &sync.Map{},
				articles:  &sync.Map{},
				minLatest: 10,
			},
		},
		{
			"latest greater than 0",
			args{7},
			&InMemoryStorage{
				feeds:     &sync.Map{},
				articles:  &sync.Map{},
				minLatest: 7,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewInMemoryStorage(tt.args.maxLatest); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewInMemoryStorage() = %v, want %v", got, tt.want)
			}
		})
	}
}