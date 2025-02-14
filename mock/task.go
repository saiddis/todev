package mock

import (
	"context"

	"github.com/saiddis/todev"
)

var _ todev.TaskService = (*TaskService)(nil)

type TaskService struct {
	FindTaskByIDFn        func(ctx context.Context, id int) (*todev.Task, error)
	FindTasksFn           func(ctx context.Context, filter todev.TaskFilter) ([]*todev.Task, int, error)
	CreateTaskFn          func(ctx context.Context, task *todev.Task) error
	UpdateTaskFn          func(ctx context.Context, id int, upd todev.TaskUpdate) (*todev.Task, error)
	DeleteTaskFn          func(ctx context.Context, id int) error
	AttachContributorFn   func(ctx context.Context, task *todev.Task, contributorID int) error
	UnattachContributorFn func(ctx context.Context, task *todev.Task, contributorID int) error
}

func (s *TaskService) FindTaskByID(ctx context.Context, id int) (*todev.Task, error) {
	return s.FindTaskByIDFn(ctx, id)
}

func (s *TaskService) FindTasks(ctx context.Context, filter todev.TaskFilter) ([]*todev.Task, int, error) {
	return s.FindTasksFn(ctx, filter)
}

func (s *TaskService) CreateTask(ctx context.Context, task *todev.Task) error {
	return s.CreateTaskFn(ctx, task)
}

func (s *TaskService) UpdateTask(ctx context.Context, id int, upd todev.TaskUpdate) (*todev.Task, error) {
	return s.UpdateTaskFn(ctx, id, upd)
}

func (s *TaskService) DeleteTask(ctx context.Context, id int) error {
	return s.DeleteTaskFn(ctx, id)
}

func (s TaskService) AttachContributor(ctx context.Context, task *todev.Task, contributorID int) error {
	return s.AttachContributorFn(ctx, task, contributorID)
}

func (s TaskService) UnattachContributor(ctx context.Context, task *todev.Task, contributorID int) error {
	return s.UnattachContributorFn(ctx, task, contributorID)
}
