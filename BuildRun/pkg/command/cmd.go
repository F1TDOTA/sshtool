package command

import "encoding/json"

type JsonCmd1 struct {
	OperAction string          `json:"oper_action"`
	Data       json.RawMessage `json:"data"`
}

type FileCmd1 struct {
	OperType string `json:"oper_type"`
	SrcHost  string `json:"src_host"`
	SrcPath  string `json:"src_path"`
	DstHost  string `json:"dst_host"`
	DstPath  string `json:"dst_path"`
}

type ToastCmd1 struct {
	AppId   string `json:"app_id"`
	Message string `json:"message"`
}
