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

drop view if exists revisions.vfiles cascade;
drop view if exists revisions.vtodos cascade;
drop view if exists revisions.vevents cascade;

create or replace view vjournals(pk, day, summary, meta, state, lastmod, person, categories) as
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

create or replace view vusers(pk, firstname, lastname, initial, email, internal, settings, passwd, positions) as
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

create or replace view vcategories(pk, name, person, lastmod) as
	select
		c.pk,
		c.name,
		coalesce(p.initial, 'gpt'),
		c.lastmod
	from schedule.categories c
		left outer join usoc.persons p on c.person=p.pk
	where
		not c.canceled;

create or replace view vtodos(pk, summary, description, state, priority, person, version, meta, categories, assignees, due, dtstart, dtend, lastmod, parent) as
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

create or replace view vslots(sid, name, person, category, lastmod, state, file) as
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

create or replace view vfiles(pk, version, name, crc, summary, meta, person, lastmod, superseeded, original, length, sum, categories, slot, location) as
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
		f.crc,
		coalesce(f.summary, ''),
		f.meta,
		p.initial,
		f.lastmod,
		f.parent is null or exists(select v.pk from schedule.files v where v.parent=f.pk),
		-- case
    --   when f.parent is null then false
    --   else exists(select v.pk from schedule.files v where v.pk=f.parent)
    -- end,
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

create or replace view vevents(pk, source, summary, description, meta, state, version, dtstart, dtend, rtstart, rtend, categories, person, attendees, lastmod, parent) as
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
		coalesce(e.source, ''),
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
		join (select * from schedule.events e where not e.canceled) e on u.event=e.pk
		join usoc.persons p on u.person=p.pk
		join (select pk, name from schedule.files f where not f.canceled and (f.content is not null and length(f.content)>0)) f on u.file=f.pk
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
		join (select * from schedule.events e where not e.canceled) e on u.event=e.pk
		join usoc.persons p on u.person=p.pk
		join (select pk from schedule.files f where not f.canceled and (f.content is not null or length(f.content)>0)) f on u.file=f.pk
		join vslots s on u.slot=s.sid;

create or replace view vtransfers(pk, state, person, lastmod, location, event, uplink, slot, file, dtstamp, category) as
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
		join (select * from schedule.events e where not e.canceled) e on t.event=e.pk
		join (select u.* from schedule.uplinks u join schedule.files f on u.file=f.pk where not f.canceled) u on t.uplink=u.pk
		join vslots s on u.slot=s.sid
		join usoc.persons p on t.person=p.pk;

create or replace view revisions.vevents(pk, source, summary, description, meta, state, version, dtstart, dtend, rtstart, rtend, person, attendees, categories, lastmod) as
	select
		e.pk,
		coalesce(e.source, ''),
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

create or replace view revisions.vtodos(pk, summary, description, state, priority, person, version, meta, categories, assignees, dtstart, dtend, due, lastmod) as
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

create or replace view revisions.vfiles(pk, name, slot, location, summary, categories, meta, version, length, sum, superseeded, person, lastmod) as
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
