package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/busoc/hourglass"
	"github.com/gorilla/mux"
)

func listTodos(r *http.Request) (interface{}, error) {
	q := r.URL.Query()
	ds, err := hourglass.ListTodos(db, q["category[]"])
	switch {
	case err != nil:
		return ds, err
	case len(ds) == 0:
		return nil, err
	default:
		return ds, err
	}
}

func viewTodo(r *http.Request) (interface{}, error) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	return hourglass.ViewTodo(db, id)
}

func newTodo(r *http.Request) (interface{}, error) {
	t := new(hourglass.Todo)
	if err := json.NewDecoder(io.LimitReader(r.Body, MaxBodySize)).Decode(t); err != nil {
		return nil, err
	}
	t.Id, _ = strconv.Atoi(mux.Vars(r)["id"])
	t.User = r.Context().Value("user").(string)
	if err := hourglass.NewTodo(db, t); err != nil {
		return nil, err
	}
	return hourglass.ViewTodo(db, t.Id)
}

func updateTodo(r *http.Request) (interface{}, error) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	s, err := hourglass.ViewTodo(db, id)
	if err != nil {
		return nil, err
	}
	t := &hourglass.Todo{
		Id:          id,
		Summary:     s.Summary,
		Description: s.Description,
		Meta:        s.Meta,
		State:       s.State,
		Priority:    s.Priority,
		Due:         s.Due,
		Starts:      s.Starts,
		Ends:        s.Ends,
		Categories:  s.Categories,
		Assignees:   s.Assignees,
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, MaxBodySize)).Decode(t); err != nil {
		return nil, err
	}
	t.User = r.Context().Value("user").(string)
	if err := hourglass.UpdateTodo(db, t); err != nil {
		return nil, err
	}
	return hourglass.ViewTodo(db, t.Id)
}

func deleteTodo(r *http.Request) (interface{}, error) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	t, err := hourglass.ViewTodo(db, id)
	if err != nil {
		return nil, err
	}
	t.User = r.Context().Value("user").(string)
	return nil, hourglass.DeleteTodo(db, t)
}
