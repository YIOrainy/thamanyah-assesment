package cms

// pagedResponse is the CMS offset-pagination envelope.
type pagedResponse struct {
	Items    any `json:"items"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

type importRequest struct {
	Source string `json:"source"`
	Query  string `json:"query"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken string `json:"access_token"`
}

type createShowRequest struct {
	Title       string `json:"title"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Format      string `json:"format"`
	Language    string `json:"language"`
}

type updateShowRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Format      *string `json:"format"`
	Language    *string `json:"language"`
}

type createEpisodeRequest struct {
	Title           string `json:"title"`
	Slug            string `json:"slug"`
	Description     string `json:"description"`
	EpisodeNumber   int    `json:"episode_number"`
	ContentType     string `json:"content_type"`
	Language        string `json:"language"`
	DurationSeconds int    `json:"duration_seconds"`
}
