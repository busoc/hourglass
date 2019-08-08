package hourglass

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"time"

	"github.com/lib/pq"
)

const CITT = 0xFFFF

type File struct {
	Id         int                    `json:"uid"`
	Name       string                 `json:"name"`
	Summary    string                 `json:"summary"`
	Content    []byte                 `json:"raw,omitempty"`
	Categories []string               `json:"categories"`
	Meta       map[string]interface{} `json:"metadata"`
	Version    int                    `json:"version"`
	Length     int                    `json:"length"`
	Sum        string                 `json:"sum"`
	Cyclic     uint16                 `json:"crc"`

	Dummy       bool `json:"dummy"`
	Superseeded bool `json:"superseeded"`
	Original    bool `json:"original"`

	User    string    `json:"user"`
	Lastmod time.Time `json:"lastmod"`

	Slot     string `json:"slot"`
	Location string `json:"location"`

	Versions []*File `json:"history,omitempty"`
	Parents  []*File `json:"parents,omitempty"`
}

func ListFiles(db *sql.DB, which string, cs []string) ([]*File, error) {
	const q = `
select
	pk, name, crc, slot, location, summary, categories, meta, version, length, sum, superseeded, original, person, lastmod
from vfiles
	where case when cardinality($1::varchar[])>0 then categories&&$1::varchar[] else true end
		and case when $2='latest' then not superseeded when $2='origin' then original else true end`
	rs, err := db.Query(q, pq.StringArray(cs), which)
	switch err {
	case nil:
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
	defer rs.Close()
	var fs []*File
	for rs.Next() {
		f, err := scanFiles(rs)
		if err != nil {
			return nil, err
		}
		fs = append(fs, f)
	}
	return fs, nil
}

func ViewFile(db *sql.DB, id int, raw, parent bool) (*File, error) {
	const (
		q = `select pk, name, crc, slot, location, summary, categories, meta, version, length, sum, superseeded, original, person, lastmod from vfiles where pk=$1`
		c = `select content from schedule.files where pk=$1 and content is not null and length(content) > 0`
		v = `select pk, name, 0 as crc, slot, location, summary, categories, meta, version, length, sum, superseeded, false, person, lastmod from revisions.vfiles where pk=$1`
	)
	f, err := scanFiles(db.QueryRow(q, id))
	switch err {
	default:
		return nil, err
	case sql.ErrNoRows:
		return nil, ErrNotFound
	case nil:
	}
	if raw {
		if err := db.QueryRow(c, id).Scan(&f.Content); err != nil && f.Length > 0 {
			return nil, err
		}
	}
	rs, err := db.Query(v, id)
	if err != nil {
		return nil, err
	}
	defer rs.Close()

	crc := f.Cyclic
	for rs.Next() {
		f, err := scanFiles(rs)
		if err != nil {
			return nil, err
		}
		f.Cyclic = crc
		f.Versions = append(f.Versions, f)
	}
	if parent {
		f.Parents = listParents(db, id)
	}
	return f, nil
}

func listParents(db *sql.DB, id int) []*File {
	const q = `with recursive others(pk, parent) as
		(select pk, parent from schedule.files where parent is not null union select all fs.pk, rs.parent from schedule.files fs join others rs on rs.pk=fs.parent),
		list(pk) as (select parent from others where pk=$1)
	select pk, name, crc, slot, location, summary, categories, meta, version, length, sum, superseeded, original, person, lastmod from vfiles where pk in (select pk from list);
	`
	rs, err := db.Query(q, id)
	if err != nil {
		return nil
	}
	defer rs.Close()

	var fs []*File
	for rs.Next() {
		f, err := scanFiles(rs)
		if err != nil {
			break
		}
		fs = append(fs, f)
	}
	return fs
}

func NewFile(db *sql.DB, f *File) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if err = createFile(tx, f); err != nil {
		tx.Rollback()
		return err
	}
	if err := linkFile2Categories(tx, f); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func UpdateFile(db *sql.DB, f *File) error {
	const q = `with
		u(pk) as (select pk from vusers where initial=$3)
		update schedule.files set
			name=$1,
			summary=$2,
			person=(select pk from u),
			meta=$4,
			lastmod=current_timestamp
		where pk=$5 and not canceled returning lastmod`
	meta, err := json.Marshal(f.Meta)
	if err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if err := tx.QueryRow(q, f.Name, f.Summary, f.User, meta, f.Id).Scan(&f.Lastmod); err != nil {
		tx.Rollback()
		return err
	}
	if err := linkFile2Categories(tx, f); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func DeleteFile(db *sql.DB, f *File) error {
	const q = `with u(pk) as (select pk from vusers where initial=$1) update schedule.files set canceled=true, person=(select pk from u), lastmod=current_timestamp where pk=$2 and not canceled`
	_, err := db.Exec(q, f.User, f.Id)
	return err
}

func createFile(tx *sql.Tx, f *File) error {
	const q = `with
		u(pk) as (select pk from vusers where initial=$4 limit 1)
		insert into schedule.files(name, summary, content, person, meta, parent, crc)
			values($1, $2, $3, (select pk from u), $5, nullif($6, 0), $7) returning pk`
	m, err := json.Marshal(f.Meta)
	if err != nil {
		return err
	}
	crc, err := calculateCRC(bytes.NewReader(f.Content))
	if err != nil {
		return err
	}
	if err := tx.QueryRow(q, f.Name, f.Summary, f.Content, f.User, m, f.Id, crc).Scan(&f.Id); err != nil {
		return err
	}
	f.Dummy = len(f.Content) == 0
	f.Version += 1

	return nil
}

func linkFile2Categories(tx *sql.Tx, f *File) error {
	const q = `with c(pk) as (select pk from schedule.categories where name=$2) insert into schedule.files_categories(file, category) values($1, (select pk from c))`
	for _, c := range f.Categories {
		if _, err := tx.Exec(q, f.Id, c); err != nil {
			return err
		}
	}
	return nil
}

func scanFiles(s Scanner) (*File, error) {
	var (
		cs   pq.StringArray
		meta []byte
	)
	f := new(File)
	if err := s.Scan(&f.Id, &f.Name, &f.Cyclic, &f.Slot, &f.Location, &f.Summary, &cs, &meta, &f.Version, &f.Length, &f.Sum, &f.Superseeded, &f.Original, &f.User, &f.Lastmod); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(meta, &f.Meta); meta != nil && err != nil {
		return nil, err
	}
	f.Dummy = f.Length == 0
	f.Categories = []string(cs)

	return f, nil
}

func calculateCRC(r io.Reader) (uint16, error) {
	rs := bufio.NewReader(r)

	v := uint16(CITT)
	for {
		b, err := rs.ReadByte()
		switch err {
		case nil:
			x := (v >> 8) ^ uint16(b)
			x ^= x >> 4
			v = (v << 8) ^ (x << 12) ^ (x << 5) ^ x
		case io.EOF:
			return v, nil
		default:
			return v, err
		}
	}
}
