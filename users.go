package hourglass

import (
	"database/sql"
	"encoding/json"

	"github.com/lib/pq"
)

type User struct {
	Id        int      `json:"id"`
	First     string   `json:"firstname"`
	Last      string   `json:"lastname"`
	Initial   string   `json:"initial"`
	Email     string   `json:"email"`
	Internal  bool     `json:"internal"`
	Positions []string `json:"positions"`

	Settings map[string]interface{} `json:"settings"`

	Events []*Event `json:"events"`
	Todos  []*Todo  `json:"todos"`
}

func ListUsers(db *sql.DB) ([]*User, error) {
	const q = `select pk, firstname, lastname, initial, email, internal, positions from vusers`
	rs, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rs.Close()

	data := make([]*User, 0, 100)
	for rs.Next() {
		u, err := scanUsers(rs)
		if err != nil {
			return nil, err
		}
		data = append(data, u)
	}
	return data, nil
}

func ViewUser(db *sql.DB, id int) (*User, error) {
	const (
		q = `select pk, firstname, lastname, initial, email, internal, positions from vusers where pk=$1`
		s = `select settings from vusers where pk=$1`
		e = `select pk, source, summary, description, meta, state, version, dtstart, dtend, rtstart, rtend, person, attendees, categories, lastmod from vevents where $1=any(attendees)`
		t = `select pk, summary, state, priority, person, version, meta, categories, assignees, dtstart, dtend, due, lastmod from vtodos where $1=any(assignees)`
	)
	u, err := scanUsers(db.QueryRow(q, id))
	switch err {
	default:
		return nil, err
	case sql.ErrNoRows:
		return nil, ErrNotFound
	case nil:
	}
	var bs []byte
	if err := db.QueryRow(s, id).Scan(&bs); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(bs, &u.Settings); err != nil && len(bs) > 0 {
		return nil, err
	}

	var rs *sql.Rows

	rs, err = db.Query(e, u.Initial)
	if err != nil {
		return nil, err
	}
	u.Events, err = listEvents(rs)
	if err != nil {
		return nil, err
	}
	rs, err = db.Query(t, u.Initial)
	if err != nil {
		return nil, err
	}
	u.Todos, err = listTodos(rs)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func UpdateUser(db *sql.DB, u *User) (*User, error) {
	const q = `update usoc.persons set
		firstname=$1,
		lastname=$2,
		initial=$3,
		email=$4,
		internal=$5,
		settings=$6
		where pk=$7 and passwd is not null`
	bs, err := json.Marshal(u.Settings)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(q, u.First, u.Last, u.Initial, u.Email, u.Internal, bs, u.Id); err != nil {
		return nil, err
	}
	return ViewUser(db, u.Id)
}

func UpdatePasswd(db *sql.DB, u *User, old, passwd string) error {
	const q = `update usoc.persons set passwd=encode(digest($1, 'sha256'), 'hex') where pk=$2 and passwd=encode(digest($3, 'sha256'), 'hex')`
	_, err := db.Exec(q, passwd, u.Id, old)
	return err
}

func RegisterUser(db *sql.DB, u *User, passwd string) error {
	const q = `insert into usoc.persons(firstname, lastname, initial, email, internal, passwd) values($1, $2, $3, $4, $5, encode(digest($6, 'sha256'), 'hex')) returning pk`
	if err := db.QueryRow(q, u.First, u.Last, u.Initial, u.Email, u.Internal, passwd).Scan(&u.Id); err != nil {
		return err
	}
	return nil
}

func Authenticate(db *sql.DB, i, p string) (*User, error) {
	const q = `select pk, firstname, lastname, initial, email, internal, positions from vusers where (initial=$1 or email=$1) and passwd=encode(digest($2, 'sha256'), 'hex')`
	u, err := scanUsers(db.QueryRow(q, i, p))
	switch err {
	default:
		return nil, err
	case sql.ErrNoRows:
		return nil, ErrUnauthenticated
	case nil:
		return u, nil
	}
}

func scanUsers(s Scanner) (*User, error) {
	u := new(User)
	var ps pq.StringArray
	if err := s.Scan(&u.Id, &u.First, &u.Last, &u.Initial, &u.Email, &u.Internal, &ps); err != nil {
		return nil, err
	}
	u.Positions = []string(ps)
	return u, nil
}
