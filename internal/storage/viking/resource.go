package viking

import "fmt"

// CompetifyVikingPaths centralizes all viking:// URI construction.
// Agents must use these methods — never hardcode viking:// strings.
type CompetifyVikingPaths struct {
	BasePath string
}

// NewCompetifyPaths creates a namespace manager rooted at viking://competify/.
func NewCompetifyPaths() *CompetifyVikingPaths {
	return &CompetifyVikingPaths{BasePath: "viking://competify/"}
}

// TaskPath: viking://competify/tasks/{task_id}/
func (p *CompetifyVikingPaths) TaskPath(taskID string) string {
	return fmt.Sprintf("%stasks/%s/", p.BasePath, taskID)
}

// TaskCollectorPath: viking://competify/tasks/{task_id}/collectors/{source_type}/
func (p *CompetifyVikingPaths) TaskCollectorPath(taskID, sourceType string) string {
	return fmt.Sprintf("%scollectors/%s/", p.TaskPath(taskID), sourceType)
}

// TaskAnalyzerPath: viking://competify/tasks/{task_id}/analyzers/{dimension}/
func (p *CompetifyVikingPaths) TaskAnalyzerPath(taskID, dimension string) string {
	return fmt.Sprintf("%sanalyzers/%s/", p.TaskPath(taskID), dimension)
}

// TaskReviewerPath: viking://competify/tasks/{task_id}/reviewer/
func (p *CompetifyVikingPaths) TaskReviewerPath(taskID string) string {
	return fmt.Sprintf("%sreviewer/", p.TaskPath(taskID))
}

// OntologyPath: viking://competify/ontology/
func (p *CompetifyVikingPaths) OntologyPath() string {
	return p.BasePath + "ontology/"
}

// CompetitorOntologyPath: viking://competify/ontology/competitors/{name}/
func (p *CompetifyVikingPaths) CompetitorOntologyPath(name string) string {
	return fmt.Sprintf("%scompetitors/%s/", p.OntologyPath(), name)
}

// ProvenancePath: viking://competify/provenance/{conclusion_id}/
func (p *CompetifyVikingPaths) ProvenancePath(conclusionID string) string {
	return fmt.Sprintf("%sprovenance/%s/", p.BasePath, conclusionID)
}
