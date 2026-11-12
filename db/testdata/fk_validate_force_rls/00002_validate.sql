-- TICK-715 guard fixture: the bare VALIDATE the guard must catch. Run by the
-- owner under FORCE RLS it checks nothing.

-- +goose Up

SET LOCAL lock_timeout = '5s';

ALTER TABLE child VALIDATE CONSTRAINT child_parent_fkey;
