package entries

import (
	"fmt"
	"net/http"

	"github.com/husio/x/feedreader/format/rss"
)

func Fetch(urlStr string) ([]*Entry, error) {
	resp, err := http.Get(urlStr)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch: %s", err)
	}
	defer resp.Body.Close()

	feed, err := rss.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot parse response: %s", err)
	}

	var entries []*Entry
	for _, item := range feed.Items {
		e := Entry{
			Title:   item.Title,
			Link:    item.Link,
			Content: coalesce(item.Content, item.Description),
		}
		entries = append(entries, &e)
	}
	return entries, nil
}

func coalesce(input ...string) string {
	for _, s := range input {
		if len(s) > 0 {
			return s
		}
	}
	return ""
}
