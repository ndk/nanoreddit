package protocol

type Post struct {
	Title     string `json:"title"`
	Author    string `json:"author" validate:"author"`
	Link      string `json:"link,omitempty" validate:"omitempty,url"`
	Subreddit string `json:"subreddit"`
	Content   string `json:"content,omitempty"`
	Score     int    `json:"score"`
	Promoted  bool   `json:"promoted"`
	NSFW      bool   `json:"nsfw"`
}
