package migrate

import "xorm.io/xorm"

// Existing definitions were executed in the browser when fetch had no mode.
// Pin their behavior before new definitions default to ordinary HTTP.
type legacyWorkflowFetchMode struct{}

func init()                                     { registerMigrate(new(legacyWorkflowFetchMode)) }
func (*legacyWorkflowFetchMode) Version() int64 { return 2026_09_26_001 }
func (*legacyWorkflowFetchMode) Desc() string {
	return "保留旧工作流版本的浏览器获取行为"
}
func (*legacyWorkflowFetchMode) Exec(e *xorm.Engine) error {
	_, err := e.Exec(`UPDATE workflow_versions
 SET definition = jsonb_set(definition,'{trigger,fetch}','{"mode":"browser"}'::jsonb,true)
 WHERE jsonb_typeof(definition->'trigger')='object'
   AND NOT jsonb_exists(definition->'trigger', 'fetch')`)
	return err
}
