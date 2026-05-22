package dag

// Workflow 用于节点内部的字段级编排（Cleaner 多源汇聚、Cross-Reviewer 多维聚合）。
//
// Phase 2 实现计划：
//
//	wf := compose.NewWorkflow()
//	wf.AddLambdaNode("extract", ...).AddInput(compose.START)
//	wf.AddLambdaNode("dedup", ...).AddInput("extract")
//	wf.AddLambdaNode("normalize", ...).AddInput("dedup")
//	wf.End().AddInput("normalize")
//	return wf.Compile(ctx)
//
// 详见 docs/dag-patterns.md。
func BuildCleanerWorkflow() error {
	// TODO Phase 2
	return nil
}

func BuildReviewerWorkflow() error {
	// TODO Phase 4
	return nil
}
