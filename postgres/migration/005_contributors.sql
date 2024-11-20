CREATE TABLE contributors (
	id SERIAL PRIMARY KEY,
	repo_id INT NOT NULL REFERENCES repos(id),
	user_id INT NOT NULL REFERENCES users(id),
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	UNIQUE(repo_id, user_id)
)
