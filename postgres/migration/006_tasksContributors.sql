CREATE TABLE IF NOT EXISTS tasks_contributors (
	task_id BIGINT REFERENCES tasks(id) ON DELETE CASCADE,
	contributor_id INT REFERENCES contributors(id) ON DELETE CASCADE,
	UNIQUE(task_id, contributor_id)
);
