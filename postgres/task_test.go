package postgres_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/saiddis/todev"
	"github.com/saiddis/todev/postgres"
)

func TestTaskService_CreateTask(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		WithSchema(t, createTask_OK)
	})

	t.Run("Errors", func(t *testing.T) {
		WithSchema(t, func(t testing.TB, conn *postgres.Conn) {
			createTask_Errors(t.(*testing.T), conn)
		})
	})
}

func TestTaskService_FindTasks(t *testing.T) {
	t.Run("ByContributorID", func(t *testing.T) {
		WithSchema(t, findTasks_ByContributorID)
	})

	t.Run("ByRepoID", func(t *testing.T) {
		WithSchema(t, findTasks_ByRepoID)
	})
}

func TestTaskService_FindTaskByID(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		WithSchema(t, findTaskByID_OK)
	})
}

func TestTaskService_UpdateTask(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		WithSchema(t, updateTask_OK)
	})

	t.Run("Errors", func(t *testing.T) {
		WithSchema(t, func(t testing.TB, conn *postgres.Conn) {
			updateTask_Errors(t.(*testing.T), conn)
		})
	})
}

func TestTaskService_DeleteTask(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		WithSchema(t, deleteTask_OK)
	})

	t.Run("Errors", func(t *testing.T) {
		WithSchema(t, func(t testing.TB, conn *postgres.Conn) {
			deleteTask_Errors(t.(*testing.T), conn)
		})
	})
}

func findTaskByID_OK(t testing.TB, conn *postgres.Conn) {
	s := postgres.NewTaskService(conn)

	ctx := context.Background()
	_, ctx0 := MustCreateUser(t, ctx, conn, &todev.User{Name: "bob", Email: "bob@gmail.com"})
	_, ctx1 := MustCreateUser(t, ctx, conn, &todev.User{Name: "judy", Email: "judy@gmail.com"})
	repo0 := MustCreateRepo(t, ctx0, conn, &todev.Repo{Name: "repo1"})

	contributor1 := MustCreateContributor(t, ctx1, conn, &todev.Contributor{RepoID: repo0.ID})

	task0 := MustCreateTask(t, ctx0, conn, &todev.Task{Description: "Do some stuff.", ContributorID: &contributor1.ID, RepoID: repo0.ID})

	if task, err := s.FindTaskByID(ctx0, task0.ID); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(task0, task) {
		t.Fatalf("mismatch: %#v !=\n%#v", task0, task)
	}
}

func updateTask_OK(t testing.TB, conn *postgres.Conn) {
	s := postgres.NewTaskService(conn)

	ctx := context.Background()
	_, ctx0 := MustCreateUser(t, ctx, conn, &todev.User{Name: "bob", Email: "bob@gmail.com"})
	_, ctx1 := MustCreateUser(t, ctx, conn, &todev.User{Name: "judy", Email: "judy@gmail.com"})
	repo0 := MustCreateRepo(t, ctx0, conn, &todev.Repo{Name: "repo1"})

	contributor1 := MustCreateContributor(t, ctx1, conn, &todev.Contributor{RepoID: repo0.ID})

	task := MustCreateTask(t, ctx0, conn, &todev.Task{Description: "Do some stuff.", ContributorID: &contributor1.ID, RepoID: repo0.ID})

	description := "Do some other stuff."
	contributorID := 2
	toggleCompletion := true
	if task, err := s.UpdateTask(ctx0, task.ID, todev.TaskUpdate{
		Description:      &description,
		ContributorID:    &contributorID,
		ToggleCompletion: toggleCompletion,
	}); err != nil {
		t.Fatal(err)
	} else if other, err := s.FindTaskByID(ctx0, task.ID); err != nil {
		t.Fatal(err)
	} else if got, want := other.Description, description; got != want {
		t.Fatalf("Description: %s, want %s", got, want)
	} else if got, want := *other.ContributorID, contributorID; got != want {
		t.Fatalf("ContributorID: %d, want %d", got, want)
	} else if got, want := other.IsCompleted, true; got != want {
		t.Fatalf("IsComleted: %v, want %v", got, want)
	} else if !reflect.DeepEqual(task, other) {
		t.Fatalf("mismatch: %#v !=\n%#v", task, other)
	}
	// else if !reflect.DeepEqual(repo0.Tasks[0], other) {
	// 	t.Fatalf("mismatch: %#v !=\n%#v", repo0.Tasks[0], other)
	// }
}

