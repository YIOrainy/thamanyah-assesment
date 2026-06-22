package catalog

import (
	"time"

	"github.com/google/uuid"
)

type ContentType string

const (
	ContentTypeAudio ContentType = "audio"
	ContentTypeVideo ContentType = "video"
)

func (c ContentType) IsValid() bool {
	switch c {
	case ContentTypeAudio, ContentTypeVideo:
		return true
	}
	return false
}

type Episode struct {
	ID          uuid.UUID `json:"id"`
	ShowID      uuid.UUID `json:"show_id"`
	Title       string    `json:"title"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`

	EpisodeNumber   int         `json:"episode_number"`
	ContentType     ContentType `json:"content_type"`
	Language        string      `json:"language"`
	DurationSeconds int         `json:"duration_seconds"`
	Status          Status      `json:"status"`
	PublishedAt     *time.Time  `json:"published_at,omitempty"`

	SearchText string `json:"-"`

	People []Credit `json:"people,omitempty"`
	Topics []Topic  `json:"topics,omitempty"`

	CreatedBy uuid.UUID `json:"created_by"`
	UpdatedBy uuid.UUID `json:"updated_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// creates a draft episode
func NewEpisode(showID uuid.UUID, title, slug, description string, number int, ct ContentType, language string, durationSeconds int, createdBy uuid.UUID) *Episode {
	now := time.Now().UTC()
	return &Episode{
		ID:              uuid.Must(uuid.NewV7()),
		ShowID:          showID,
		Title:           title,
		Slug:            slug,
		Description:     description,
		EpisodeNumber:   number,
		ContentType:     ct,
		Language:        language,
		DurationSeconds: durationSeconds,
		Status:          StatusDraft,
		CreatedBy:       createdBy,
		UpdatedBy:       createdBy,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}
