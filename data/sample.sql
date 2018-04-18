BEGIN;

insert into usoc.positions(name, abbr) values
  ('developper', 'dev'),
  ('ground controller', 'gc'),
  ('operator', 'op');

insert into usoc.persons(firstname, lastname, initial, email, internal, passwd) values
  ('roger', 'lamotte', 'rla', 'roger.lamotte@busoc.be', true, encode(digest('helloworld', 'sha256'), 'hex')),
  ('pierre', 'dupond', 'pdu', 'pierre.dupond@busoc.be', true, encode(digest('helloworld', 'sha256'), 'hex'));

insert into usoc.persons_positions(person, position) values
  (1, 1),
  (1, 2),
  (2, 3);

insert into schedule.categories(name, person) values
  ('solar', 1),
  ('fsl', 1),
  ('asim', 1),
  ('command', 2),
  ('configuration', 2),
  ('uplink', 1),
  ('transfer', 2);

insert into schedule.events(summary, dtstart, dtend, person) values
  ('evt-uplink-0', '2017-06-23T13:00:00Z', '2017-06-23T14:00:00Z', 1),
  ('evt-uplink-1', '2017-06-24T13:00:00Z', '2017-06-24T14:00:00Z', 1),
  ('evt-uplink-2', '2017-06-25T13:00:00Z', '2017-06-25T14:00:00Z', 1),
  ('evt-uplink-3', '2017-06-26T13:00:00Z', '2017-06-26T14:00:00Z', 1),
  ('evt-uplink-4', '2017-06-27T13:00:00Z', '2017-06-27T14:00:00Z', 1),
  ('evt-uplink-5', '2017-06-28T13:00:00Z', '2017-06-28T14:00:00Z', 1),
  ('evt-uplink-6', '2017-06-29T13:00:00Z', '2017-06-29T14:00:00Z', 1),
  ('evt-uplink-7', '2017-06-30T13:00:00Z', '2017-06-30T14:00:00Z', 1),
  ('evt-transfer-0', '2017-06-23T16:00:00Z', '2017-06-23T18:00:00Z', 1),
  ('evt-downlink-0', '2017-08-10T00:00:00Z', '2017-08-10T01:00:00Z', 1);

insert into schedule.todos(summary, due, person) values
  ('td-todo-0', '2017-07-09T17:00:00Z', 1),
  ('td-todo-1', '2017-07-09T17:00:00Z', 1),
  ('td-todo-2', '2017-07-09T17:00:00Z', 1),
  ('td-todo-3', '2017-07-09T17:00:00Z', 1),
  ('td-todo-4', '2017-07-09T17:00:00Z', 1),
  ('td-todo-5', '2017-07-09T17:00:00Z', 1);

insert into schedule.todos_categories(todo, category) values
  (1, 1),
  (1, 5),
  (2, 2),
  (3, 3),
  (4, 3),
  (5, 6),
  (5, 7);

insert into schedule.assignees(todo, person) values
  (1, 1),
  (1, 2),
  (2, 2),
  (3, 2),
  (4, 1),
  (4, 2),
  (5, 1),
  (6, 1),
  (6, 2);

insert into schedule.events_categories(event, category) values
  (1, 6),
  (2, 6),
  (3, 6),
  (4, 6),
  (5, 6),
  (6, 6),
  (7, 6),
  (8, 6),
  (9, 7);

insert into schedule.slots(pk, name, category, person) values
  (328825001, 's-solar-0', 1, 1),
  (328825002, 's-solar-1', 1, 1),
  (328825003, 's-solar-2', 1, 1),
  (318855001, 's-fsl-0', 2, 1),
  (318855002, 's-fsl-1', 2, 1),
  (318855003, 's-fsl-2', 2, 1),
  (318855004, 's-fsl-3', 2, 1),
  (318855005, 's-fsl-4', 2, 1),
  (318855006, 's-fsl-5', 2, 1),
  (318615001, 's-asim-0', 3, 1),
  (318615002, 's-asim-1', 3, 1);

insert into schedule.files(name, content, person) values
  ('f-solar-0.bin', convert_to('solar-0', 'UTF8'), 1),
  ('f-solar-1.bin', convert_to('solar-1', 'UTF8'), 1),
  ('f-solar-2.dat', convert_to('solar-2', 'UTF8'), 1),
  ('f-solar-3.dat', convert_to('solar-3', 'UTF8'), 1),
  ('f-solar-4.dat', convert_to('solar-4', 'UTF8'), 1),
  ('f-fsl-0', convert_to('fsl-0', 'UTF8'), 1),
  ('f-fsl-1', convert_to('fsl-1', 'UTF8'), 1),
  ('f-fsl-2', convert_to('fsl-2', 'UTF8'), 1),
  ('f-fsl-3.fsl', convert_to('fsl-3', 'UTF8'), 1),
  ('f-fsl-4.fsl', convert_to('fsl-4', 'UTF8'), 1),
  ('f-asim-0.rc', convert_to('asim-0', 'UTF8'), 1),
  ('f-asim-1.rc', convert_to('asim-1', 'UTF8'), 1),
  ('d-dummy-0', null, 1),
  ('d-dummy-1', null, 1),
  ('d-dummy-2', null, 1),
  ('d-dummy-3', null, 1);

insert into schedule.files_categories (file, category) values
  (1, 1), -- solar-0
  (2, 1), -- solar-1
  (3, 1), -- solar-2
  (4, 1), -- solar-3
  (5, 1), -- solar-4
  (6, 2), -- fsl-0
  (7, 2), -- fsl-1
  (8, 2), -- fsl-2
  (9, 2), -- fsl-3
  (10, 2), -- fsl-4
  (11, 3), -- asim-0
  (12, 3), -- asim-1
  (1, 4), --solar-0, command
  (2, 4); --solar-1, command

insert into schedule.uplinks(event, slot, file, lastmod, person, state) values
  (1, 328825001, 1, '2017-06-18T09:00:00Z', 1, 'completed'), -- event-0, s-solar-0, f-solar-0
  (1, 328825002, 2, '2017-06-18T09:00:00Z', 1, 'completed'), -- event-0, s-solar-1, f-solar-1
  (2, 328825001, 3, '2017-06-21T09:00:00Z', 1, 'completed'), -- event-1, s-solar-1, f-solar-3
  (10, 318855005, 14, '2017-08-14T11:15:23Z', 1, 'completed');

insert into schedule.transfers(event, uplink, lastmod, location, person, state) values
  (9, 3, '2017-06-20T10:00:00Z', '/commands/solar/', 2, 'completed');

COMMIT;
