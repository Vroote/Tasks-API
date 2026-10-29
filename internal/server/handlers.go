package server

import (
	"context"
	"errors"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/example/tasks-api/internal/oapi"
	"github.com/example/tasks-api/internal/store"
)

type Handlers struct {
	store *store.MemoryStore
}

var _ oapi.StrictServerInterface = (*Handlers)(nil)

func (h *Handlers) ListTasks(_ context.Context, req oapi.ListTasksRequestObject) (oapi.ListTasksResponseObject, error) {
	limit, offset, status := 20, 0, ""
	if req.Params.Limit != nil {
		limit = *req.Params.Limit
	}
	if req.Params.Offset != nil {
		offset = *req.Params.Offset
	}
	if req.Params.Status != nil {
		status = string(*req.Params.Status)
	}

	tasks, total := h.store.List(status, limit, offset)

	items := make([]oapi.Task, 0, len(tasks))
	for _, t := range tasks {
		items = append(items, toAPI(t))
	}
	return oapi.ListTasks200JSONResponse{Items: items, Total: total}, nil
}

func (h *Handlers) CreateTask(_ context.Context, req oapi.CreateTaskRequestObject) (oapi.CreateTaskResponseObject, error) {
	task := store.Task{
		Title:  req.Body.Title,
		Status: string(oapi.Todo),
	}
	if req.Body.Description != nil {
		task.Description = *req.Body.Description
	}
	if req.Body.Status != nil {
		task.Status = string(*req.Body.Status)
	}
	if req.Body.DueDate != nil {
		task.DueDate = &req.Body.DueDate.Time
	}

	task = h.store.Create(task)

	return oapi.CreateTask201JSONResponse{
		Body:    toAPI(task),
		Headers: oapi.CreateTask201ResponseHeaders{Location: "/api/v1/tasks/" + task.ID.String()},
	}, nil
}

func (h *Handlers) GetTask(_ context.Context, req oapi.GetTaskRequestObject) (oapi.GetTaskResponseObject, error) {
	task, err := h.store.Get(req.Id)
	if errors.Is(err, store.ErrNotFound) {
		return oapi.GetTask404JSONResponse{NotFoundJSONResponse: oapi.NotFoundJSONResponse{Message: "task not found"}}, nil
	}
	if err != nil {
		return nil, err
	}
	return oapi.GetTask200JSONResponse(toAPI(task)), nil
}

func toAPI(t store.Task) oapi.Task {
	res := oapi.Task{
		Id:        t.ID,
		Title:     t.Title,
		Status:    oapi.TaskStatus(t.Status),
		CreatedAt: t.CreatedAt,
	}
	if t.Description != "" {
		res.Description = &t.Description
	}
	if t.DueDate != nil {
		res.DueDate = &openapi_types.Date{Time: *t.DueDate}
	}
	return res
}
