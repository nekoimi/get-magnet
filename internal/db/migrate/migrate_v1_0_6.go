package migrate

import "xorm.io/xorm"

type magnetDownloadStatus struct {
}

func init() {
	registerMigrate(new(magnetDownloadStatus))
}

func (m *magnetDownloadStatus) Version() int64 {
	return 2026_07_22_001
}

func (m *magnetDownloadStatus) Desc() string {
	return "为磁力信息增加下载调度状态字段并将历史数据标记为已完成"
}

func (m *magnetDownloadStatus) Exec(e *xorm.Engine) error {
	_, err := e.Exec(`
ALTER TABLE magnets ADD COLUMN IF NOT EXISTS download_error text NOT NULL DEFAULT '';
ALTER TABLE magnets ADD COLUMN IF NOT EXISTS download_retry_count integer NOT NULL DEFAULT 0;
ALTER TABLE magnets ADD COLUMN IF NOT EXISTS last_submit_at timestamp NULL;
ALTER TABLE magnets ADD COLUMN IF NOT EXISTS download_completed_at timestamp NULL;

UPDATE magnets
SET followed_by = ''
WHERE followed_by = 'unknow';

UPDATE magnets
SET status = 3,
    post_process_done = true,
    download_error = '',
    download_completed_at = COALESCE(updated_at, created_at, NOW());
`)
	return err
}
