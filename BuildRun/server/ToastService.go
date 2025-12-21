package server

import (
	"encoding/json"
	"fmt"

	"github.com/go-toast/toast"
)

type JsonToastCmd struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

type ToastService struct {
	ToastCh chan string
}

func NewToastService(toastCh chan string) *ToastService {
	return &ToastService{
		ToastCh: toastCh,
	}
}

func (t *ToastService) executeToast(msg string) {

	var jsonToastObj JsonToastCmd
	err := json.Unmarshal([]byte(msg), &jsonToastObj)
	if err != nil {
		fmt.Printf("ToastService json unmarshal err %v \n", err)
		return
	}

	notification := toast.Notification{
		AppID:   "ssh统一管理工具",
		Title:   jsonToastObj.Title,
		Message: jsonToastObj.Message,
	}

	err = notification.Push()
	if err != nil {
		fmt.Println("notification push error:", err)
	}
}

func (t *ToastService) Run() {
	go func() {
		for {
			select {
			case msg := <-t.ToastCh:
				fmt.Println("Receive toast msg:", msg)
				t.executeToast(msg)
			}
		}
	}()
}

func (t *ToastService) SendToastMsg(msg string) {
	t.ToastCh <- msg
}

func (t *ToastService) HandleCommand(cmdJson JsonCmd) {
	//命令格式如下：
	//{"oper_action":"notify","warn_msg":"test message"}
	msg := cmdJson.WarnMsg

	var jsonToast = &JsonToastCmd{
		Title:   "自定义告警",
		Message: msg,
	}

	jsonStr, err := json.Marshal(jsonToast)
	if err != nil {
		fmt.Printf("jsonToast marshal err %s\n", err)
		return
	}

	t.SendToastMsg(string(jsonStr))
}

func (t *ToastService) stop() {

}
