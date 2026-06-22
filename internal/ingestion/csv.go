package ingestion

import (
	"context"
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type CSVImporter struct{}

func NewCSVImporter() *CSVImporter { return &CSVImporter{} }

var _ SourceImporter = (*CSVImporter)(nil)

func (c *CSVImporter) Import(_ context.Context, query string) (*ImportedShow, error) {
	f, err := os.Open(query)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, err
	}

	name := strings.TrimSuffix(filepath.Base(query), filepath.Ext(query))
	show := &ImportedShow{
		ExternalID: "csv:" + query,
		Title:      name,
		Slug:       slugify(name),
		Language:   "ar",
		URL:        query,
	}
	for i, row := range rows {
		if i == 0 && len(row) > 0 && row[0] == "external_id" {
			continue // header
		}
		if len(row) < 5 {
			continue
		}
		num, _ := strconv.Atoi(row[3])
		dur, _ := strconv.Atoi(row[4])
		media := ""
		if len(row) > 5 {
			media = row[5]
		}
		show.Episodes = append(show.Episodes, ImportedEpisode{
			ExternalID:      row[0],
			Title:           row[1],
			Description:     row[2],
			Slug:            slugify(row[0]),
			EpisodeNumber:   num,
			DurationSeconds: dur,
			MediaURL:        media,
		})
	}
	return show, nil
}
