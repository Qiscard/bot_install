package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OneBotClient 是 OneBot v11 HTTP 接口的最小客户端。
// SnowLuma 与 NapCat 均兼容该协议，文件类接口一致。
type OneBotClient struct {
	baseURL string
	token   string
	http    *http.Client
}

// NewOneBotClient 创建客户端。baseURL 例如 http://snowluma:3000
func NewOneBotClient(baseURL, token string) *OneBotClient {
	return &OneBotClient{
		baseURL: baseURL,
		token:   token,
		http: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// apiResponse OneBot 通用响应包裹
type apiResponse struct {
	Status  string          `json:"status"`
	RetCode int             `json:"retcode"`
	Data    json.RawMessage `json:"data"`
	Message string          `json:"message"`
	Wording string          `json:"wording"`
}

// call 调用指定 action，params 为请求体
func (c *OneBotClient) call(ctx context.Context, action string, params map[string]any, out any) error {
	body, err := json.Marshal(params)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/%s", c.baseURL, action)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("调用 %s 失败: %w", action, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s 返回 HTTP %d: %s", action, resp.StatusCode, string(raw))
	}

	var ar apiResponse
	if err := json.Unmarshal(raw, &ar); err != nil {
		return fmt.Errorf("解析 %s 响应失败: %w", action, err)
	}
	if ar.RetCode != 0 && ar.Status != "ok" {
		return fmt.Errorf("%s 业务错误 retcode=%d: %s %s", action, ar.RetCode, ar.Message, ar.Wording)
	}
	if out != nil && len(ar.Data) > 0 {
		return json.Unmarshal(ar.Data, out)
	}
	return nil
}

// FileInfo get_file / get_image / get_record 的返回
type FileInfo struct {
	File     string `json:"file"`      // 本地绝对路径
	FileName string `json:"file_name"` // 文件名
	FileSize string `json:"file_size"` // 字节数（字符串）
	Base64   string `json:"base64"`    // 可选 base64
	URL      string `json:"url"`       // 可选下载 URL
}

// GetFile 通过 file_id 获取文件本地路径信息
func (c *OneBotClient) GetFile(ctx context.Context, fileID string) (*FileInfo, error) {
	var fi FileInfo
	if err := c.call(ctx, "get_file", map[string]any{"file_id": fileID, "file": fileID}, &fi); err != nil {
		return nil, err
	}
	return &fi, nil
}

// GetImage 通过 file 获取图片本地路径信息
func (c *OneBotClient) GetImage(ctx context.Context, file string) (*FileInfo, error) {
	var fi FileInfo
	if err := c.call(ctx, "get_image", map[string]any{"file": file}, &fi); err != nil {
		return nil, err
	}
	return &fi, nil
}

// GetRecord 通过 file 获取语音本地路径信息
func (c *OneBotClient) GetRecord(ctx context.Context, file, outFormat string) (*FileInfo, error) {
	if outFormat == "" {
		outFormat = "mp3"
	}
	var fi FileInfo
	if err := c.call(ctx, "get_record", map[string]any{"file": file, "out_format": outFormat}, &fi); err != nil {
		return nil, err
	}
	return &fi, nil
}

// Ping 通过 get_status 探活
func (c *OneBotClient) Ping(ctx context.Context) error {
	return c.call(ctx, "get_status", map[string]any{}, nil)
}
