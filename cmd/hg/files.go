package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/busoc/hourglass"
	"github.com/gorilla/mux"
)

func listFiles(r *http.Request) (interface{}, error) {
	q := r.URL.Query()
	ds, err := hourglass.ListFiles(db, q.Get("status"), q["category[]"])
	switch {
	case err != nil:
		return ds, err
	case len(ds) == 0:
		return nil, err
	default:
		return ds, err
	}
}

func viewFile(r *http.Request) (interface{}, error) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	return hourglass.ViewFile(db, id, true, true)
}

func newFile(r *http.Request) (interface{}, error) {
	f := new(hourglass.File)
	if err := json.NewDecoder(io.LimitReader(r.Body, MaxBodySize)).Decode(f); err != nil {
		return nil, err
	}
	f.Id, _ = strconv.Atoi(mux.Vars(r)["id"])
	f.User = r.Context().Value("user").(string)
	if err := hourglass.NewFile(db, f); err != nil {
		return nil, err
	}
	return hourglass.ViewFile(db, f.Id, true, true)
}

func updateFile(r *http.Request) (interface{}, error) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	s, err := hourglass.ViewFile(db, id, false, false)
	if err != nil {
		return nil, err
	}
	f := &hourglass.File{
		Id:         id,
		Name:       s.Name,
		Summary:    s.Summary,
		Categories: s.Categories,
		Meta:       s.Meta,
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, MaxBodySize)).Decode(f); err != nil {
		return nil, err
	}
	f.User = r.Context().Value("user").(string)
	if err := hourglass.UpdateFile(db, f); err != nil {
		return nil, err
	}
	return hourglass.ViewFile(db, f.Id, true, true)
}

func deleteFile(r *http.Request) (interface{}, error) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	f, err := hourglass.ViewFile(db, id, false, false)
	if err != nil {
		return nil, err
	}
	f.User = r.Context().Value("user").(string)
	return nil, hourglass.DeleteFile(db, f)
}
