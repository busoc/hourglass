drop schema if exists usoc cascade;
drop schema if exists schedule cascade;
drop schema if exists revisions cascade;

create extension if not exists pgcrypto;

create schema usoc;
create schema schedule;
create schema revisions;

create type usoc.priority as ENUM('low', 'normal', 'high', 'urgent');
create type usoc.status as ENUM ('n/a', 'tentative', 'scheduled', 'on going', 'completed', 'canceled', 'aborted');

create table usoc.positions (
	pk serial,
	name varchar(128) not null unique,
	abbr varchar(4) not null unique,
	primary key(pk)
);

create table usoc.persons (
	pk serial,
	firstname varchar(64) not null,
	lastname varchar(64) not null,
	initial varchar(4) not null,
	email varchar(256) not null,
	internal boolean default false,
	passwd text,
	settings json,
	primary key(pk),
	unique(initial),
	unique(email)
);

create table usoc.persons_positions (
	person int not null,
	position int not null,
	primary key(person, position),
	foreign key (person) references usoc.persons(pk),
	foreign key (position) references usoc.positions(pk)
);

create table schedule.categories (
	pk serial not null,
	name varchar not null,
	lastmod timestamp not null default current_timestamp,
	person int,
	canceled boolean default false,
	parent int,
	primary key(pk),
	foreign key(parent) references schedule.categories(pk),
	foreign key(person) references usoc.persons(pk),
	constraint categories_name_unique unique (name),
	constraint categories_name_length check (length(name) > 0)
);

create table schedule.files (
	pk serial not null,
	name varchar(1024) not null,
	summary text,
	meta json,
	person int not null,
	content bytea null,
	lastmod timestamp not null default current_timestamp,
	canceled bool default false,
	parent int,
	primary key(pk),
	foreign key(person) references usoc.persons(pk),
	foreign key(parent) references schedule.files(pk),
	constraint files_name_unique unique(name),
	constraint files_name_length check(length(name)>0)
);

create table revisions.files (
	like schedule.files INCLUDING DEFAULTS,
	categories text[]
);

alter table revisions.files drop column content;

create table schedule.files_categories (
	file int not null,
	category int not null,
	primary key(file, category),
	foreign key(file) references schedule.files(pk),
	foreign key(category) references schedule.categories(pk)
);

create table schedule.slots (
	pk int not null,
	name varchar not null,
	category int not null,
	canceled boolean default false,
	lastmod timestamp default current_timestamp,
	person int not null,
	primary key(pk),
	foreign key(category) references schedule.categories(pk),
	foreign key(person) references usoc.persons(pk),
	constraint slots_name_unique unique(name),
	constraint slots_name_length check (length(name) > 0)
);

create table schedule.events (
	pk serial not null,
	summary varchar(256) not null,
	description text,
	source varchar(64) default "busoc" not null,
	meta json,
	state usoc.status default 'scheduled',
	dtstart timestamp not null,
	dtend timestamp not null,
	rtstart timestamp,
	rtend timestamp,
	file int,
	person int,
	parent int,
	canceled boolean default false,
	lastmod timestamp default current_timestamp,
	primary key(pk),
	foreign key(person) references usoc.persons(pk),
	foreign key(file) references schedule.files(pk),
	foreign key(parent) references schedule.events(pk)
);

create table schedule.attendees (
	event int not null,
	person int not null,
	primary key(event, person),
	foreign key(event) references schedule.events(pk),
	foreign key(person) references usoc.persons(pk)
);

create table schedule.events_categories (
	event int not null,
	category int not null,
	primary key(event, category),
	foreign key(event) references schedule.events(pk),
	foreign key(category) references schedule.categories(pk)
);

create table revisions.events (
	like schedule.events INCLUDING DEFAULTS,
	attendees text[],
	categories text[]
);

