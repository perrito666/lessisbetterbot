package skills

import (
	"log"
	"regexp"

	"github.com/mmcdole/gofeed"
)

const rssPolitics = "https://www.lavoz.com.ar/rss/politica.xml"

var econoRe = regexp.MustCompile(`divisa|dólar|cotiz|cepo`)

// FetchPoliticalNews goes to the yellow local paper rss and fetches what passes for
// political news.
func FetchPoliticalNews(logger *log.Logger) (map[string]string, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(rssPolitics)
	if err != nil {
		return nil, err
	}
	results := map[string]string{}

	for i := range feed.Items {
		if econoRe.Match([]byte(feed.Items[i].Description)) {
			results[feed.Items[i].Link] = feed.Items[i].Description
		}
	}

	return results, nil
}
