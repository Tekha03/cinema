CREATE TABLE movies (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    title VARCHAR(200) NOT NULL
        CHECK (char_length(btrim(title)) > 0),

    description TEXT NOT NULL DEFAULT ''
        CHECK (char_length(description) <= 5000),
    
    release_year INTEGER NOT NULL
        CHECK (release_year BETWEEN 1888 AND 2100),
    
    video_url VARCHAR(2048) NOT NULL
        CHECK (video_url ~ '^https?://'),
    
    created_at TIMESTAMPZ NOT NULL DEFAULT NOW(),

    updated_at TIMESTAMPZ NOT NULL DEFAULT NOW()
);