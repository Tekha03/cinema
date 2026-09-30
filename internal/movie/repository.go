package movie

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const columns = `
	id,
	title,
	description,
	release_year,
	video_url,
	created_at,
	updated_at
`

func scanMovie(row pgx.Row) (Movie, error) {
	var movie Movie

	err := row.Scan(
		&movie.ID,
		&movie.Title,
		&movie.Description,
		&movie.ReleaseYear,
		&movie.VideoURL,
		&movie.CreatedAt,
		&movie.UpdatedAt,
	)

	return movie, err
}

func (r *Repository) List(ctx context.Context, limit, offset int) ([]Movie, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+columns+`
		FROM movies
		ORDER BY id DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	movies := make([]Movie, 0)

	for rows.Next() {
		movie, err := scanMovie(rows)
		if err != nil {
			return nil, err
		}

		movies = append(movies, movie)
	}

	return movies, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id int64) (Movie, error) {
	return scanMovie(r.pool.QueryRow(ctx, `
		SELECT `+columns+`
		FROM movies
		WHERE id = $1
	`, id))
}

func (r *Repository) Create(ctx context.Context, in Input) (Movie, error) {
	return scanMovie(r.pool.QueryRow(ctx, `
		INSERT INTO movies (
			title,
			description,
			release_year,
			video_url
		)
		VALUES ($1, $2, $3, $4)
		RETURNING `+columns,
		in.Title,
		in.Description,
		in.ReleaseYear,
		in.VideoURL,
	))
}

func (r *Repository) Update(ctx context.Context, id int64, in Input) (Movie, error) {
	return scanMovie(r.pool.QueryRow(ctx, `
		UPDATE movies
		SET
			title = $2,
			description = $3,
			release_year = $4,
			video_url = $5,
			updated_at = NOW()
		WHERE id = $1
		RETURNING `+columns,
		id,
		in.Title,
		in.Description,
		in.ReleaseYear,
		in.VideoURL,
	))
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	result, err := r.pool.Exec(ctx, `
		DELETE FROM movies
		WHERE id = $1
	`, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}