create table schedule.uplinks (
	pk serial not null,
	state usoc.status default 'scheduled',
	event int not null,
	slot int not null,
	file int not null,
	person int not null,
	lastmod timestamp not null default current_timestamp,
	primary key(pk),
	foreign key(event) references schedule.events(pk),
	foreign key(slot) references schedule.slots(pk),
	foreign key(file) references schedule.files(pk),
	foreign key(person) references usoc.persons(pk),
	constraint uplinks_file_event_unique unique(file, event),
	constraint uplinks_slot_event_unique unique(slot, event)
);

create table schedule.transfers (
	pk serial not null,
	state usoc.status not null default 'scheduled',
	event int not null,
	uplink int not null,
	location varchar default '/',
	lastmod timestamp not null default current_timestamp,
	person int not null,
	primary key(pk),
	foreign key(event) references schedule.events(pk),
	foreign key(uplink) references schedule.uplinks(pk),
	foreign key(person) references usoc.persons(pk),
	constraint transfers_event_uplink_unique unique(event, uplink)
);

create table schedule.todos (
	pk serial not null,
	summary varchar(1024) not null,
	description text,
	meta json,
	dtstart timestamp,
	dtend timestamp,
	due timestamp not null,
	state usoc.status not null default 'scheduled',
	priority usoc.priority not null default 'normal',
	person int not null,
	parent int,
	lastmod timestamp not null default current_timestamp,
	canceled bool not null default false,
	primary key(pk),
	foreign key(person) references usoc.persons(pk),
	foreign key(parent) references schedule.todos(pk)
);

create table schedule.assignees (
	todo int not null,
	person int not null,
	primary key(todo, person),
	foreign key(todo) references schedule.todos(pk),
	foreign key (person) references usoc.persons(pk)
);

create table schedule.todos_categories (
	todo int not null,
	category int not null,
	primary key(todo, category),
	foreign key(todo) references schedule.todos(pk),
	foreign key (category) references schedule.categories(pk)
);

create table revisions.todos (
	like schedule.todos INCLUDING DEFAULTS,
	assignees text[],
	categories text[]
);

create table schedule.journals (
	pk serial not null,
	day timestamp not null default current_timestamp,
	summary varchar(4096),
	meta json,
	state usoc.status not null default 'scheduled',
	lastmod timestamp not null default current_timestamp,
	person int not null,
	primary key(pk),
	foreign key(person) references usoc.persons(pk)
);

create table schedule.journals_categories (
	journal int not null,
	category int not null,
	primary key(journal, category),
	foreign key(journal) references schedule.journals(pk),
	foreign key(category) references schedule.categories(pk)
);

create table revisions.journals (
	like schedule.journals including DEFAULTS
);

drop view if exists vslots cascade;
drop view if exists vfiles cascade;
drop view if exists vevents cascade;
drop view if exists vtodos cascade;
drop view if exists vuplinks cascade;
drop view if exists vdownlinks cascade;
drop view if exists vtransfers cascade;
drop view if exists vcategories cascade;
drop view if exists vusers cascade;
drop view if exists vjournals cascade;

create view vjournals(pk, day, summary, meta, state, lastmod, person, categories) as
	with cs(pk, vs) as (
		select
			j.journal,
			array_agg(c.name)
		from
			schedule.journals_categories j
			join schedule.categories c on j.category=c.pk
		group by
			j.journal
	)
	select
		j.pk,
		j.day,
		j.summary,
		j.meta,
		j.state,
		j.lastmod,
		p.initial,
		coalesce(c.vs, '{}'::text[])
	from
		schedule.journals j
		join usoc.persons p on j.person=p.pk
		left outer join cs c on j.pk=c.pk;

create view vusers(pk, firstname, lastname, initial, email, internal, settings, passwd, positions) as
	with jobs(person, positions) as (
		select
			p.person,
			array_agg(o.abbr)
		from usoc.persons_positions p
			join usoc.positions o on p.position=o.pk
		group by
			p.person
	)
	select
		p.pk,
		p.firstname,
		p.lastname,
		p.initial,
		p.email,
		p.internal,
		p.settings,
		p.passwd,
		coalesce(j.positions, '{}'::text[])
	from usoc.persons p
		left outer join jobs j on p.pk=j.person
	where
		passwd is not null;

