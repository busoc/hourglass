package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/busoc/hourglass"
	"github.com/gorilla/mux"
)

func listJournals(r *http.Request) (interface{}, error) {
	var fd, td time.Time
	q := r.URL.Query()
	if q.Get("dtstart") != "" || q.Get("dtend") != "" {
		var err error
		if fd, err = time.Parse(time.RFC3339, q.Get("dtstart")); err != nil {
			return nil, fmt.Errorf("dtstart bad format")
		}
		if td, err = time.Parse(time.RFC3339, q.Get("dtend")); err != nil {
			return nil, fmt.Errorf("dtend bad format")
		}
	}

	ds, err := hourglass.ListJournals(db, fd, td, q["category[]"])
	switch {
	case err != nil:
		return ds, err
	case len(ds) == 0:
		return nil, err
	default:
		return ds, err
	}
}

func viewJournal(r *http.Request) (interface{}, error) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	return hourglass.ViewJournal(db, id)
}

func newJournal(r *http.Request) (interface{}, error) {
	var j hourglass.Journal
	if err := json.NewDecoder(io.LimitReader(r.Body, MaxBodySize)).Decode(&j); err != nil {
		return nil, err
	}
	j.User = r.Context().Value("user").(string)
	if err := hourglass.NewJournal(db, &j); err != nil {
		return nil, err
	}
	return hourglass.ViewJournal(db, j.Id)
}

func updateJournal(r *http.Request) (interface{}, error) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	s, err := hourglass.ViewJournal(db, id)
	if err != nil {
		return nil, err
	}
	j := hourglass.Journal{
		Id:         id,
		Summary:    s.Summary,
		Day:        s.Day,
		State:      s.State,
		Meta:       s.Meta,
		Categories: s.Categories,
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, MaxBodySize)).Decode(&j); err != nil {
		return nil, err
	}
	j.User = r.Context().Value("user").(string)
	if err := hourglass.UpdateJournal(db, &j); err != nil {
		return nil, err
	}
	return hourglass.ViewJournal(db, j.Id)
}

func deleteJournal(r *http.Request) (interface{}, error) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	j, err := hourglass.ViewJournal(db, id)
	if err != nil {
		return nil, err
	}
	j.User = r.Context().Value("user").(string)
	return nil, hourglass.DeleteJournal(db, j)
}
