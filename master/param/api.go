package param

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
)

//BuildResp 通用 resp
func BuildResp() ApiResponse {
	return ApiResponse{
		Code: 200,
		Msg:  "success",
		Data: nil,
	}
}
