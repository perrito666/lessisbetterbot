package skills

import (
	"log"

	"github.com/mmcdole/gofeed"
)

const rssPolitics = "https://www.lavoz.com.ar/rss/politica.xml"

// FetchPoliticalNews goes to the yellow local paper rss and fetches what passes for
// political news.
func FetchPoliticalNews(logger *log.Logger) ([]string, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(rssPolitics)
	if err != nil {
		return nil, err
	}
	results := make([]string, len(feed.Items), len(feed.Items))
	log.Printf("Found %d news from la voz", len(feed.Items))
	for i := range feed.Items {
		results[i] = feed.Items[i].Link
	}
	return results, nil
}