create view vcategories(pk, name, person, lastmod) as
	select
		c.pk,
		c.name,
		coalesce(p.initial, 'gpt'),
		c.lastmod
	from schedule.categories c
		left outer join usoc.persons p on c.person=p.pk
	where
		not c.canceled;

create view vtodos(pk, summary, description, state, priority, person, version, meta, categories, assignees, due, dtstart, dtend, lastmod, parent) as
	with
		ps(pk, vs) as (
			select
				a.todo,
				array_agg(p.initial)
			from
				schedule.assignees a
				join vusers p on a.person=p.pk
			group by
				a.todo
		),
		cs(pk, vs) as (
			select
				t.todo,
				array_agg(c.name)
			from
				schedule.todos_categories t
				join schedule.categories c on t.category=c.pk
			group by
				t.todo
		),
		rs(pk, count) as (
			select
				pk,
				count(pk)
			from
				revisions.todos
			group by
				pk
		)
	select
		t.pk,
		t.summary,
		coalesce(t.description, ''),
		t.state,
		t.priority,
		p.initial,
		coalesce(rs.count+1, 1),
		t.meta,
		coalesce(cs.vs, '{}'::text[]),
		coalesce(ps.vs, '{}'::text[]),
		t.due,
		t.dtstart,
		t.dtend,
		t.lastmod,
		t.parent
	from
		schedule.todos t
		join usoc.persons p on t.person=p.pk
		left outer join rs on rs.pk=t.pk
		left outer join cs on cs.pk=t.pk
		left outer join ps on ps.pk=t.pk
	where
		not t.canceled;

create view vslots(sid, name, person, category, lastmod, state, file) as
	with
		us(pk) as (
			select
				max(pk)
			from
				schedule.uplinks
			group by slot
	)
	select
		s.pk,
		s.name,
		p.initial,
		c.name,
		coalesce(u.lastmod, s.lastmod),
		coalesce(u.state, 'n/a'),
		coalesce(f.name, '')
	from
		schedule.slots s
		join schedule.categories c on s.category=c.pk
		join usoc.persons p on s.person=p.pk
		left outer join (select * from schedule.uplinks where pk in (select pk from us)) u on s.pk=u.slot
		left outer join schedule.files f on u.file=f.pk
	where
		not s.canceled;

create view vfiles(pk, version, name, summary, meta, person, lastmod, superseeded, original, length, sum, categories, slot, location) as
	with
		cs(pk, vs) as (
				select
					f.file,
					array_agg(c.name)
				from
					schedule.files_categories f
					join schedule.categories c on f.category=c.pk
				group by f.file
		), rs(pk, version) as (
				select
					f.pk,
					count(f.pk)
				from
					revisions.files f
				group by f.pk
		), us(pk) as (
				select
					max(pk)
				from
					schedule.uplinks
				group by
			slot
	)
	select
		f.pk,
		coalesce(rs.version+1, 1),
		f.name,
		coalesce(f.summary, ''),
		f.meta,
		p.initial,
		f.lastmod,
		not(exists(select v.pk from schedule.files v where v.pk=f.parent)),
		f.parent is null,
		coalesce(length(f.content), 0),
		coalesce(md5(f.content), ''),
		coalesce(cs.vs, '{}'::varchar[]),
		case
			when u.slot is not null and u.state='completed' then s.name
			else ''
		end,
		case
			when t.pk is not null and t.state='completed' then t.location
			when t.pk is null and f.content is null then '/'
			else ''
		end
	from
		schedule.files f
		join usoc.persons p on f.person=p.pk
		left outer join rs on rs.pk=f.pk
		left outer join cs on cs.pk=f.pk
		left outer join (select * from schedule.uplinks where pk in (select pk from us))u on f.pk=u.file
		left outer join schedule.slots s on u.slot=s.pk
		left outer join schedule.transfers t on u.pk=t.uplink
	where
		not f.canceled;

