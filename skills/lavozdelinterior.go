package skills

import "github.com/mmcdole/gofeed"

const rssPolitics = "https://www.lavoz.com.ar/rss/politica.xml"

// FetchPoliticalNews goes to the yellow local paper rss and fetches what passes for
// political news.
func FetchPoliticalNews() ([]string, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(rssPolitics)
	if err != nil {
		return nil, err
	}
	results := make([]string, len(feed.Items), len(feed.Items))
	for i := range feed.Items {
		results[i] = feed.Items[i].Link
	}
	return results, nil
}
