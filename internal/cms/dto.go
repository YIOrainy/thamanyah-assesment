package cms

// pagedResponse is the CMS offset-pagination envelope.
type pagedResponse struct {
	Items    any `json:"items"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

type importRequest struct {
	Source string `json:"source" validate:"required,oneof=rss csv youtube"`
	Query  string `json:"query" validate:"required"`
}

type loginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type loginResponse struct {
	AccessToken string `json:"access_token"`
}

type createShowRequest struct {
	Title       string `json:"title" validate:"required"`
	Slug        string `json:"slug" validate:"required"`
	Description string `json:"description"`
	Format      string `json:"format" validate:"required,oneof=podcast documentary sports"`
	Language    string `json:"language" validate:"required"`
}

type updateShowRequest struct {
	Title       *string `json:"title" validate:"omitempty,min=1"`
	Description *string `json:"description"`
	Format      *string `json:"format" validate:"omitempty,oneof=podcast documentary sports"`
	Language    *string `json:"language" validate:"omitempty,min=1"`
}

type createEpisodeRequest struct {
	Title           string `json:"title" validate:"required"`
	Slug            string `json:"slug" validate:"required"`
	Description     string `json:"description"`
	EpisodeNumber   int    `json:"episode_number" validate:"gte=0"`
	ContentType     string `json:"content_type" validate:"required,oneof=audio video"`
	Language        string `json:"language" validate:"required"`
	DurationSeconds int    `json:"duration_seconds" validate:"gte=0"`
}
