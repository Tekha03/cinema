package movie

import (
	"errors"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

type Movie struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	ReleaseYear string    `json:"release_year"`
	VideoURL    string    `json:"json_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Input struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ReleaseYear int    `json:"release_year"`
	VideoURL    string `json:"video_url"`
}

func (in *Input) Validate() error {
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	in.VideoURL = strings.TrimSpace(in.VideoURL)

	if strings.ContainsRune(
		in.Title+in.Description+in.VideoURL,
		'\x00',
	) {
		return errors.New("the zero character is forbidden in strings")
	}

	titleLength := utf8.RuneCountInString(in.Title)
	if titleLength < 1 || titleLength > 200 {
		return errors.New("the name must be between 1 and 200 characters")
	}
	if utf8.RuneCountInString(in.Description) > 5000 {
		return errors.New("the description must contain no more than 5000 characters")
	}
	if in.ReleaseYear < 1888 || in.ReleaseYear > 2100 {
		return errors.New("the year of release must be between 1888 and 2100")
	}
	if len(in.VideoURL) > 2048 {
		return errors.New("video url is too long")
	}

	u, err := url.Parse(in.VideoURL)
	if err != nil {
		return errors.New("invalid video url")
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil {
		return errors.New("need absolute HTTP/HTTPS-link without login and password")
	}

	return nil
}