func updateTask_Errors(t *testing.T, conn *postgres.Conn) {
	type testData struct {
		ctx      context.Context
		input    todev.TaskUpdate
		expected error
		id       int
	}

	s := postgres.NewTaskService(conn)

	ctx := context.Background()
	_, ctx0 := MustCreateUser(t, ctx, conn, &todev.User{Name: "bob", Email: "bob@gmail.com"})
	_, ctx1 := MustCreateUser(t, ctx, conn, &todev.User{Name: "judy", Email: "judy@gmail.com"})
	repo0 := MustCreateRepo(t, ctx0, conn, &todev.Repo{Name: "repo1"})

	contributor1 := MustCreateContributor(t, ctx1, conn, &todev.Contributor{RepoID: repo0.ID})

	task := MustCreateTask(t, ctx0, conn, &todev.Task{Description: "Do some stuff.", ContributorID: &contributor1.ID, RepoID: repo0.ID})

	description := "Do some other stuff."
	upd := todev.TaskUpdate{Description: &description}

	tests := map[string]testData{
		"ErrUpdateNotAllowed": testData{
			ctx:   ctx1,
			input: upd,
			expected: &todev.Error{
				Code:    todev.ECONFLICT,
				Message: "You are not allowed to update tasks.",
			},
			id: task.ID,
		},
		"ErrNotFound": testData{
			ctx:   ctx0,
			input: upd,
			expected: &todev.Error{
				Code:    todev.ENOTFOUND,
				Message: "Task not found.",
			},
			id: 2,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := s.UpdateTask(tt.ctx, tt.id, tt.input); err == nil {
				t.Fatal("error expected")
			} else if err.Error() != tt.expected.Error() {
				t.Fatalf("unexpected error: %#v", err)
			}
		})
	}

}

func deleteTask_OK(t testing.TB, conn *postgres.Conn) {
	s := postgres.NewTaskService(conn)

	ctx := context.Background()
	_, ctx0 := MustCreateUser(t, ctx, conn, &todev.User{Name: "bob", Email: "bob@gmail.com"})
	_, ctx1 := MustCreateUser(t, ctx, conn, &todev.User{Name: "judy", Email: "judy@gmail.com"})
	repo0 := MustCreateRepo(t, ctx0, conn, &todev.Repo{Name: "repo1"})

	contributor1 := MustCreateContributor(t, ctx1, conn, &todev.Contributor{RepoID: repo0.ID})

	task := MustCreateTask(t, ctx0, conn, &todev.Task{Description: "Do some stuff.", ContributorID: &contributor1.ID, RepoID: repo0.ID})

	if err := s.DeleteTask(ctx0, task.ID); err != nil {
		t.Fatal(err)

	} else if tasks, _, err := s.FindTasks(ctx, todev.TaskFilter{RepoID: &task.RepoID}); err != nil {
		t.Fatal(err)
	} else if got, want := len(tasks), 0; got != want {
		t.Fatalf("len=%d, want %d", got, want)
	}
}

func deleteTask_Errors(t *testing.T, conn *postgres.Conn) {
	type testData struct {
		ctx      context.Context
		input    int
		expected error
	}
	s := postgres.NewTaskService(conn)

	ctx := context.Background()
	_, ctx0 := MustCreateUser(t, ctx, conn, &todev.User{Name: "bob", Email: "bob@gmail.com"})
	_, ctx1 := MustCreateUser(t, ctx, conn, &todev.User{Name: "judy", Email: "judy@gmail.com"})
	repo0 := MustCreateRepo(t, ctx0, conn, &todev.Repo{Name: "repo1"})

	contributor1 := MustCreateContributor(t, ctx1, conn, &todev.Contributor{RepoID: repo0.ID})

	task := MustCreateTask(t, ctx0, conn, &todev.Task{Description: "Do some stuff.", ContributorID: &contributor1.ID, RepoID: repo0.ID})

	tests := map[string]testData{
		"ErrDeleteNotAllowed": testData{
			ctx:   ctx1,
			input: task.ID,
			expected: &todev.Error{
				Code:    todev.ECONFLICT,
				Message: "You are not allowed to delete tasks.",
			},
		},
		"ErrNotFound": testData{
			ctx:   ctx0,
			input: 2,
			expected: &todev.Error{
				Code:    todev.ENOTFOUND,
				Message: "Task not found.",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if err := s.DeleteTask(tt.ctx, tt.input); err == nil {
				t.Fatal("error expected")
			} else if err.Error() != tt.expected.Error() {
				t.Fatalf("unexpected error: %#v", err)
			}
		})
	}
}

