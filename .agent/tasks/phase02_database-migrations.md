# Task: Database Migrations

## Phase
phase02

## Status
done

## Completed At
2026-03-06

## Files Created / Modified
- backend/migrations/001_initial_schema.sql
- backend/migrations/002_seed_workers.sql

## Notes
- Tables: users, sessions, worker_groups, workers, jobs, messages
- job.status enum: draft/queued/assigned/busy/pending_user/done/failed/cancelled
- worker.status enum: busy/pending_user/idle/offline
- message.role: user/oagent/worker; message.kind: instruction/question/answer/summary/system
- set_updated_at() trigger on jobs BEFORE UPDATE
- 002 seeds fi-backend, fi-frontend, ib-kha groups and 5 workers with placeholder api_key_hash
