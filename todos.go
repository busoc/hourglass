package hourglass

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

type Todo struct {
	Id          int                    `json:"uid"`
	Summary     string                 `json:"summary"`
	Description string                 `json:"description"`
	State       string                 `json:"status"`
	Priority    string                 `json:"priority"`
	Meta        map[string]interface{} `json:"metadata"`
	Version     int                    `json:"version"`
	Due         time.Time              `json:"due"`
	Starts      time.Time              `json:"dtstart"`
	Ends        time.Time              `json:"dtend"`
	Lastmod     time.Time              `json:"lastmod"`
	User        string                 `json:"user"`
	Categories  []string               `json:"categories"`
	Assignees   []string               `json:"assignees"`

	Todos    []*Todo `json:"todos,omitempty"`
	Versions []*Todo `json:"history,omitempty"`
}

func ListTodos(db *sql.DB, cs []string) ([]*Todo, error) {
	const q = `select pk, summary, description, state, priority, person, version, meta, categories, assignees, dtstart, dtend, due, lastmod from vtodos where case when cardinality($1::varchar[])>0 then categories&&$1::varchar[] else true end`
	rs, err := db.Query(q, pq.StringArray(cs))
	if err != nil {
		return nil, err
	}
	return listTodos(rs)
}

func ViewTodo(db *sql.DB, id int) (*Todo, error) {
	const (
		q = `select pk, summary, description, state, priority, person, version, meta, categories, assignees, dtstart, dtend, due, lastmod from vtodos where pk=$1`
		h = `select pk, summary, description, state, priority, person, version, meta, categories, assignees, dtstart, dtend, due, lastmod from vtodos where parent=$1`
		v = `select pk, summary, description, state, priority, person, version, meta, categories, assignees, dtstart, dtend, due, lastmod from revisions.vtodos where pk=$1`
	)
	t, err := scanTodos(db.QueryRow(q, id))
	switch err {
	default:
		return nil, err
	case sql.ErrNoRows:
		return nil, ErrNotFound
	case nil:
	}

	var rs *sql.Rows

	rs, err = db.Query(h, id)
	if err != nil {
		return nil, err
	}
	t.Todos, err = listTodos(rs)
	if err != nil {
		return nil, err
	}

	rs, err = db.Query(v, id)
	if err != nil {
		return nil, err
	}
	t.Versions, err = listTodos(rs)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func NewTodo(db *sql.DB, t *Todo) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if err := createTodo(tx, t); err != nil {
		tx.Rollback()
		return err
	}
	if err := linkTodo2Categories(tx, t); err != nil {
		tx.Rollback()
		return err
	}
	if err := linkTodo2Assignees(tx, t); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func UpdateTodo(db *sql.DB, t *Todo) error {
	const q = `with u(pk) as (select pk from vusers where initial=$8) update schedule.todos set summary=$1, description=$2, state=$3, priority=$4, due=$5, dtstart=$6, dtend=$7, person=(select pk from u), meta=$9, lastmod=current_timestamp where pk=$10 returning lastmod`
	m, err := json.Marshal(t.Meta)
	if err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if err := tx.QueryRow(q, t.Summary, t.Description, t.State, t.Priority, t.Due, t.Starts, t.Ends, t.User, m, t.Id).Scan(&t.Lastmod); err != nil {
		tx.Rollback()
		return err
	}
	if err := linkTodo2Categories(tx, t); err != nil {
		tx.Rollback()
		return err
	}
	if err := linkTodo2Assignees(tx, t); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func DeleteTodo(db *sql.DB, t *Todo) error {
	const q = `with u(pk) as (select pk from vusers where initial=$2) update schedule.todos set canceled=true, lastmod=current_timestamp, person=(select pk from u) where pk=$1`
	_, err := db.Exec(q, t.Id, t.User)
	return err
}

func createTodo(tx *sql.Tx, t *Todo) error {
	const q = `with u(pk) as
		(select pk from vusers where initial=$8)
		insert into schedule.todos(summary, description, state, priority, due, meta, parent, person)
			values($1, nullif($2, ''), $3, $4, $5, $6, nullif($7, 0), (select pk from u)) returning pk, lastmod`
	m, err := json.Marshal(t.Meta)
	if err != nil {
		return err
	}
	return tx.QueryRow(q, t.Summary, t.Description, t.State, t.Priority, t.Due, m, t.Id, t.User).Scan(&t.Id, &t.Lastmod)
}

func linkTodo2Categories(tx *sql.Tx, t *Todo) error {
	const q = ` with c(pk) as (select pk from schedule.categories where name=$2) insert into schedule.todos_categories(todo, category) values($1, (select pk from c))`
	for _, c := range t.Categories {
		if _, err := tx.Exec(q, t.Id, c); err != nil {
			return err
		}
	}
	return nil
}

func linkTodo2Assignees(tx *sql.Tx, t *Todo) error {
	const q = ` with u(pk) as (select pk from vusers where initial=$2) insert into schedule.assignees(todo, person) values($1, (select pk from u))`
	for _, a := range t.Assignees {
		if _, err := tx.Exec(q, t.Id, a); err != nil {
			return err
		}
	}
	return nil
}

func listTodos(rs *sql.Rows) ([]*Todo, error) {
	defer rs.Close()

	data := make([]*Todo, 0, 100)
	for rs.Next() {
		t, err := scanTodos(rs)
		if err != nil {
			return nil, err
		}
		data = append(data, t)
	}
	return data, nil
}

func scanTodos(s Scanner) (*Todo, error) {
	t := new(Todo)

	var (
		cs, as pq.StringArray
		fd, td pq.NullTime
		m      []byte
	)
	if err := s.Scan(&t.Id, &t.Summary, &t.Description, &t.State, &t.Priority, &t.User, &t.Version, &m, &cs, &as, &fd, &td, &t.Due, &t.Lastmod); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(m, &t.Meta); len(m) > 0 && err != nil {
		return nil, err
	}
	t.Categories = []string(cs)
	t.Assignees = []string(as)
	if fd.Valid {
		t.Starts = fd.Time
	}
	if td.Valid {
		t.Ends = td.Time
	}

	return t, nil
}
