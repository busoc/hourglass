package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"hourglass"
)

func listEvents(r *http.Request) (interface{}, error) {
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

	ds, err := hourglass.ListEvents(db, fd, td, q["category[]"])
	switch {
	case err != nil:
		return ds, err
	case len(ds) == 0:
		return nil, err
	default:
		return ds, err
	}
}

func importEvents(r *http.Request) (interface{}, error) {
	v := struct {
		Source string             `json:"source"`
		Events []*hourglass.Event `json:"events"`
	}{}
	s := mux.Vars(r)["source"]
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return nil, err
	}
	if v.Source == "" {
		v.Source = s
	}
	return nil, hourglass.ImportEvents(db, v.Source, v.Events)
}

func newEvent(r *http.Request) (interface{}, error) {
	e := new(hourglass.Event)
	if err := json.NewDecoder(io.LimitReader(r.Body, MaxBodySize)).Decode(e); err != nil {
		return nil, err
	}
	e.Id, _ = strconv.Atoi(mux.Vars(r)["id"])
	e.User = r.Context().Value("user").(string)
	if err := hourglass.NewEvent(db, e); err != nil {
		return nil, err
	}
	return hourglass.ViewEvent(db, e.Id)
}

func viewEvent(r *http.Request) (interface{}, error) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	return hourglass.ViewEvent(db, id)
}

func updateEvent(r *http.Request) (interface{}, error) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	s, err := hourglass.ViewEvent(db, id)
	if err != nil {
		return nil, err
	}
	e := &hourglass.Event{
		Summary:     s.Summary,
		Description: s.Description,
		Categories:  s.Categories,
		Attendees:   s.Attendees,
		State:       s.State,
		Starts:      s.Starts,
		Ends:        s.Ends,
		ExStarts:    s.ExStarts,
		ExEnds:      s.ExEnds,
		Meta:        s.Meta,
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, MaxBodySize)).Decode(e); err != nil {
		return nil, err
	}
	e.User = r.Context().Value("user").(string)
	e.Id = id
	if err := hourglass.UpdateEvent(db, e); err != nil {
		return nil, err
	}
	return hourglass.ViewEvent(db, e.Id)
}

func deleteEvent(r *http.Request) (interface{}, error) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	e, err := hourglass.ViewEvent(db, id)
	if err != nil {
		return nil, err
	}
	return nil, hourglass.DeleteEvent(db, e)
}
