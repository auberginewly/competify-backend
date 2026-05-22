// Command worker — Agent worker entrypoint (for distributed deployments).
// Phase 5+: subscribe to NATS task queue, execute long-running Agent jobs.
package main

import "log"

func main() {
	// TODO Phase 5:
	//   1. Connect to NATS
	//   2. Subscribe to competify.task.> subject
	//   3. For each task message, run the DAG runnable
	//   4. Publish completion events back to NATS
	log.Println("CompetifyAI worker scaffold ready. Implement in Phase 5.")
}
