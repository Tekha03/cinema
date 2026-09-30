package movie

import (
	"strings"
	"testing"
)

func TestInputValidate(t *testing.T) {
	tests := []struct {
		name    string
		change  func(*Input)
		wantErr bool
	}{
		{
			name:   "valid input",
			change: func(in *Input) {},
		},
		{
			name: "empty title",
			change: func(in *Input) {
				in.Title = "   "
			},
			wantErr: true,
		},
		{
			name: "200 Cyrillic characters",
			change: func(in *Input) {
				in.Title = strings.Repeat("я", 200)
			},
		},
		{
			name: "title too long",
			change: func(in *Input) {
				in.Title = strings.Repeat("я", 201)
			},
			wantErr: true,
		},
		{
			name: "invalid year",
			change: func(in *Input) {
				in.ReleaseYear = 1700
			},
			wantErr: true,
		},
		{
			name: "unsupported URL scheme",
			change: func(in *Input) {
				in.VideoURL = "javascript:alert(1)"
			},
			wantErr: true,
		},
		{
			name: "URL without host",
			change: func(in *Input) {
				in.VideoURL = "https:///movie.mp4"
			},
			wantErr: true,
		},
		{
			name: "credentials in URL",
			change: func(in *Input) {
				in.VideoURL = "https://user:password@example.org/movie.mp4"
			},
			wantErr: true,
		},
		{
			name: "zero byte",
			change: func(in *Input) {
				in.Description = "text\x00"
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			in := Input{
				Title:       "Фильм",
				Description: "Описание",
				ReleaseYear: 2024,
				VideoURL:    "https://example.org/movie.mp4",
			}

			test.change(&in)

			err := in.Validate()
			if (err != nil) != test.wantErr {
				t.Fatalf(
					"Validate() error = %v, wantErr = %v",
					err,
					test.wantErr,
				)
			}
		})
	}
}

func TestInputValidateTrimsWhitespace(t *testing.T) {
	in := Input{
		Title:       "  Фильм  ",
		Description: "  Описание  ",
		ReleaseYear: 2024,
		VideoURL:    "  https://example.org/movie.mp4  ",
	}

	if err := in.Validate(); err != nil {
		t.Fatal(err)
	}

	if in.Title != "Фильм" ||
		in.Description != "Описание" ||
		in.VideoURL != "https://example.org/movie.mp4" {
		t.Fatalf("whitespace was not trimmed: %+v", in)
	}
}
