package hourglass

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
)

type Slot struct {
	Id       int       `json:"uid"`
	Name     string    `json:"name"`
	Category string    `json:"category"`
	User     string    `json:"user"`
	Lastmod  time.Time `json:"lastmod"`

	File  string `json:"file"`
	State string `json:"status"`

	Uplinks []*Uplink `json:"uplinks,omitempty"`
}

func ListSlots(db *sql.DB, cs []string) ([]*Slot, error) {
	const q = `select sid, name, category, person, file, state, lastmod from vslots where case when cardinality($1::varchar[])>0 then category = any($1::varchar[]) else true end`
	rs, err := db.Query(q, pq.StringArray(cs))
	if err != nil {
		return nil, err
	}
	defer rs.Close()

	data := make([]*Slot, 0, 100)
	for rs.Next() {
		s, err := scanSlots(rs)
		if err != nil {
			return nil, err
		}
		data = append(data, s)
	}
	return data, nil
}

func ViewSlot(db *sql.DB, id int) (*Slot, error) {
	s, err := viewSlot(db, id)
	if err != nil {
		return nil, err
	}
	const q = `select pk, state, person, lastmod, slot, event, file from schedule.uplinks where slot=$1`

	rs, err := db.Query(q, id)
	if err != nil {
		return nil, err
	}
	defer rs.Close()
	data := make([]*Uplink, 0, 100)
	for rs.Next() {
		u := new(Uplink)
		var s, e, f int
		if err := rs.Scan(&u.Id, &u.Status, &u.User, &u.Lastmod, &s, &e, &f); err != nil {
			return nil, err
		}
		data = append(data, u)
	}
	s.Uplinks = data
	return s, nil
}

func viewSlot(db *sql.DB, id int) (*Slot, error) {
	const q = `select sid, name, category, person, file, state, lastmod from vslots where sid=$1`
	s, err := scanSlots(db.QueryRow(q, id))
	switch err {
	default:
		return nil, err
	case sql.ErrNoRows:
		return nil, ErrNotFound
	case nil:
		return s, nil
	}
}

func NewSlot(db *sql.DB, s *Slot) error {
	const q = `with c(pk) as (select pk from schedule.categories where name=$3), u(pk) as (select pk from vusers where initial=$4) insert into schedule.slots(pk, name, category, person) values($1, $2, (select pk from c), (select pk from u))`
	_, err := db.Exec(q, s.Id, s.Name, s.Category, s.User)
	return err
}

func DeleteSlot(db *sql.DB, s *Slot) error {
	const q = `with u(pk) as (select pk from vusers where initial=$2) update schedule.slots set canceled=true, person=(select pk from u), lastmod=current_timestamp where pk=$1 and not canceled`
	_, err := db.Exec(q, s.Id, s.User)
	return err
}

func scanSlots(s Scanner) (*Slot, error) {
	i := new(Slot)
	if err := s.Scan(&i.Id, &i.Name, &i.Category, &i.User, &i.File, &i.State, &i.Lastmod); err != nil {
		return nil, err
	}
	return i, nil
}
