package hourglass

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

type Event struct {
	Id          int                    `json:"uid"`
	Summary     string                 `json:"summary"`
	Description string                 `json:"description"`
	Source      string                 `json:"source"`
	State       string                 `json:"status"`
	Meta        map[string]interface{} `json:"metadata"`
	Starts      time.Time              `json:"dtstart"`
	Ends        time.Time              `json:"dtend"`
	ExStarts    time.Time              `json:"rtstart"`
	ExEnds      time.Time              `json:"rtend"`
	User        string                 `json:"user"`
	Attendees   []string               `json:"attendees"`
	Categories  []string               `json:"categories"`
	Version     int                    `json:"version"`
	Lastmod     time.Time              `json:"lastmod"`
	Attachment  *File                  `json:"attachment"`
	Events      []*Event               `json:"events,omitempty"`

	Versions []*Event `json:"history,omitempty"`
}

func ListSources(db *sql.DB) ([]string, error) {
	const q = `select distinct source from vevents where source is not null`
	rs, err := db.Query(q)
	if err != nil {
		return nil, nil
	}
	defer rs.Close()

	vs := make([]string, 0, 10)
	for rs.Next() {
		var v string
		if err := rs.Scan(&v); err != nil {
			return nil, err
		}
		vs = append(vs, v)
	}
	return vs, nil
}

func ListEvents(db *sql.DB, f, t time.Time, cs, vs []string) ([]*Event, error) {
	if f.IsZero() && t.IsZero() {
		f = time.Now().Truncate(time.Hour * 24)
		t = f.Add(time.Hour * 24)
	}
	const q = `select
			pk, source, summary, description, meta, state, version, dtstart, dtend, rtstart, rtend, person, attendees, categories, lastmod
		from vevents
		where
			(dtstart between $1 and $2 or ($1, $2) overlaps(dtstart, dtend))
			and case when cardinality($3::varchar[])>0 then categories&&$3::varchar[] else true end
			and case when cardinality($4::varchar[])>0 then source=any($4) else source='' end`
	rs, err := db.Query(q, f.UTC(), t.UTC(), pq.StringArray(cs), pq.StringArray(vs))
	if err != nil {
		return nil, err
	}
	return listEvents(rs)
}

func ViewEvent(db *sql.DB, id int) (*Event, error) {
	const (
		q = `select pk, source, summary, description, meta, state, version, dtstart, dtend, rtstart, rtend, person, attendees, categories, lastmod from vevents where pk=$1`
		h = `select pk, source, summary, description, meta, state, version, dtstart, dtend, rtstart, rtend, person, attendees, categories, lastmod from vevents where parent=$1`
		v = `select pk, source, summary, description, meta, state, version, dtstart, dtend, rtstart, rtend, person, attendees, categories, lastmod from revisions.vevents where pk=$1`
	)
	e, err := scanEvents(db.QueryRow(q, id))
	switch err {
	default:
		return nil, err
	case sql.ErrNoRows:
		return nil, ErrNotFound
	case nil:
	}

	var rs *sql.Rows

	rs, err = db.Query(h, e.Id)
	if err != nil {
		return nil, err
	}
	if e.Events, err = listEvents(rs); err != nil {
		return nil, err
	}

	rs, err = db.Query(v, e.Id)
	if err != nil {
		return nil, err
	}
	if e.Versions, err = listEvents(rs); err != nil {
		return nil, err
	}

	return e, nil
}

