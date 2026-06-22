package catalog

import (
	"time"

	"github.com/google/uuid"
)

type Format string

const (
	FormatPodcast     Format = "podcast"
	FormatDocumentary Format = "documentary"
	FormatSports      Format = "sports"
)

func (f Format) IsValid() bool {
	switch f {
	case FormatPodcast, FormatDocumentary, FormatSports:
		return true
	}
	return false
}

type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
	StatusArchived  Status = "archived"
)

func (s Status) IsValid() bool {
	switch s {
	case StatusDraft, StatusPublished, StatusArchived:
		return true
	}
	return false
}

type Show struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	Format      Format    `json:"format"`
	Language    string    `json:"language"`
	Status      Status    `json:"status"`

	People []Credit `json:"people,omitempty"`
	Topics []Topic  `json:"topics,omitempty"`

	CreatedBy uuid.UUID `json:"created_by"`
	UpdatedBy uuid.UUID `json:"updated_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewShow(title, slug, description string, format Format, language string, createdBy uuid.UUID) *Show {
	now := time.Now().UTC()
	return &Show{
		ID:          uuid.Must(uuid.NewV7()),
		Title:       title,
		Slug:        slug,
		Description: description,
		Format:      format,
		Language:    language,
		Status:      StatusDraft,
		CreatedBy:   createdBy,
		UpdatedBy:   createdBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}
