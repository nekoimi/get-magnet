package magnet_repo

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nekoimi/get-magnet/internal/db"
	"github.com/nekoimi/get-magnet/internal/db/table"
	log "github.com/sirupsen/logrus"
)

func Save(m *table.Magnets) {
	_, err := db.Instance().InsertOne(m)
	if err != nil {
		log.Errorf("保存资源异常：%s", err.Error())
		return
	}
}

func ExistsByPath(rowURLPath string) bool {
	m := new(table.Magnets)
	m.RawURLPath = rowURLPath
	if exist, err := db.Instance().Exist(m); err != nil {
		log.Errorf("查询资源Path(%s)是否存在异常：%s", rowURLPath, err.Error())
		return false
	} else {
		return exist
	}
}

func ExistsByNumber(number string) bool {
	m := new(table.Magnets)
	m.Number = strings.ToUpper(number)
	if exist, err := db.Instance().Exist(m); err != nil {
		log.Errorf("查询资源Number(%s)是否存在异常：%s", number, err.Error())
		return false
	} else {
		return exist
	}
}

func GetByFollowed(followedBy string) (*table.Magnets, bool) {
	m := new(table.Magnets)
	m.FollowedBy = followedBy
	if has, err := db.Instance().Get(m); err != nil {
		log.Errorf("查询资源FollowedBy(%s)异常：%s", followedBy, err.Error())
		return nil, false
	} else {
		return m, has
	}
}

func UpdateFollowedBy(source string, target string) error {
	m, exists := GetByFollowed(source)
	if !exists {
		// 忽略
		return errors.New(fmt.Sprintf("查询资源FollowedBy(%s)不存在", source))
	}

	m.FollowedBy = target
	m.PostProcessDone = false

	if _, err := db.Instance().ID(m.Id).Cols("followed_by", "post_process_done").Update(m); err != nil {
		return err
	}

	return nil
}

func MarkPostProcessDone(id int64) error {
	m := &table.Magnets{
		Id:              id,
		PostProcessDone: true,
	}
	if _, err := db.Instance().ID(id).Cols("post_process_done").Update(m); err != nil {
		log.Errorf("更新资源下载后处理状态异常：%s", err.Error())
		return err
	}
	return nil
}

func MarkPostProcessDoneWithPlayInfo(id int64, playFileID, playFilePath string, playFileSize int64, strmPath string) error {
	m := &table.Magnets{
		Id:              id,
		PostProcessDone: true,
		PlayFileID:      playFileID,
		PlayFilePath:    playFilePath,
		PlayFileSize:    playFileSize,
		STRMPath:        strmPath,
	}
	if _, err := db.Instance().ID(id).Cols("post_process_done", "play_file_id", "play_file_path", "play_file_size", "strm_path").Update(m); err != nil {
		log.Errorf("更新资源下载后处理播放信息异常：%s", err.Error())
		return err
	}
	return nil
}

func MarkPostProcessDoneByFollowed(followedBy string) error {
	m, exists := GetByFollowed(followedBy)
	if !exists {
		return errors.New(fmt.Sprintf("查询资源FollowedBy(%s)不存在", followedBy))
	}
	return MarkPostProcessDone(m.Id)
}

func ListPendingPostProcess(limit int) ([]table.Magnets, error) {
	var list []table.Magnets
	err := db.Instance().
		Where("followed_by <> ''").
		And("followed_by <> ?", "unknow").
		And("status = ?", table.MagnetStatusDownloading).
		And("post_process_done = ?", false).
		Limit(limit).
		Find(&list)
	if err != nil {
		log.Errorf("查询未完成下载后处理资源异常：%s", err.Error())
		return nil, err
	}
	return list, nil
}

func ListPendingDownload(limit int, maxRetry int) ([]table.Magnets, error) {
	var list []table.Magnets
	session := db.Instance().
		Where("optimal_link <> ''").
		And("(followed_by = '' OR followed_by IS NULL OR followed_by = ?)", "unknow").
		And("(status = ? OR (status = ? AND download_retry_count < ?))",
			table.MagnetStatusCollected,
			table.MagnetStatusFailed,
			maxRetry,
		).
		OrderBy("created_at ASC")
	if limit > 0 {
		session = session.Limit(limit)
	}
	err := session.Find(&list)
	if err != nil {
		log.Errorf("查询待提交下载资源异常：%s", err.Error())
		return nil, err
	}
	return list, nil
}

func MarkDownloadSubmitting(id int64, maxRetry int) (bool, error) {
	now := time.Now()
	m := &table.Magnets{
		Status:       table.MagnetStatusSubmitting,
		LastSubmitAt: &now,
	}
	affected, err := db.Instance().
		Where("id = ?", id).
		And("(status = ? OR (status = ? AND download_retry_count < ?))",
			table.MagnetStatusCollected,
			table.MagnetStatusFailed,
			maxRetry,
		).
		Cols("status", "last_submit_at").
		Update(m)
	if err != nil {
		log.Errorf("领取下载资源异常：%d - %s", id, err.Error())
		return false, err
	}
	return affected > 0, nil
}