create view vevents(pk, source, summary, description, meta, state, version, dtstart, dtend, rtstart, rtend, categories, person, attendees, lastmod, parent) as
	with items(event, categories) as (
		select
			e.event,
			array_agg(c.name)
		from
			schedule.categories c
			join schedule.events_categories e on c.pk=e.category
		group by e.event
	), attendees (event, persons) as (
		select
			a.event,
			array_agg(p.initial)
		from
			vusers p
			join schedule.attendees a on p.pk=a.person
		group by a.event
	),
	rs(event, count) as (
		select
			pk,
			count(pk)
		from
			revisions.events
		group by
			pk
	)
	select
		e.pk,
		e.source,
		e.summary,
		coalesce(e.description, ''),
		e.meta,
		e.state,
		coalesce(rs.count+1, 1),
		e.dtstart,
		e.dtend,
		coalesce(e.rtstart, e.dtstart),
		coalesce(e.rtend, e.dtend),
		coalesce(i.categories, '{}'::text[]),
		coalesce(p.initial, 'gpt'),
		coalesce(a.persons, '{}'::text[]),
		e.lastmod,
		e.parent
	from
		schedule.events e
		left outer join rs on e.pk=rs.event
		left outer join items i on e.pk=i.event
		left outer join attendees a on e.pk=a.event
		left outer join usoc.persons p on e.person=p.pk
	where
		not e.canceled;

create or replace view vuplinks(pk, dropbox, state, person, lastmod, event, file, slot, dtstamp, category) as
	select
		u.pk,
		concat_ws('_', 'S', u.slot, upper(regexp_replace(s.name, '\.', '_')), upper(split_part(f.name, '.', 1)), to_char(e.dtstart, 'YY_DDD_HH24_MI')),
		u.state,
		p.initial,
		u.lastmod,
		u.event,
		u.file,
		u.slot,
		coalesce(e.rtstart, e.dtstart),
		s.category
	from
		schedule.uplinks u
		join schedule.events e on u.event=e.pk
		join usoc.persons p on u.person=p.pk
		join (select pk, name from schedule.files f where f.content is not null and length(f.content)>0) f on u.file=f.pk
		join vslots s on u.slot=s.sid
	where
		u.pk in (select max(pk) from schedule.uplinks group by(slot));

create or replace view vdownlinks(pk, state, person, lastmod, event, file, slot, dtstamp, category) as
	select
		u.pk,
		u.state,
		p.initial,
		u.lastmod,
		u.event,
		u.file,
		u.slot,
		coalesce(e.rtstart, e.dtstart),
		s.category
	from
		schedule.uplinks u
		join schedule.events e on u.event=e.pk
		join usoc.persons p on u.person=p.pk
		join (select pk from schedule.files f where f.content is null or length(f.content)=0) f on u.file=f.pk
		join vslots s on u.slot=s.sid;

create view vtransfers(pk, state, person, lastmod, location, event, uplink, slot, file, dtstamp, category) as
	select
		t.pk,
		t.state,
		p.initial,
		t.lastmod,
		t.location,
		t.event,
		t.uplink,
		u.slot,
		u.file,
		e.dtstart,
		s.category
	from schedule.transfers t
		join schedule.events e on t.event=e.pk
		join schedule.uplinks u on t.uplink=u.pk
		join vslots s on u.slot=s.sid
		join usoc.persons p on t.person=p.pk;

drop function if exists updateFiles() cascade;
drop function if exists updateTodos() cascade;
drop function if exists updateEvents() cascade;

create function updateTodos() returns trigger as $auditTodos$
	begin
		insert into revisions.todos
			select
				OLD.pk,
				OLD.summary,
				OLD.description,
				OLD.meta,
				OLD.dtstart,
				OLD.dtend,
				OLD.due,
				OLD.state,
				OLD.priority,
				OLD.person,
				OLD.parent,
				OLD.lastmod,
				OLD.canceled,
				v.assignees,
				v.categories
			from vtodos v
			where v.pk=OLD.pk;
		delete from schedule.assignees where todo=OLD.pk;
		delete from schedule.todos_categories where todo=OLD.pk;
		return NEW;
	end;
