package ingestion

import (
	"log/slog"
	"sort"

	"github.com/yazeedalorainy/thmanyah/internal/config"
)

// Importers builds the registry of enabled source importers from config.
// Adding a source = a new adapter + one line here (open/closed).
func Importers(cfg config.IngestionConfig) map[string]SourceImporter {
	m := make(map[string]SourceImporter)
	if cfg.RSS.Enabled {
		m["rss"] = NewRSSImporter(nil)
	}
	if cfg.CSV.Enabled {
		m["csv"] = NewCSVImporter()
	}
	if cfg.YouTube.Enabled {
		m["youtube"] = NewYouTubeImporter(cfg.YouTube.APIKey)
	}
	sources := make([]string, 0, len(m))
	for source := range m {
		sources = append(sources, source)
	}
	sort.Strings(sources)
	slog.Info("ingestion importers registered", "sources", sources)
	return m
}