func ImportEvents(db *sql.DB, s string, es []*Event) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	var fd, td time.Time
	for _, e := range es {
		if e.Source == "" {
			e.Source = s
		}
		if fd.IsZero() || e.Starts.Before(fd) {
			fd = e.Starts
		}
		if td.IsZero() || e.Ends.After(td) {
			td = e.Ends
		}
		if err := createEvent(tx, e); err != nil {
			tx.Rollback()
			return err
		}
		if err := linkEvent2Categories(tx, e); err != nil {
			tx.Rollback()
			return err
		}
	}
	const q = `with q(pk) as (delete from schedule.events where source=$1 and dtstart between $2 and $3 and lastmod<now() returning pk) delete from schedule.events_categories where event in (select pk from q)`
	if _, err := db.Exec(q, s, fd, td); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func NewEvent(db *sql.DB, e *Event) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if err := createEvent(tx, e); err != nil {
		tx.Rollback()
		return err
	}
	if err := linkEvent2Categories(tx, e); err != nil {
		tx.Rollback()
		return err
	}
	if err := linkEvent2Attendees(tx, e); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func UpdateEvent(db *sql.DB, e *Event) error {
	const q = `
		with
			u(pk) as (select pk from vusers where initial=$6)
		update schedule.events set summary=$1, description=$2, dtstart=$3, dtend=$4, meta=$5, person=(select pk from u), state=$7, rtstart=$8, rtend=$9, lastmod=current_timestamp where pk=$10 and source is null returning lastmod`
	if e.ExStarts.IsZero() {
		e.ExStarts = e.Starts
	}
	if e.ExEnds.IsZero() {
		e.ExEnds = e.Ends
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	m, err := json.Marshal(e.Meta)
	if err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.QueryRow(q, e.Summary, e.Description, e.Starts.UTC(), e.Ends.UTC(), m, e.User, e.State, e.ExStarts.UTC(), e.ExEnds.UTC(), e.Id).Scan(&e.Lastmod); err != nil {
		tx.Rollback()
		return err
	}

	if err := linkEvent2Categories(tx, e); err != nil {
		tx.Rollback()
		return err
	}
	if err := linkEvent2Attendees(tx, e); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func DeleteEvent(db *sql.DB, e *Event) error {
	const q = `with u(pk) as (select pk from vusers where initial=$2) update schedule.events set canceled=true, person=(select pk from u), lastmod=current_timestamp where pk=$1 and source is null`
	_, err := db.Exec(q, e.Id, e.User)
	return err
}

func listEvents(rs *sql.Rows) ([]*Event, error) {
	defer rs.Close()

	data := make([]*Event, 0, 100)
	for rs.Next() {
		e, err := scanEvents(rs)
		if err != nil {
			return nil, err
		}
		data = append(data, e)
	}
	return data, nil
}

func scanEvents(s Scanner) (*Event, error) {
	var (
		cs, as pq.StringArray
		m      []byte
	)

	e := new(Event)
	if err := s.Scan(&e.Id, &e.Source, &e.Summary, &e.Description, &m, &e.State, &e.Version, &e.Starts, &e.Ends, &e.ExStarts, &e.ExEnds, &e.User, &as, &cs, &e.Lastmod); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(m, &e.Meta); err != nil && m != nil {
		return nil, err
	}

	e.Starts = e.Starts.UTC()
	e.Ends = e.Ends.UTC()
	e.ExStarts = e.ExStarts.UTC()
	e.ExEnds = e.ExEnds.UTC()
	e.Categories = []string(cs)
	e.Attendees = []string(as)

	return e, nil
}

func createEvent(tx *sql.Tx, e *Event) error {
	const q = `
		with
			u(pk) as (select pk from vusers where initial=$7)
		insert into schedule.events(summary, description, source, dtstart, dtend, meta, person, parent)
			values($1, nullif($2, ''), nullif($3, ''), $4, $5, $6, (select pk from u), nullif($8, 0)) returning pk`
	m, err := json.Marshal(e.Meta)
	if err != nil {
		return err
	}
	r := tx.QueryRow(q, e.Summary, e.Description, e.Source, e.Starts.UTC(), e.Ends.UTC(), m, e.User, e.Id)
	return r.Scan(&e.Id)
}

func linkEvent2Categories(tx *sql.Tx, e *Event) error {
	const (
		r = `insert into schedule.categories(name) values($1) on conflict(name) do update set name=$1 returning pk`
		s = `insert into schedule.events_categories(event, category) values($1, $2) on conflict do nothing`
	)
	for _, c := range e.Categories {
		var cid int
		if err := tx.QueryRow(r, c).Scan(&cid); err != nil {
			return err
		}
		if _, err := tx.Exec(s, e.Id, cid); err != nil {
			return err
		}
	}
	return nil
}

func linkEvent2Attendees(tx *sql.Tx, e *Event) error {
	const q = `with u(pk) as (select pk from vusers where initial=$2) insert into schedule.attendees(event, person) values($1, (select pk from u)) on conflict do nothing`
	for _, a := range e.Attendees {
		if _, err := tx.Exec(q, e.Id, a); err != nil {
			return err
		}
	}
	return nil
}
