CREATE TABLE tasks (
	id BIGSERIAL PRIMARY KEY,
	description TEXT NOT NULL,
	is_completed BOOLEAN NOT NULL DEFAULT FALSE,
	repo_id INT NOT NULL REFERENCES repos(id),
	contributor_id INT REFERENCES contributors(id),
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)
