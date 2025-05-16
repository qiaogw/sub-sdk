package notifyx

import (
	"context"
	"fmt"
	"github.com/go-gomail/gomail"
	"github.com/zeromicro/go-zero/core/logx"
	"strings"
	"time"
)

type Mail struct {
	ctx    context.Context
	option *Option
}
type Option struct {
	Host      string     `json:"host"`
	Port      int        `json:"port"`
	User      string     `json:"user"`
	Password  string     `json:"password"`
	MailUsers []MailUser `json:"mail_users"`
	Template  string     `json:"template"`
}
type MailUser struct {
	Id       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

func NewMail(svcCtx context.Context, option *Option) *Mail {
	return &Mail{
		ctx:    svcCtx,
		option: option,
	}
}

func (m *Mail) Send(msg Message) error {

	if m.option.Host == "" {
		return fmt.Errorf("#mail#Host为空")
	}
	if m.option.Port == 0 {
		return fmt.Errorf("#mail#Port为空")

	}
	if m.option.User == "" {
		return fmt.Errorf("#mail#User为空")

	}
	if m.option.Password == "" {
		return fmt.Errorf("#mail#Password为空")
	}
	msg["content"] = parseNotifyTemplate(m.option.Template, msg)
	toUsers := m.getActiveMailUsers(msg)
	m.send(toUsers, msg)
	return nil
}

func (m *Mail) send(toUsers []string, msg Message) {
	body := msg["content"].(string)
	body = strings.Replace(body, "\n", "<br>", -1)
	gomailMessage := gomail.NewMessage()
	gomailMessage.SetHeader("From", m.option.User)
	gomailMessage.SetHeader("To", toUsers...)
	gomailMessage.SetHeader("Subject", "gocron-定时任务通知")
	gomailMessage.SetBody("text/html", body)
	mailer := gomail.NewDialer(m.option.Host, m.option.Port,
		m.option.User, m.option.Password)

	maxTimes := 3
	i := 0

	for i < maxTimes {
		err := mailer.DialAndSend(gomailMessage)
		if err == nil {
			logx.Debugf("🐛 发送消息成功，发送人：%v", toUsers)
			return
		}
		i++
		time.Sleep(2 * time.Second)
		logx.Errorf("❌发送消息失败：%v", err)
	}
	logx.Errorf("❌达到最大重试次数，发送消息失败")
}

func (m *Mail) getActiveMailUsers(msg Message) []string {
	taskReceiverIds := strings.Split(msg["task_receiver_id"].(string), ",")
	var users []string
	for _, v := range m.option.MailUsers {
		if InStringSlice(taskReceiverIds, v.Id) {
			users = append(users, v.Email)
		}
	}
	return users
}
