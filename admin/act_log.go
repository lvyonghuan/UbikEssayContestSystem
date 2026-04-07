package admin

import (
	"main/database/pgsql"
	"main/model"
	"main/util/log"
)

// 操作日志引擎
// 管理员操作日志的主要构成分为两个部分：
// 资源和动作

func newAdminActionLog(adminID int, res, act string, details map[string]interface{}) {
	actionLog := model.ActionLog{
		AdminID:  adminID,
		Resource: res,
		Action:   act,
		Details:  details,
	}

	err := pgsql.CreateActionLog(actionLog)
	if err != nil {
		log.Logger.Error(err)
	}
}

// genDetails 生成操作日志的详情信息
func genDetails(key, value []string) map[string]interface{} {
	details := make(map[string]interface{})
	for i := 0; i < len(key) && i < len(value); i++ {
		details[key[i]] = value[i]
	}

	return details
}
