package ingestion

import (
	"context"
	"errors"
)

type YouTubeImporter struct {
	apiKey string
}

func NewYouTubeImporter(apiKey string) *YouTubeImporter { return &YouTubeImporter{apiKey: apiKey} }

var _ SourceImporter = (*YouTubeImporter)(nil)

var errYouTubeNotConfigured = errors.New("ingestion: youtube importer requires an API key")

// mock since it requires paid API key
// page := ""
//
//	for {
//	    // 1. list the playlist's videos
//	    GET https://www.googleapis.com/youtube/v3/playlistItems
//	        ?part=snippet,contentDetails&playlistId={query}&maxResults=50
//	        &pageToken={page}&key={apiKey}
//
//	    for _, item := range resp.Items {
//	        videoID := item.ContentDetails.VideoId
//
//	        // 2. fetch video details (duration is ISO-8601, e.g. "PT1H2M3S")
//	        GET https://www.googleapis.com/youtube/v3/videos
//	            ?part=snippet,contentDetails&id={videoID}&key={apiKey}
//
//	        episodes = append(episodes, ImportedEpisode{
//	            ExternalID:      videoID,
//	            Title:           video.Snippet.Title,
//	            Description:     video.Snippet.Description,
//	            Slug:            slugify(videoID),
//	            PublishedAt:     video.Snippet.PublishedAt,
//	            DurationSeconds: parseISO8601(video.ContentDetails.Duration),
//	            MediaURL:        "https://youtube.com/watch?v=" + videoID,
//	        })
//	    }
//	    if resp.NextPageToken == "" { break }
//	    page = resp.NextPageToken
//	}
//
// return &ImportedShow{ExternalID: query, Title: playlist.Title, Episodes: episodes}, nil
func (y *YouTubeImporter) Import(_ context.Context, _ string) (*ImportedShow, error) {
	if y.apiKey == "" {
		return nil, errYouTubeNotConfigured
	}
	return nil, errors.New("ingestion: youtube importer not implemented in this build")
}
