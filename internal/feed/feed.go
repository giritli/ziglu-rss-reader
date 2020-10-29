package feed

import (
	"crypto/md5"
	"encoding/json"
	"github.com/google/uuid"
	"net/url"
	"time"
)

type Feed struct {
	FeedLink   *url.URL
	ModifiedAt time.Time
	Title      string
	Link       string
}

type JSONFeed Feed

func (f *Feed) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		UUID     string
		FeedLink string
		JSONFeed
	}{
		f.UUID().String(),
		f.FeedLink.String(),
		JSONFeed(*f),
	})
}

func (f *Feed) UUID() uuid.UUID {
	return UUIDFromString(f.FeedLink.String())
}

type Image struct {
	Title string
	URL   string
}

type Article struct {
	// We don't show GUID when outputting to JSON
	// because we calculate a UUID from a GUID or
	// Link instead.
	GUID        string `json:"-"`
	Link        string
	Published   time.Time
	Title       string
	Description string
	Image       *Image
}

type JSONArticle Article

func (a *Article) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		JSONArticle
		UUID string
	}{
		JSONArticle(*a),
		a.UUID().String(),
	})
}

func (a *Article) UUID() uuid.UUID {
	if a.GUID != "" {
		return UUIDFromString(a.GUID)
	}

	return UUIDFromString(a.Link)
}

// UUIDFromString will convert a given string to a UUID
func UUIDFromString(s string) uuid.UUID {
	// md5 the given string which is a 16 byte
	// hash. UUID's are also 16 bytes so we
	// convert the md5 hash into a UUID.
	res := md5.Sum([]byte(s))

	// Ignore the error as an error only returns if
	// byte length is not 16, which we can guarantee
	// that it is.
	uuid, _ := uuid.FromBytes(res[:])

	return uuid
}