func findTasks_ByRepoID(t testing.TB, conn *postgres.Conn) {
	s := postgres.NewTaskService(conn)

	ctx := context.Background()
	_, ctx0 := MustCreateUser(t, ctx, conn, &todev.User{Name: "bob", Email: "bob@gmail.com"})
	_, ctx1 := MustCreateUser(t, ctx, conn, &todev.User{Name: "judy", Email: "judy@gmail.com"})
	_, ctx2 := MustCreateUser(t, ctx, conn, &todev.User{Name: "george", Email: "george@gmail.com"})
	repo0 := MustCreateRepo(t, ctx0, conn, &todev.Repo{Name: "repo1"})
	repo1 := MustCreateRepo(t, ctx1, conn, &todev.Repo{Name: "repo1"})

	contributor1 := MustCreateContributor(t, ctx1, conn, &todev.Contributor{RepoID: repo0.ID})
	contributor2 := MustCreateContributor(t, ctx2, conn, &todev.Contributor{RepoID: repo1.ID})

	task0 := MustCreateTask(t, ctx0, conn, &todev.Task{Description: "Do some stuff.", ContributorID: &contributor1.ID, RepoID: repo0.ID})
	task1 := MustCreateTask(t, ctx0, conn, &todev.Task{Description: "Do some stuff.", ContributorID: &contributor2.ID, RepoID: repo0.ID})
	task2 := MustCreateTask(t, ctx1, conn, &todev.Task{Description: "Do some stuff.", ContributorID: &contributor2.ID, RepoID: repo1.ID})

	if tasks, n, err := s.FindTasks(ctx0, todev.TaskFilter{RepoID: &repo0.ID}); err != nil {
		t.Fatal(err)
	} else if got, want := len(tasks), 2; got != want {
		t.Fatalf("len=%d, want %d", got, want)
	} else if got, want := n, 2; got != want {
		t.Fatalf("n=%d, want %d", got, want)
	} else if !reflect.DeepEqual(task0, tasks[0]) {
		t.Fatalf("mismatch: %#v !=\n%#v", task0, tasks[0])
	} else if !reflect.DeepEqual(task1, tasks[1]) {
		t.Fatalf("mismatch: %#v !=\n%#v", task1, tasks[1])
	}

	if tasks, n, err := s.FindTasks(ctx1, todev.TaskFilter{RepoID: &repo1.ID}); err != nil {
		t.Fatal(err)
	} else if got, want := len(tasks), 1; got != want {
		t.Fatalf("len=%d, want %d", got, want)
	} else if got, want := n, 1; got != want {
		t.Fatalf("n=%d, want %d", got, want)
	} else if !reflect.DeepEqual(task2, tasks[0]) {
		t.Fatalf("mismatch: %#v !=\n%#v", task2, tasks[0])
	}
}

func findTasks_ByContributorID(t testing.TB, conn *postgres.Conn) {
	s := postgres.NewTaskService(conn)

	ctx := context.Background()
	_, ctx0 := MustCreateUser(t, ctx, conn, &todev.User{Name: "bob", Email: "bob@gmail.com"})
	_, ctx1 := MustCreateUser(t, ctx, conn, &todev.User{Name: "judy", Email: "judy@gmail.com"})
	_, ctx2 := MustCreateUser(t, ctx, conn, &todev.User{Name: "george", Email: "george@gmail.com"})
	repo := MustCreateRepo(t, ctx0, conn, &todev.Repo{Name: "repo"})

	contributor1 := MustCreateContributor(t, ctx1, conn, &todev.Contributor{RepoID: repo.ID})
	contributor2 := MustCreateContributor(t, ctx2, conn, &todev.Contributor{RepoID: repo.ID})

	MustCreateTask(t, ctx0, conn, &todev.Task{Description: "Do some stuff.", ContributorID: &contributor1.ID, RepoID: repo.ID})
	task1 := MustCreateTask(t, ctx0, conn, &todev.Task{Description: "Do some stuff.", ContributorID: &contributor2.ID, RepoID: repo.ID})
	task2 := MustCreateTask(t, ctx0, conn, &todev.Task{Description: "Do some stuff.", ContributorID: &contributor2.ID, RepoID: repo.ID})

	if tasks, n, err := s.FindTasks(ctx0, todev.TaskFilter{ContributorID: &contributor2.ID}); err != nil {
		t.Fatal(err)
	} else if got, want := len(tasks), 2; got != want {
		t.Fatalf("len=%d, want %d", got, want)
	} else if got, want := n, 2; got != want {
		t.Fatalf("n=%d, want %d", got, want)
	} else if !reflect.DeepEqual(task1, tasks[0]) {
		t.Fatalf("mismatch: %#v !=\n%#v", task1, tasks[0])
	} else if !reflect.DeepEqual(task2, tasks[1]) {
		t.Fatalf("mismatch: %#v !=\n%#v", task2, tasks[1])
	}
}

