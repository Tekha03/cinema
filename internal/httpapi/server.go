package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"
	"time"

	"example.com/cinema/internal/movie"
	"example.com/cinema/web"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type API struct {
	repo *movie.Repository
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) http.Handler {
	api := &API{
		repo: movie.NewRepository(pool),
		pool: pool,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", api.health)
	mux.HandleFunc("GET /readyz", api.ready)

	mux.HandleFunc("GET /api/movies", api.list)
	mux.HandleFunc("POST /api/movies", api.create)

	mux.HandleFunc("GET /api/movies/{id}", api.get)
	mux.HandleFunc("PUT /api/movies/{id}", api.update)
	mux.HandleFunc("DELETE /api/movies/{id}", api.delete)

	mux.Handle("GET /", http.FileServer(http.FS(web.Files)))

	return accessLog(mux)
}

func (a *API) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "alive",
	})
}

func (a *API) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	// Проверяет доступ к БД и наличие основной таблицы.
	_, err := a.pool.Exec(ctx, `SELECT id FROM movies LIMIT 0`)
	if err != nil {
		writeError(
			w,
			http.StatusServiceUnavailable,
			"база данных или схема недоступна",
		)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ready",
	})
}

func (a *API) list(w http.ResponseWriter, r *http.Request) {
	limit, ok := queryInt(w, r, "limit", 12, 1, 100)
	if !ok {
		return
	}

	offset, ok := queryInt(w, r, "offset", 0, 0, 1_000_000)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	movies, err := a.repo.List(ctx, limit, offset)
	if err != nil {
		serverError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, movies)
}

func (a *API) get(w http.ResponseWriter, r *http.Request) {
	id, ok := readID(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	m, err := a.repo.Get(ctx, id)
	if err != nil {
		serverError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, m)
}

func (a *API) create(w http.ResponseWriter, r *http.Request) {
	in, ok := readInput(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	m, err := a.repo.Create(ctx, in)
	if err != nil {
		serverError(w, err)
		return
	}

	w.Header().Set(
		"Location",
		"/api/movies/"+strconv.FormatInt(m.ID, 10),
	)
	writeJSON(w, http.StatusCreated, m)
}

func (a *API) update(w http.ResponseWriter, r *http.Request) {
	id, ok := readID(w, r)
	if !ok {
		return
	}

	in, ok := readInput(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	m, err := a.repo.Update(ctx, id, in)
	if err != nil {
		serverError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, m)
}

func (a *API) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := readID(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := a.repo.Delete(ctx, id); err != nil {
		serverError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func readID(
	w http.ResponseWriter,
	r *http.Request,
) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "некорректный ID фильма")
		return 0, false
	}

	return id, true
}

func queryInt(
	w http.ResponseWriter,
	r *http.Request,
	name string,
	fallback, min, max int,
) (int, bool) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback, true
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value < min || value > max {
		writeError(
			w,
			http.StatusBadRequest,
			"некорректный параметр "+name,
		)
		return 0, false
	}

	return value, true
}

func readInput(
	w http.ResponseWriter,
	r *http.Request,
) (movie.Input, bool) {
	var in movie.Input

	contentType, _, err := mime.ParseMediaType(
		r.Header.Get("Content-Type"),
	)
	if err != nil || contentType != "application/json" {
		writeError(
			w,
			http.StatusUnsupportedMediaType,
			"ожидается Content-Type: application/json",
		)
		return in, false
	}

	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&in); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(
				w,
				http.StatusRequestEntityTooLarge,
				"тело запроса слишком большое",
			)
		} else {
			writeError(w, http.StatusBadRequest, "некорректный JSON")
		}
		return in, false
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		writeError(
			w,
			http.StatusBadRequest,
			"ожидается один JSON-объект",
		)
		return in, false
	}

	if err := in.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return in, false
	}

	return in, true
}

func serverError(w http.ResponseWriter, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "фильм не найден")
		return
	}

	slog.Error("database operation failed", "error", err)
	writeError(
		w,
		http.StatusInternalServerError,
		"внутренняя ошибка сервера",
	)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Debug("response write failed", "error", err)
	}
}

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (w *responseRecorder) WriteHeader(status int) {
	if w.status != 0 {
		return
	}

	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseRecorder) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}

	return w.ResponseWriter.Write(data)
}

func (w *responseRecorder) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &responseRecorder{ResponseWriter: w}

		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")

		next.ServeHTTP(recorder, r)

		slog.Info(
			"http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.status,
			"duration_ms", time.Since(started).Milliseconds(),
		)
	})
}
