CREATE TABLE repos (
	id SERIAL PRIMARY KEY,
	user_id INT NOT NULL REFERENCES users(id),
	name VARCHAR(32) NOT NULL,
	invite_code VARCHAR(255) UNIQUE NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)
