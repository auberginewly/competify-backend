// Package handler implements HTTP and WebSocket routes.
// Routes call dag.Runnable or agent layers — never access storage directly.
// See docs/api.md for the full API contract.
package handler

// TODO Phase 2:
//   - POST /api/v1/tasks           → CreateTask (body: UserQuery, response: {task_id})
//   - GET  /api/v1/tasks/:id       → GetTask
//   - GET  /api/v1/tasks/:id/dag   → GetDAGState (WebSocket)
