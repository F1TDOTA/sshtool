package server

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// 你的机器人 Webhook（不含 sign/timestamp 参数）和加签密钥
const (
	webhookBase = "https://oapi.dingtalk.com/robot/send?access_token=cb3fd7742ea2934843471c4aed16c677bc1e8e5eb90108454644ba6bbbe559c9"
	secret      = "SECe4ba8caf46df6773477a394db6ee79d7f675b7710d7a696364baad6954c0c066"
)

type JsonDingCmd struct {
	OperType string `json:"oper_type"`
	MdTitle  string `json:"md_title"`
	Message  string `json:"message"`
}

type DdNoticeService struct {
	dingCh   chan string
	toastSvr *ToastService
}

func NewDdNoticeService(toastSvr *ToastService) *DdNoticeService {
	return &DdNoticeService{
		dingCh:   make(chan string),
		toastSvr: toastSvr,
	}
}

// 计算加签
func (d *DdNoticeService) makeSign(secret string, ts int64) (string, error) {
	strToSign := fmt.Sprintf("%d\n%s", ts, secret)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strToSign))
	raw := mac.Sum(nil)
	sig := base64.StdEncoding.EncodeToString(raw)
	// DingTalk 要求 URL 编码
	return url.QueryEscape(sig), nil
}

// 发送原始 JSON
func (d *DdNoticeService) postJSON(fullURL string, body any) error {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", fullURL, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	cli := &http.Client{Timeout: 10 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)

	// DingTalk 返回形如：{"errcode":0,"errmsg":"ok"}
	var rr struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(rb, &rr); err != nil {
		return fmt.Errorf("decode resp fail: %v, body=%s", err, string(rb))
	}
	if rr.ErrCode != 0 {
		return fmt.Errorf("dingtalk error: %d %s", rr.ErrCode, rr.ErrMsg)
	}
	return nil
}

// 发送文本消息
func (d *DdNoticeService) sendText(content string, atMobiles []string, isAtAll bool) error {
	ts := time.Now().UnixMilli()
	sign, _ := d.makeSign(secret, ts)
	fullURL := fmt.Sprintf("%s&timestamp=%d&sign=%s", webhookBase, ts, sign)

	payload := map[string]any{
		"msgtype": "text",
		"text": map[string]string{
			"content": content,
		},
		"at": map[string]any{
			"atMobiles": atMobiles,
			"isAtAll":   isAtAll,
		},
	}
	return d.postJSON(fullURL, payload)
}

// 发送 Markdown 消息
func (d *DdNoticeService) sendMarkdown(title, markdown string, atMobiles []string, isAtAll bool) error {
	ts := time.Now().UnixMilli()
	sign, _ := d.makeSign(secret, ts)
	fullURL := fmt.Sprintf("%s&timestamp=%d&sign=%s", webhookBase, ts, sign)

	payload := map[string]any{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": title,
			"text":  markdown,
		},
		"at": map[string]any{
			"atMobiles": atMobiles,
			"isAtAll":   isAtAll,
		},
	}

	return d.postJSON(fullURL, payload)
}

func (d *DdNoticeService) execute(msg string) {
	var jsonDingObj JsonDingCmd
	err := json.Unmarshal([]byte(msg), &jsonDingObj)
	if err != nil {
		fmt.Printf("DdNoticeService json unmarshal err %v \n", err)
		return
	}

	operType := jsonDingObj.OperType
	msgContent := jsonDingObj.Message
	mdTitle := jsonDingObj.MdTitle

	switch operType {
	case "text":
		if err := d.sendText(msgContent, nil, false); err != nil {
			fmt.Printf("ding ding SendText %s error: %s", msg, err)
		}
	case "markdown":
		if err := d.sendMarkdown(mdTitle, msgContent, nil, false); err != nil {
			fmt.Println("SendMarkdown error:", err)
		}
	default:
		fmt.Printf("unknown oper_type: %s\n", operType)
		break
	}
}

func (d *DdNoticeService) Run() {
	go func() {
		for {
			select {
			case msg := <-d.dingCh:
				fmt.Println("Receive ding ding msg:", msg)
				d.execute(msg)
			}
		}
	}()
}

func (d *DdNoticeService) SendDingMsg(msg string) {
	d.dingCh <- msg
}

func (d *DdNoticeService) HandleCommand(cmdJson JsonCmd) {
	//命令格式如下：
	//{"oper_action":"notify_dd","oper_type": "text", "warn_msg":"test message"}
	msg := cmdJson.WarnMsg
	operType := cmdJson.OperType
	
	// 发消息给自己
	var jsonDing = &JsonDingCmd{
		OperType: operType,
		Message:  msg,
		MdTitle:  "钉钉告警",
	}

	jsonDingStr, err := json.Marshal(jsonDing)
	if err != nil {
		fmt.Printf("jsonToast marshal err %s\n", err)
	}
	d.SendDingMsg(string(jsonDingStr))

	// 发消息给Toast服务
	var jsonToast = &JsonToastCmd{
		AppId:   "钉钉告警",
		Message: msg,
	}

	jsonStr, err := json.Marshal(jsonToast)
	if err != nil {
		fmt.Printf("jsonToast marshal err %s\n", err)
	}

	d.toastSvr.SendToastMsg(string(jsonStr))

}
