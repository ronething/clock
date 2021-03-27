package param

const (
	Success = 0
	Failed  = -1
)

// 请求
type (
	Page struct {
		Count int `query:"count" json:"count"`
		Index int `query:"index" json:"index"`
		Total int `json:"total"`
	}
)

// 返回
type (
	ApiResponse struct {
		Code int         `json:"code"`
		Msg  string      `json:"msg"`
		Data interface{} `json:"data"`
	}

	// 分页返回的请求体
	ListResponse struct {
		Items     interface{} `json:"items"`
		PageQuery interface{} `json:"page"`
	}

	DisableTask struct {
		Disable bool `json:"disable"`
	}

	//新的修改任务 api
	NewSpecTask struct {
		Expression string `json:"expression"`
		Timezone   string `json:"timezone"`
	}
)

//BuildResp 通用 resp
func BuildResp() ApiResponse {
	return ApiResponse{
		Code: Success, // 成功为 0 不成功为 -1
		Msg:  "success",
		Data: nil,
	}
}
