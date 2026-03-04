package taskstore

import (
	"neuralclaw/pkg/types"
)

type Store interface {
	SaveTask(task types.Task) error
	GetTask(id string) (types.Task, error)
	ListTasks(scope string) ([]types.Task, error)

	SaveRun(run types.Run) error
	GetRun(id string) (types.Run, error)
	ListRuns(scope string) ([]types.Run, error)
	GetRunsByTask(taskID string) ([]types.Run, error)
}