func createTask_OK(t testing.TB, conn *postgres.Conn) {
	s := postgres.NewTaskService(conn)

	ctx := context.Background()
	_, ctx0 := MustCreateUser(t, ctx, conn, &todev.User{Name: "bob", Email: "bob@gmail.com"})
	_, ctx1 := MustCreateUser(t, ctx, conn, &todev.User{Name: "judy", Email: "judy@gmail.com"})
	repo := MustCreateRepo(t, ctx0, conn, &todev.Repo{Name: "repo"})

	contributor1 := MustCreateContributor(t, ctx1, conn, &todev.Contributor{RepoID: repo.ID})

	task := &todev.Task{Description: "Do some stuff.", ContributorID: &contributor1.ID, RepoID: repo.ID}

	if err := s.CreateTask(ctx0, task); err != nil {
		t.Fatal(err)
	}
	// else if len(repo.Tasks) == 0 {
	// 	t.Fatalf("expected task append")
	// }
	// else if !reflect.DeepEqual(task, repo.Tasks[0]) {
	// 	t.Fatalf("mismatch: %#v !=\n%#v", task, repo.Tasks[0])
	// }
}

func createTask_Errors(t *testing.T, conn *postgres.Conn) {
	type testData struct {
		input    *todev.Task
		expected error
		ctx      context.Context
	}

	s := postgres.NewTaskService(conn)

	ctx := context.Background()
	_, ctx0 := MustCreateUser(t, ctx, conn, &todev.User{Name: "bob", Email: "bob@gmail.com"})
	_, ctx1 := MustCreateUser(t, ctx, conn, &todev.User{Name: "judy", Email: "judy@gmail.com"})
	repo0 := MustCreateRepo(t, ctx0, conn, &todev.Repo{Name: "repo"})
	repo1 := MustCreateRepo(t, ctx0, conn, &todev.Repo{Name: "repo"})

	MustCreateContributor(t, ctx1, conn, &todev.Contributor{RepoID: repo0.ID})

	tests := map[string]testData{
		"ErrTaskCreationReject": testData{
			input: &todev.Task{Description: "Do some stuff.", RepoID: repo1.ID},
			expected: &todev.Error{
				Code:    todev.ECONFLICT,
				Message: "Only repo owner can create tasks.",
			},
			ctx: ctx1,
		},
		"ErrRepoIDRequired": testData{
			input: &todev.Task{Description: "Go sleep."},
			expected: &todev.Error{
				Code:    todev.EINVALID,
				Message: "Repo ID required.",
			},
			ctx: ctx0,
		},
		"ErrDescriptionRequired": testData{
			input: &todev.Task{RepoID: repo0.ID},
			expected: &todev.Error{
				Code:    todev.EINVALID,
				Message: "Task description required.",
			},
			ctx: ctx0,
		},
		"ErrDescriptionTooLong": testData{
			input: &todev.Task{Description: strings.Repeat("X", todev.MaxTaskDescriptionLen+1), RepoID: repo0.ID},
			expected: &todev.Error{
				Code:    todev.EINVALID,
				Message: "Task description too long.",
			},
			ctx: ctx0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if err := s.CreateTask(tt.ctx, tt.input); err == nil {
				t.Fatal("error expected")
			} else if err.Error() != tt.expected.Error() {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}

}

func MustCreateTask(tb testing.TB, ctx context.Context, conn *postgres.Conn, task *todev.Task) *todev.Task {
	tb.Helper()
	if err := postgres.NewTaskService(conn).CreateTask(ctx, task); err != nil {
		tb.Fatalf("MustCreateRepo: %v", err)
	}
	return task
}
