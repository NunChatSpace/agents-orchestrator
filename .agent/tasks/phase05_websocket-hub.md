# Task: WebSocket Hub

## Phase
phase05

## Status
done

## Completed At
2026-03-06

## Files Created / Modified
- backend/internal/ws/hub.go
- backend/internal/ws/handler.go

## Notes
- Hub maps userID → []*client; broadcastAll used since app is single-user personal tool
- Events: job_updated, message_added, worker_updated
- Write pump: gorilla/websocket, ping every 30s, 10s write deadline
- Read pump: drains pong/close frames only (maxMsgSize=512)
- ServeWS registered in routes.go under /ws, session-protected