func MarkDownloadSubmitted(id int64, followedBy string) error {
	now := time.Now()
	m := &table.Magnets{
		Status:          table.MagnetStatusDownloading,
		FollowedBy:      followedBy,
		PostProcessDone: false,
		DownloadError:   "",
		LastSubmitAt:    &now,
	}
	if _, err := db.Instance().
		ID(id).
		Cols("status", "followed_by", "post_process_done", "download_error", "last_submit_at").
		Update(m); err != nil {
		log.Errorf("标记资源已提交下载异常：%d - %s", id, err.Error())
		return err
	}
	return nil
}

func MarkDownloadSubmitFailed(id int64, submitErr error) error {
	message := ""
	if submitErr != nil {
		message = submitErr.Error()
	}
	if _, err := db.Instance().Exec(
		"UPDATE magnets SET status = ?, download_retry_count = download_retry_count + 1, download_error = ?, updated_at = NOW() WHERE id = ?",
		table.MagnetStatusFailed,
		truncateDownloadError(message),
		id,
	); err != nil {
		log.Errorf("标记资源提交下载失败异常：%d - %s", id, err.Error())
		return err
	}
	return nil
}

func MarkDownloadCompletedByFollowed(followedBy string) error {
	now := time.Now()
	m := &table.Magnets{
		Status:              table.MagnetStatusCompleted,
		PostProcessDone:     true,
		DownloadError:       "",
		DownloadCompletedAt: &now,
	}
	if _, err := db.Instance().
		Where("followed_by = ?", followedBy).
		Cols("status", "post_process_done", "download_error", "download_completed_at").
		Update(m); err != nil {
		log.Errorf("标记资源下载完成异常：%s - %s", followedBy, err.Error())
		return err
	}
	return nil
}

func MarkDownloadFailedByFollowed(followedBy string, reason string) error {
	if followedBy == "" {
		return nil
	}
	if _, err := db.Instance().Exec(
		"UPDATE magnets SET status = ?, download_retry_count = download_retry_count + 1, download_error = ?, updated_at = NOW() WHERE followed_by = ?",
		table.MagnetStatusFailed,
		truncateDownloadError(reason),
		followedBy,
	); err != nil {
		log.Errorf("标记资源下载失败异常：%s - %s", followedBy, err.Error())
		return err
	}
	return nil
}

func truncateDownloadError(message string) string {
	message = strings.TrimSpace(message)
	if len(message) <= 2000 {
		return message
	}
	return message[:2000]
}

// GetByNumber 根据番号获取磁力链接
func GetByNumber(number string) (*table.Magnets, bool) {
	m := new(table.Magnets)
	m.Number = strings.ToUpper(number)
	if has, err := db.Instance().Get(m); err != nil {
		log.Errorf("查询资源Number(%s)异常：%s", number, err.Error())
		return nil, false
	} else {
		return m, has
	}
}

// GetById 根据 ID 获取磁力链接
func GetById(id int64) (*table.Magnets, bool) {
	m := new(table.Magnets)
	if has, err := db.Instance().ID(id).Get(m); err != nil {
		log.Errorf("查询资源ID(%d)异常：%s", id, err.Error())
		return nil, false
	} else {
		return m, has
	}
}

// PageList 分页查询磁力链接列表
func PageList(pageNum, pageSize int, keyword string, status *uint8) ([]table.Magnets, int64, error) {
	session := db.Instance().NewSession()
	defer session.Close()

	// 构建查询条件
	if keyword != "" {
		session = session.Where("(title LIKE ? OR number LIKE ?)", "%"+keyword+"%", "%"+keyword+"%")
	}
	if status != nil {
		session = session.Where("status = ?", *status)
	}

	// 获取总数
	total, err := session.Count(new(table.Magnets))
	if err != nil {
		log.Errorf("查询磁力链接总数异常：%s", err.Error())
		return nil, 0, err
	}

	// 分页查询
	var list []table.Magnets
	err = session.OrderBy("created_at DESC").Limit(pageSize, (pageNum-1)*pageSize).Find(&list)
	if err != nil {
		log.Errorf("查询磁力链接列表异常：%s", err.Error())
		return nil, 0, err
	}

	return list, total, nil
}

// Update 更新磁力链接
func Update(m *table.Magnets) error {
	_, err := db.Instance().ID(m.Id).AllCols().Update(m)
	if err != nil {
		log.Errorf("更新磁力链接异常：%s", err.Error())
		return err
	}
	return nil
}

// Delete 删除磁力链接
func Delete(id int64) error {
	_, err := db.Instance().ID(id).Delete(new(table.Magnets))
	if err != nil {
		log.Errorf("删除磁力链接异常：%s", err.Error())
		return err
	}
	return nil
}

// BatchDelete 批量删除磁力链接
func BatchDelete(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := db.Instance().In("id", ids).Delete(new(table.Magnets))
	if err != nil {
		log.Errorf("批量删除磁力链接异常：%s", err.Error())
		return err
	}
	return nil
}
