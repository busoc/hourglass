package hourglass

import (
	"database/sql"
	"time"

	"github.com/lib/pq"
)

type Uplink struct {
	Id      int       `json:"uid"`
	Name    string    `json:"dropbox"`
	Status  string    `json:"status"`
	User    string    `json:"user"`
	Lastmod time.Time `json:"lastmod"`

	*Slot  `json:"slot"`
	*Event `json:"event"`
	*File  `json:"file"`
}

type Transfer struct {
	Id       int       `json:"uid"`
	Status   string    `json:"status"`
	User     string    `json:"user"`
	Location string    `json:"location"`
	Lastmod  time.Time `json:"lastmod"`

	*Event `json:"event"`
	*File  `json:"file"`
	*Slot  `json:"slot"`
}

func ListDownlinks(db *sql.DB, fd, td time.Time, cs, ts []string) ([]*Uplink, error) {
	const q = `
		select
			pk, '', state, person, lastmod, slot, event, file
		from vdownlinks
		where
			dtstamp between $1 and $2
			and case when cardinality($3::usoc.status[]) > 0 then state=any($3::usoc.status[]) else true end
			and case when cardinality($4::text[]) > 0 then category=any($4::text[]) else true end`
	rs, err := db.Query(q, fd, td, pq.StringArray(ts), pq.StringArray(cs))
	switch err {
	case nil:
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
	data := make([]*Uplink, 0, 100)
	for rs.Next() {
		u, err := scanUplink(rs, db)
		if err != nil {
			return nil, err
		}
		data = append(data, u)
	}
	if len(data) == 0 {
		return nil, nil
	}
	return data, nil
}

func ViewDownlink(db *sql.DB, id int) (*Uplink, error) {
	const q = `select pk, '', state, person, lastmod, slot, event, file from vdownlinks where pk=$1`
	return scanUplink(db.QueryRow(q, id), db)
}

func NewDownlink(db *sql.DB, e, s, f int, u string) (*Uplink, error) {
	return createUplink(db, s, e, f, u, true)
}

func UpdateDownlink(db *sql.DB, id int, s, u string) (*Uplink, error) {
	if err := updateUplink(db, id, s, u); err != nil {
		return nil, err
	}
	return ViewDownlink(db, id)
}

func ListTransfers(db *sql.DB, fd, td time.Time, cs, ts []string) ([]*Transfer, error) {
	const q = `
		select
			pk, state, person, location, lastmod, event, file, slot
		from vtransfers
		where
			dtstamp between $1 and $2
			and state=any($3::usoc.status[])
			and category=any($4::text[])`
	rs, err := db.Query(q, fd, td, pq.StringArray(ts), pq.StringArray(cs))
	switch err {
	case nil:
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
	defer rs.Close()
	data := make([]*Transfer, 0, 100)
	for rs.Next() {
		t := new(Transfer)
		var e, s, f int
		if err := rs.Scan(&t.Id, &t.Status, &t.User, &t.Location, &t.Lastmod, &e, &f, &s); err != nil {
			return nil, err
		}
		t.Event, _ = ViewEvent(db, e)
		t.Slot, _ = viewSlot(db, s)
		t.File, _ = ViewFile(db, f)
		data = append(data, t)
	}
	return data, nil
}

func ViewTransfer(db *sql.DB, id int) (*Transfer, error) {
	const q = `select pk, state, person, location, lastmod, event, file, slot from vtransfers where pk=$1`

	t := new(Transfer)
	var e, s, f int
	err := db.QueryRow(q, id).Scan(&t.Id, &t.Status, &t.User, &t.Location, &t.Lastmod, &e, &f, &s)
	switch err {
	case nil:
		t.Event, _ = ViewEvent(db, e)
		t.Slot, _ = viewSlot(db, s)
		t.File, _ = ViewFile(db, f)
		return t, nil
	case sql.ErrNoRows:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}

func NewTransfer(db *sql.DB, i, e int, u, d string) (*Transfer, error) {
	const q = `
		with
			u(pk) as (select pk from vusers where initial=$3),
			e(pk) as (select pk from schedule.events where source is null and pk=$2 and not canceled)
		insert into schedule.transfers(uplink, event, person, location) values($1, (select pk from e), (select pk from u), $4) returning pk`
	var id int
	if err := db.QueryRow(q, i, e, u, d).Scan(&id); err != nil {
		return nil, err
	}
	return ViewTransfer(db, id)
}

func UpdateTransfer(db *sql.DB, id int, s, u string) (*Transfer, error) {
	const q = `
		with
			u(pk) as (select pk from vusers where initial=$3 limit 1)
		update schedule.transfers set state=$2, person=(select pk from u), lastmod=current_timestamp where pk=$1`
	if _, err := db.Exec(q, id, s, u); err != nil {
		return nil, err
	}
	return ViewTransfer(db, id)
}

func ListUplinks(db *sql.DB, fd, td time.Time, cs, ts []string) ([]*Uplink, error) {
	const q = `
		select
			pk, dropbox, state, person, lastmod, slot, event, file
		from vuplinks
		where
			dtstamp between $1 and $2
			and case when cardinality($3::usoc.status[]) > 0 then state=any($3::usoc.status[]) else true end
			and case when cardinality($4::text[]) > 0 then category=any($4::text[]) else true end`
	rs, err := db.Query(q, fd, td, pq.StringArray(ts), pq.StringArray(cs))
	if err != nil {
		return nil, err
	}
	data := make([]*Uplink, 0, 100)
	for rs.Next() {
		u, err := scanUplink(rs, db)
		if err != nil {
			return nil, err
		}
		data = append(data, u)
	}
	return data, nil
}

func ViewUplink(db *sql.DB, id int) (*Uplink, error) {
	const q = `select pk, dropbox, state, person, lastmod, slot, event, file from vuplinks where pk=$1`
	return scanUplink(db.QueryRow(q, id), db)
}

func NewUplink(db *sql.DB, s, e, f int, u string) (*Uplink, error) {
	return createUplink(db, s, e, f, u, false)
}

func UpdateUplink(db *sql.DB, id int, s, u string) (*Uplink, error) {
	if err := updateUplink(db, id, s, u); err != nil {
		return nil, err
	}
	return ViewUplink(db, id)
}

func updateUplink(db *sql.DB, id int, s, u string) error {
	const q = `
		with
			u(pk) as (select pk from vusers where initial=$3 limit 1)
		update schedule.uplinks set state=$2, person=(select pk from u), lastmod=current_timestamp where pk=$1`
	if _, err := db.Exec(q, id, s, u); err != nil {
		return err
	}
	return nil
}

func createUplink(db *sql.DB, slot, event, file int, user string, dummy bool) (*Uplink, error) {
	const q = `
		with
			f(pk) as(select pk from schedule.files where pk=$3 and case when $4::boolean then (content is null or length(content)=0) else (content is not null and length(content)>0) end),
			u(pk) as (select pk from vusers where initial=$5 limit 1),
			e(pk) as (select pk from schedule.events where source is null and pk=$2 and not canceled)
		insert into schedule.uplinks(slot, event, file, person) values($1, (select pk from e), (select pk from f), (select pk from u)) returning pk`
	// const q = `
	// 	with
	// 		f(pk) as(select pk from schedule.files where pk=$3),
	// 		u(pk) as (select pk from vusers where initial=$4 limit 1),
	// 		e(pk) as (select pk from schedule.events where source is null and pk=$2 and not canceled)
	// 	insert into schedule.uplinks(slot, event, file, person) values($1, (select pk from e), (select pk from f), (select pk from u)) returning pk`

	var id int
	if err := db.QueryRow(q, slot, event, file, dummy, user).Scan(&id); err != nil {
		return nil, err
	}
	if dummy {
		return ViewDownlink(db, id)
	} else {
		return ViewUplink(db, id)
	}
}

func scanUplink(sc Scanner, db *sql.DB) (*Uplink, error) {
	var s, f, e int
	u := new(Uplink)
	err := sc.Scan(&u.Id, &u.Name, &u.Status, &u.User, &u.Lastmod, &s, &e, &f)
	switch err {
	default:
		return nil, err
	case sql.ErrNoRows:
		return nil, ErrNotFound
	case nil:
		u.Slot, _ = viewSlot(db, s)
		u.Event, _ = ViewEvent(db, e)
		u.File, _ = ViewFile(db, f)
		return u, nil
	}
}
