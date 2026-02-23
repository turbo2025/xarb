package notification

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
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// Client 飞书机器人客户端
type Client struct {
	webhook string
	secret  string
	channel string
	httpcli *http.Client
}

// NewClient 创建飞书客户端
func NewClient(webhook, secret, channel string) *Client {
	return &Client{
		webhook: webhook,
		secret:  secret,
		channel: channel,
		httpcli: &http.Client{Timeout: 10 * time.Second},
	}
}

// Message 飞书消息体
type Message struct {
	MsgType string      `json:"msg_type"`
	Content interface{} `json:"content"`
}

// TextContent 文本消息内容
type TextContent struct {
	Text string `json:"text"`
}

// PostContent 富文本消息内容
type PostContent struct {
	Post *PostPost `json:"post"`
}

type PostPost struct {
	ZhCN *PostZhCN `json:"zh_cn"`
}

type PostZhCN struct {
	Title   string        `json:"title"`
	Content [][]PostBlock `json:"content"`
}

type PostBlock struct {
	Tag  string `json:"tag"`
	Text string `json:"text,omitempty"`
	Href string `json:"href,omitempty"`
}

// SendText 发送纯文本消息
func (c *Client) SendText(text string) error {
	msg := Message{
		MsgType: "text",
		Content: TextContent{Text: text},
	}
	return c.send(msg)
}

// SendPost 发送富文本消息
func (c *Client) SendPost(title string, content []string) error {
	// 过滤掉空行
	var nonEmptyLines []string
	for _, line := range content {
		if strings.TrimSpace(line) != "" {
			nonEmptyLines = append(nonEmptyLines, line)
		}
	}

	// 如果没有非空行，添加一个默认内容
	if len(nonEmptyLines) == 0 {
		nonEmptyLines = []string{"(No additional details)"}
	}

	blocks := make([][]PostBlock, len(nonEmptyLines))
	for i, line := range nonEmptyLines {
		blocks[i] = []PostBlock{
			{
				Tag:  "text",
				Text: line,
			},
		}
	}

	msg := Message{
		MsgType: "post",
		Content: PostContent{
			Post: &PostPost{
				ZhCN: &PostZhCN{
					Title:   title,
					Content: blocks,
				},
			},
		},
	}
	return c.send(msg)
}

// send 发送消息到飞书
func (c *Client) send(msg Message) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message failed: %w", err)
	}

	// 计算签名（如果提供了 secret）
	urlWH := c.webhook
	if c.secret != "" {
		sign, err := c.genSign(c.secret, time.Now())
		if err != nil {
			return fmt.Errorf("gen sign failed: %w", err)
		}
		ts := strconv.FormatInt(time.Now().Unix(), 10)

		// 检查 webhook URL 中是否已有查询参数
		separator := "?"
		if strings.Contains(c.webhook, "?") {
			separator = "&"
		}
		urlWH = c.webhook + separator + "timestamp=" + ts + "&sign=" + url.QueryEscape(sign)
	}

	req, err := http.NewRequest("POST", urlWH, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("new request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpcli.Do(req)
	if err != nil {
		log.Error().Err(err).Str("channel", c.channel).Msg("send feishu message failed")
		return fmt.Errorf("send request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Warn().
			Str("channel", c.channel).
			Int("status_code", resp.StatusCode).
			RawJSON("response", respBody).
			Msg("feishu api returned non-200 status")
		return fmt.Errorf("feishu api returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		log.Warn().Err(err).RawJSON("response", respBody).Msg("failed to parse feishu response")
		return nil
	}

	// 检查飞书 API 返回的 code
	if code, ok := result["code"].(float64); ok && code != 0 {
		log.Warn().
			Str("channel", c.channel).
			Float64("code", code).
			Interface("msg", result).
			Msg("feishu api returned error")
		return fmt.Errorf("feishu api error code: %d", int(code))
	}

	log.Debug().Str("channel", c.channel).Msg("feishu message sent successfully")
	return nil
}

// genSign 生成飞书签名
func (c *Client) genSign(secret string, timestamp time.Time) (string, error) {
	ts := strconv.FormatInt(timestamp.Unix(), 10)
	str := ts + "\n" + secret
	h := hmac.New(sha256.New, []byte(str))
	h.Write([]byte(""))
	sign := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return sign, nil
}
