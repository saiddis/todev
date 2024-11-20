CREATE TABLE auths (
	id SERIAL PRIMARY KEY,
	user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	source VARCHAR(16) NOT NULL,
	source_id VARCHAR(16) NOT NULL,
	access_token VARCHAR(255) NOT NULL,
	refresh_token VARCHAR(255) NOT NULL,
	expiry TIMESTAMP,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

	UNIQUE(user_id, source),
	UNIQUE(source, source_id)
)