$auditTodos$ language plpgsql;

create function updateEvents() returns trigger as $auditEvents$
	begin
		insert into revisions.events
			select
				OLD.pk,
				OLD.summary,
				OLD.description,
				OLD.source,
				OLD.meta,
				OLD.state,
				OLD.dtstart,
				OLD.dtend,
				OLD.rtstart,
				OLD.rtend,
				OLD.file,
				OLD.person,
				OLD.parent,
				OLD.canceled,
				OLD.lastmod,
				v.attendees,
				v.categories
			from vevents v
			where
				v.pk=OLD.pk;
		delete from schedule.attendees where event=OLD.pk;
		delete from schedule.events_categories where event=OLD.pk;
		return NEW;
	end;
$auditEvents$ language plpgsql;

create function updateFiles() returns trigger as $auditFiles$
	begin
		if OLD.content != NEW.content then
			raise exception 'content is frozen';
		end if;
		insert into revisions.files
			select
				OLD.pk,
				OLD.name,
				OLD.summary,
				OLD.meta,
				OLD.person,
				OLD.lastmod,
				OLD.canceled,
				OLD.parent,
				v.categories
			from
				vfiles v
			where v.pk=OLD.pk;
		delete from schedule.files_categories where file=OLD.pk;
		return NEW;
	end;
$auditFiles$ language plpgsql;

drop trigger if exists trackFiles on schedule.files;
drop trigger if exists trackEvents on schedule.events;
drop trigger if exists trackTodos on schedule.todos;

create trigger trackTodos
	before update on schedule.todos
	for each row
	when (not OLD.canceled)
	execute procedure updateTodos();

create trigger trackEvents
	before update on schedule.events
	for each row
	when (not OLD.canceled and OLD.source is null)
	execute procedure updateEvents();

create trigger trackFiles
	before update on schedule.files
	for each row
	when (not OLD.canceled or OLD.parent is null)
	execute procedure updateFiles();

create view revisions.vevents(pk, source, summary, description, meta, state, version, dtstart, dtend, rtstart, rtend, person, attendees, categories, lastmod) as
	select
		e.pk,
		e.source,
		e.summary,
		coalesce(e.description, ''),
		e.meta,
		e.state,
		row_number() over (partition by e.pk order by e.lastmod),
		e.dtstart,
		e.dtend,
		coalesce(e.rtstart, e.dtstart),
		coalesce(e.rtend, e.dtend),
		p.initial,
		e.attendees,
		e.categories,
		e.lastmod
	from
		revisions.events e
		join usoc.persons p on e.person=p.pk;

create view revisions.vtodos(pk, summary, description, state, priority, person, version, meta, categories, assignees, dtstart, dtend, due, lastmod) as
	select
		t.pk,
		t.summary,
		coalesce(t.description, ''),
		t.state,
		t.priority,
		p.initial,
		row_number() over (partition by t.pk order by t.lastmod),
		t.meta,
		coalesce(t.categories, '{}'::text[]),
		coalesce(t.assignees, '{}'::text[]),
		t.dtstart,
		t.dtend,
		t.due,
		t.lastmod
	from
		revisions.todos t
		join usoc.persons p on t.person=p.pk;

create view revisions.vfiles(pk, name, slot, location, summary, categories, meta, version, length, sum, superseeded, person, lastmod) as
	select
		f.pk,
		f.name,
		''::varchar,
		''::varchar,
		coalesce(f.summary, ''),
		f.categories,
		f.meta,
		row_number() over(partition by f.pk order by f.lastmod),
		coalesce(length(s.content), 0),
		coalesce(md5(s.content), ''),
		s.parent is null,
		p.initial,
		f.lastmod
	from
		revisions.files f
		join schedule.files s on f.pk=s.pk
		join usoc.persons p on f.person=p.pk;
