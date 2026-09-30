"use strict";

const form = document.getElementById("movie-form");
const formFields = document.getElementById("form-fields");
const formTitle = document.getElementById("form-title");
const cancelEdit = document.getElementById("cancel-edit");

const fields = {
  title: document.getElementById("title"),
  description: document.getElementById("description"),
  releaseYear: document.getElementById("release-year"),
  videoURL: document.getElementById("video-url"),
};

const list = document.getElementById("movie-list");
const template = document.getElementById("movie-template");
const message = document.getElementById("message");

const previousPage = document.getElementById("previous-page");
const nextPage = document.getElementById("next-page");
const pageLabel = document.getElementById("page-label");

const player = document.getElementById("player");
const playerPanel = document.getElementById("player-panel");
const playerTitle = document.getElementById("player-title");

const pageSize = 12;

let offset = 0;
let hasNextPage = false;
let editingID = null;
let playingID = null;
let busy = false;

async function api(path, options = {}) {
  const response = await fetch(path, options);

  if (response.status === 204) {
    return null;
  }

  const data = await response.json();

  if (!response.ok) {
    throw new Error(data.error || `Ошибка HTTP ${response.status}`);
  }

  return data;
}

function updateControls() {
  document.querySelectorAll("button").forEach((button) => {
    button.disabled = busy;
  });

  formFields.disabled = busy;
  previousPage.disabled = busy || offset === 0;
  nextPage.disabled = busy || !hasNextPage;
}

async function run(action) {
  if (busy) {
    return;
  }

  busy = true;
  message.textContent = "";
  updateControls();

  try {
    await action();
  } catch (error) {
    message.textContent = error.message || "Не удалось выполнить действие";
  } finally {
    busy = false;
    updateControls();
  }
}

function resetEditor() {
  editingID = null;
  form.reset();
  fields.releaseYear.value = new Date().getFullYear();
  formTitle.textContent = "Добавить фильм";
  cancelEdit.hidden = true;
}

function closePlayer() {
  player.pause();
  player.removeAttribute("src");
  player.load();

  playerPanel.hidden = true;
  playingID = null;
}

async function watchMovie(id) {
  const movie = await api(`/api/movies/${id}`);

  playingID = movie.id;
  playerTitle.textContent = movie.title;
  player.src = movie.video_url;
  player.load();

  playerPanel.hidden = false;
  playerPanel.scrollIntoView({ behavior: "smooth", block: "start" });
}

async function editMovie(id) {
  const movie = await api(`/api/movies/${id}`);

  editingID = movie.id;
  fields.title.value = movie.title;
  fields.description.value = movie.description;
  fields.releaseYear.value = movie.release_year;
  fields.videoURL.value = movie.video_url;

  formTitle.textContent = "Редактировать фильм";
  cancelEdit.hidden = false;
  form.scrollIntoView({ behavior: "smooth", block: "start" });
}

async function deleteMovie(movie) {
  if (!window.confirm(`Удалить фильм «${movie.title}»?`)) {
    return;
  }

  await api(`/api/movies/${movie.id}`, {
    method: "DELETE",
  });

  if (editingID === movie.id) {
    resetEditor();
  }

  if (playingID === movie.id) {
    closePlayer();
  }

  await loadMovies();
  message.textContent = "Фильм удалён";
}

function renderMovies(movies) {
  list.replaceChildren();

  if (movies.length === 0) {
    const empty = document.createElement("p");
    empty.textContent = "Каталог пока пуст. Добавь первый фильм.";
    list.append(empty);
    return;
  }

  for (const movie of movies) {
    const card = template.content.firstElementChild.cloneNode(true);

    card.querySelector(".movie-title").textContent = movie.title;
    card.querySelector(".movie-year").textContent =
      `Год выпуска: ${movie.release_year}`;
    card.querySelector(".movie-description").textContent =
      movie.description || "Описание не добавлено";

    card.querySelector('[data-action="watch"]').addEventListener(
      "click",
      () => void run(() => watchMovie(movie.id)),
    );

    card.querySelector('[data-action="edit"]').addEventListener(
      "click",
      () => void run(() => editMovie(movie.id)),
    );

    card.querySelector('[data-action="delete"]').addEventListener(
      "click",
      () => void run(() => deleteMovie(movie)),
    );

    list.append(card);
  }
}

async function loadMovies() {
  // Берём на одну запись больше, чтобы узнать, есть ли следующая страница.
  let movies = await api(
    `/api/movies?limit=${pageSize + 1}&offset=${offset}`,
  );

  // После удаления последней записи на странице возвращаемся назад.
  if (movies.length === 0 && offset > 0) {
    offset = Math.max(0, offset - pageSize);

    movies = await api(
      `/api/movies?limit=${pageSize + 1}&offset=${offset}`,
    );
  }

  hasNextPage = movies.length > pageSize;
  renderMovies(movies.slice(0, pageSize));

  pageLabel.textContent = `Страница ${Math.floor(offset / pageSize) + 1}`;
}

form.addEventListener("submit", (event) => {
  event.preventDefault();

  void run(async () => {
    const input = {
      title: fields.title.value,
      description: fields.description.value,
      release_year: Number(fields.releaseYear.value),
      video_url: fields.videoURL.value,
    };

    const path = editingID === null
      ? "/api/movies"
      : `/api/movies/${editingID}`;

    const method = editingID === null ? "POST" : "PUT";

    const saved = await api(path, {
      method,
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(input),
    });

    if (playingID === saved.id) {
      closePlayer();
    }

    resetEditor();
    offset = 0;
    await loadMovies();

    message.textContent = "Фильм сохранён";
  });
});

cancelEdit.addEventListener("click", resetEditor);

document.getElementById("refresh").addEventListener("click", () => {
  void run(loadMovies);
});

previousPage.addEventListener("click", () => {
  void run(async () => {
    offset = Math.max(0, offset - pageSize);
    await loadMovies();
  });
});

nextPage.addEventListener("click", () => {
  void run(async () => {
    offset += pageSize;
    await loadMovies();
  });
});

document.getElementById("close-player").addEventListener(
  "click",
  closePlayer,
);

player.addEventListener("error", () => {
  if (player.getAttribute("src")) {
    message.textContent =
      "Видео не загрузилось. Проверь прямую ссылку и поддерживаемый формат.";
  }
});

resetEditor();
void run(loadMovies);
