package hourglass

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

type Journal struct {
	Id         int                    `json:"uid"`
	Day        time.Time              `json:"dtstamp"`
	Summary    string                 `json:"summary"`
	User       string                 `json:"user"`
	State      string                 `json:"status"`
	Lastmod    time.Time              `json:"lastmod"`
	Meta       map[string]interface{} `json:"metadata"`
	Categories []string               `json:"categories"`
}

func ListJournals(db *sql.DB, f, t time.Time, cs []string) ([]*Journal, error) {
	if f.IsZero() && t.IsZero() {
		f = time.Now().Truncate(time.Hour * 24)
		t = f.Add(time.Hour * 24)
	}
	const q = `select pk, day, summmary, meta, state, lastmod, person, categories from vjournals where day between $1 and $2 and case when cardinality($3::varchar[])>0 then categories&&$3::varchar[] else true end`
	rs, err := db.Query(q, f, t, pq.StringArray(cs))
	switch err {
	case nil:
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
	return listJournals(rs)
}

func ViewJournal(db *sql.DB, id int) (*Journal, error) {
	const q = `select pk, day, summmary, meta, state, lastmod, person, categories from vjournals where pk=$1`
	j, err := scanJournals(db.QueryRow(q, id))
	switch err {
	default:
		return nil, err
	case sql.ErrNoRows:
		return nil, ErrNotFound
	case nil:
		return j, err
	}
}

func NewJournal(db *sql.DB, j *Journal) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if err := createJournal(tx, j); err != nil {
		tx.Rollback()
		return err
	}
	if err := linkJournal2Categories(tx, j); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func listJournals(rs *sql.Rows) ([]*Journal, error) {
	defer rs.Close()

	var js []*Journal
	for rs.Next() {
		j, err := scanJournals(rs)
		if err != nil {
			return nil, err
		}
		js = append(js, j)
	}
	return js, nil
}

func scanJournals(s Scanner) (*Journal, error) {
	var (
		j  Journal
		cs pq.StringArray
		m  []byte
	)
	if err := s.Scan(&j.Id, &j.Day, &j.Summary, &m, &j.State, &j.Lastmod, &j.User, &cs); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(m, &j.Meta); err != nil && m != nil {
		return nil, err
	}
	j.Categories = []string(cs)
	return &j, nil
}

func createJournal(tx *sql.Tx, j *Journal) error {
	const q = `
  with
    u(pk) as (select pk from vusers where initial=$5)
  insert int schedule.journals(day, summary, meta, state, person) values($1, $2, $3, $4, (select pk from u)) returning pk, lastmod
  `
	m, err := json.Marshal(j.Meta)
	if err != nil {
		return err
	}
	r := tx.QueryRow(q, j.Day, j.Summary, m, j.State, j.User)
	return r.Scan(&j.Id, &j.Lastmod)
}

func linkJournal2Categories(tx *sql.Tx, j *Journal) error {
	const (
		r = `insert into schedule.categories(name) values($1) on conflict(name) do update set name=$1 returning pk`
		s = `insert into schedule.journals_categories(journal, category) values($1, $2) on conflict do nothing`
	)
	for _, c := range j.Categories {
		var cid int
		if err := tx.QueryRow(r, c).Scan(&cid); err != nil {
			return err
		}
		if _, err := tx.Exec(s, j.Id, cid); err != nil {
			return err
		}
	}
	return nil
}
