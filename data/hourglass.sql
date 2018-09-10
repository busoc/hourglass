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
	crc int default 0,
	primary key(pk),
	foreign key(person) references usoc.persons(pk),
	foreign key(parent) references schedule.files(pk),
	-- constraint files_name_unique unique(name),
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
	source varchar(64),
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
	foreign key(person) references usoc.persons(pk)
	-- constraint uplinks_file_event_unique unique(file, event),
	-- constraint uplinks_slot_event_unique unique(slot, event)
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
