package service

import (
	"encoding/json"
	"ferry/global/orm"
	"ferry/models/process"
	"ferry/pkg/pagination"
	"ferry/tools"
	"fmt"

	"github.com/gin-gonic/gin"
)

/*
  @Author : lanyulei
  @todo: 添加新的处理人时候，需要修改（先完善功能，后续有时间的时候优化一下这部分。）
*/

type WorkOrder struct {
	Classify int
	GinObj   *gin.Context
}

type workOrderInfo struct {
	process.WorkOrderInfo
	Principals   string `json:"principals"`
	DataClassify int    `json:"data_classify"`
}

func (w *WorkOrder) PureWorkOrderList() (result interface{}, err error) {
	var (
		workOrderInfoList []workOrderInfo
	)

	title := w.GinObj.DefaultQuery("title", "")
	db := orm.Eloquent.Model(&process.WorkOrderInfo{}).Where("title like ?", fmt.Sprintf("%%%v%%", title))

	// 获取当前用户信息
	switch w.Classify {
	case 1:
		// 待办工单
		// 1. 个人
		personSelect := fmt.Sprintf("(JSON_CONTAINS(state, JSON_OBJECT('processor', %v)) and JSON_CONTAINS(state, JSON_OBJECT('process_method', 'person')))", tools.GetUserId(w.GinObj))

		// 2. 小组
		//groupList := make([]int, 0)
		//err = orm.Eloquent.Model(&user.UserGroup{}).
		//	Where("user = ?", tools.GetUserId(c)).
		//	Pluck("`group`", &groupList).Error
		//if err != nil {
		//	return
		//}
		//groupSqlList := make([]string, 0)
		//if len(groupList) > 0 {
		//	for _, group := range groupList {
		//		groupSqlList = append(groupSqlList, fmt.Sprintf("JSON_CONTAINS(state, JSON_OBJECT('processor', %v))", group))
		//	}
		//} else {
		//	groupSqlList = append(groupSqlList, fmt.Sprintf("JSON_CONTAINS(state, JSON_OBJECT('processor', 0))"))
		//}
		//
		//personGroupSelect := fmt.Sprintf(
		//	"((%v) and %v)",
		//	strings.Join(groupSqlList, " or "),
		//	"JSON_CONTAINS(state, JSON_OBJECT('process_method', 'persongroup'))",
		//)

		// 3. 部门
		//departmentSelect := fmt.Sprintf("(JSON_CONTAINS(state, JSON_OBJECT('processor', %v)) and JSON_CONTAINS(state, JSON_OBJECT('process_method', 'department')))", userInfo.Dept)

		// 4. 变量会转成个人数据

		//db = db.Where(fmt.Sprintf("(%v or %v or %v or %v) and is_end = 0", personSelect, personGroupSelect, departmentSelect, variableSelect))
		db = db.Where(fmt.Sprintf("(%v) and is_end = 0", personSelect))
	case 2:
		// 我创建的
		db = db.Where("creator = ?", tools.GetUserId(w.GinObj))
	case 3:
		// 我相关的
		db = db.Where(fmt.Sprintf("JSON_CONTAINS(related_person, '%v')", tools.GetUserId(w.GinObj)))
	case 4:
	// 所有工单
	default:
		return nil, fmt.Errorf("请确认查询的数据类型是否正确")
	}

	result, err = pagination.Paging(&pagination.Param{
		C:  w.GinObj,
		DB: db,
	}, &workOrderInfoList)
	if err != nil {
		err = fmt.Errorf("查询工单列表失败，%v", err.Error())
		return
	}
	return
}

func (w *WorkOrder) WorkOrderList() (result interface{}, err error) {

	var (
		principals string
		StateList  []map[string]interface{}
	)

	result, err = w.PureWorkOrderList()
	if err != nil {
		return
	}

	for i, w := range *result.(*pagination.Paginator).Data.(*[]workOrderInfo) {
		err = json.Unmarshal(w.State, &StateList)
		if err != nil {
			err = fmt.Errorf("json反序列化失败，%v", err.Error())
			return
		}
		if len(StateList) != 0 {
			processorList := make([]int, 0)
			for _, v := range StateList[0]["processor"].([]interface{}) {
				processorList = append(processorList, int(v.(float64)))
			}
			principals, err = GetPrincipal(processorList, StateList[0]["process_method"].(string))
			if err != nil {
				err = fmt.Errorf("查询处理人名称失败，%v", err.Error())
				return
			}
		}
		workOrderDetails := *result.(*pagination.Paginator).Data.(*[]workOrderInfo)
		workOrderDetails[i].Principals = principals
		workOrderDetails[i].DataClassify = w.Classify
	}

	return result, nil
}
