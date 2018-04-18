package hourglass

import (
	"database/sql"
	"errors"
	"io"
	"time"
)

var (
	ErrNotFound        = errors.New("not found")
	ErrUnauthenticated = errors.New("invalid credentials")
	ErrNotSupported    = errors.New("not supported")
	ErrInvalid         = errors.New("invalid")
)

type Error struct {
	Severity string `json:"severity"`
	Source   string `json:"source"`
	Message  string `json:"message"`
}

func (e Error) Error() string {
	return e.Message
}

type Exporter interface {
	Export(io.Writer, string) error
}

type Scanner interface {
	Scan(...interface{}) error
}

type Category struct {
	Id      int       `json:"uid"`
	Name    string    `json:"name"`
	User    string    `json:"user"`
	Lastmod time.Time `json:"lastmod"`
}

func ListCategories(db *sql.DB) ([]*Category, error) {
	const q = `select pk, name, person, lastmod from vcategories`
	rs, err := db.Query(q)
	switch err {
	case nil:
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
	defer rs.Close()
	data := make([]*Category, 0, 100)
	for rs.Next() {
		c := new(Category)
		if err := rs.Scan(&c.Id, &c.Name, &c.User, &c.Lastmod); err != nil {
			return nil, err
		}
		data = append(data, c)
	}
	return data, nil
}

func ViewCategory(db *sql.DB, id int) (*Category, error) {
	const q = `select pk, name, person, lastmod from vcategories where pk=$1`
	c := new(Category)
	err := db.QueryRow(q, id).Scan(&c.Id, &c.Name, &c.User, &c.Lastmod)

	switch err {
	case nil:
		return c, nil
	case sql.ErrNoRows:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}

func NewCategory(db *sql.DB, c *Category) error {
	const q = `with u(pk) as (select pk from vusers where initial=$2) insert into schedule.categories(name, person, parent) values($1, (select pk from u), nullif($3, 0)) returning pk, lastmod`
	if err := db.QueryRow(q, c.Name, c.User, &c.Id).Scan(&c.Id, &c.Lastmod); err != nil {
		return err
	}
	return nil
}

func UpdateCategory(db *sql.DB, c *Category) error {
	const q = `with u(pk) as (select pk from vusers where initial=$2) update schedule.categories set name=$1, person=(select pk from u), lastmod=current_timestamp where pk=$3 and not canceled returning lastmod`
	if err := db.QueryRow(q, c.Name, c.User, c.Id).Scan(&c.Lastmod); err != nil {
		return err
	}
	return nil
}
