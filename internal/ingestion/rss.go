package ingestion

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type RSSImporter struct {
	client *http.Client
}

func NewRSSImporter(client *http.Client) *RSSImporter {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &RSSImporter{client: client}
}

var _ SourceImporter = (*RSSImporter)(nil)

type rssFeed struct {
	Channel struct {
		Title       string `xml:"title"`
		Description string `xml:"description"`
		Language    string `xml:"language"`
		Items       []struct {
			GUID        string `xml:"guid"`
			Title       string `xml:"title"`
			Description string `xml:"description"`
			PubDate     string `xml:"pubDate"`
			Duration    string `xml:"duration"` // itunes:duration (matched by local name)
			Enclosure   struct {
				URL string `xml:"url,attr"`
			} `xml:"enclosure"`
		} `xml:"item"`
	} `xml:"channel"`
}

func (r *RSSImporter) Import(ctx context.Context, query string) (*ImportedShow, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, query, nil)
	if err != nil {
		return nil, err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rss: fetch %s: status %d", query, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, err
	}

	var feed rssFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("rss: parse: %w", err)
	}

	show := &ImportedShow{
		ExternalID:  query, // the feed URL is the stable id
		Title:       feed.Channel.Title,
		Slug:        slugify(feed.Channel.Title),
		Description: feed.Channel.Description,
		Language:    feed.Channel.Language,
		URL:         query,
	}
	for i, it := range feed.Channel.Items {
		extID := firstNonEmpty(it.GUID, it.Title)
		show.Episodes = append(show.Episodes, ImportedEpisode{
			ExternalID:      extID,
			Title:           it.Title,
			Description:     it.Description,
			Slug:            slugify(extID),
			EpisodeNumber:   i + 1,
			DurationSeconds: parseDuration(it.Duration),
			PublishedAt:     parsePubDate(it.PubDate),
			MediaURL:        it.Enclosure.URL,
		})
	}
	return show, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func parseDuration(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if !strings.Contains(s, ":") {
		n, _ := strconv.Atoi(s)
		return n
	}
	total := 0
	for _, part := range strings.Split(s, ":") {
		n, err := strconv.Atoi(part)
		if err != nil {
			return 0
		}
		total = total*60 + n
	}
	return total
}

func parsePubDate(s string) time.Time {
	for _, layout := range []string{time.RFC1123Z, time.RFC1123, time.RFC822Z, time.RFC822} {
		if t, err := time.Parse(layout, strings.TrimSpace(s)); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}
