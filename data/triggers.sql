drop function if exists updateFiles() cascade;
drop function if exists updateTodos() cascade;
drop function if exists updateEvents() cascade;

create function updateJournals() returns trigger as $auditJournals$
	begin
		insert into revisions.journals
			select
				OLD.pk,
				OLD.day,
				OLD.summary,
				OLD.meta,
				OLD.state,
				OLD.lastmod,
				OLD.person,
				v.categories
			from vjournals v
			when v.pk=OLD.pk;
		delete from schedule.journals_categories where journal=OLD.pk;
	end
$audiJournals$ language plpgsql;

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
drop trigger if exists trackJournals on schedule.journals;

create trigger trackJournals
	before update on schedule.journals
	for each row
	when (not OLD.canceled)
	execute procedure updateJournals();

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
