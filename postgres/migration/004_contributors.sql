CREATE TABLE IF NOT EXISTS contributors (
	id SERIAL PRIMARY KEY,
	repo_id INT NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
	user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	owner_id INT NOT NULL,
	is_admin BOOLEAN NOT NULL DEFAULT FALSE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	UNIQUE(repo_id, user_id)
)
