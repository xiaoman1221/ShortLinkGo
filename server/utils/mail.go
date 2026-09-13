// Package utils 提供通用工具：统一响应、密码哈希、JWT、邮件发送等。
package utils

import (
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strconv"
	"strings"
)

// SMTPConfig SMTP 邮件配置。
type SMTPConfig struct {
	Host string
	Port string
	User string
	Pass string
	From string
}

// Enabled 判断是否已配置 SMTP。
func (c SMTPConfig) Enabled() bool {
	return c.Host != ""
}

// SendMail 发送一封纯文本邮件。
// 端口 465 使用隐式 TLS；25/587 优先 STARTTLS，无 TLS 时回退明文（本地调试）。
func SendMail(cfg SMTPConfig, to, subject, body string) error {
	if !cfg.Enabled() {
		return fmt.Errorf("SMTP 未配置")
	}
	// 拒绝含换行的地址，防止 SMTP 头/命令注入
	if strings.ContainsAny(to, "\r\n") || strings.ContainsAny(cfg.From, "\r\n") {
		return fmt.Errorf("邮件地址不合法")
	}
	if cfg.From == "" {
		cfg.From = cfg.User
	}
	if cfg.From == "" {
		return fmt.Errorf("发件人地址为空")
	}
	port := strings.TrimSpace(cfg.Port)
	if port == "" {
		port = "465"
	}
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("SMTP 端口无效: %s", port)
	}
	addr := net.JoinHostPort(cfg.Host, port)

	var conn net.Conn
	if portNum == 465 {
		conn, err = tls.Dial("tcp", addr, &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12})
	} else {
		conn, err = net.Dial("tcp", addr)
	}
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return err
	}
	defer client.Close()

	if portNum != 465 {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
				return err
			}
		}
	}

	if cfg.User != "" {
		auth := smtp.PlainAuth("", cfg.User, cfg.Pass, cfg.Host)
		if err := client.Auth(auth); err != nil {
			return err
		}
	}

	if err := client.Mail(cfg.From); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	msg := buildMessage(cfg.From, to, subject, body)
	if _, err := w.Write([]byte(msg)); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

// buildMessage 组装简单 MIME 文本邮件。
func buildMessage(from, to, subject, body string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", from)
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Subject: %s\r\n", mime.QEncoding.Encode("UTF-8", subject))
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	b.WriteString("\r\n")
	return b.String()
}
