// Package llm 提供大模型底层调用能力（DeepSeek，OpenAI 兼容 /chat/completions 协议）。
//
// 职责单一：只负责 HTTP 传输——把一组消息发给模型、拿回原始文本。
// 不关心系统人设、提示词拼装、意图解析或降级策略，那些属于上层 service。
package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DeepSeekClient 是 DeepSeek（OpenAI 兼容）对话客户端。
//
// 并发安全：底层 http.Client 可被多个 goroutine 共用，DeepSeekClient 自身无可变状态。
type DeepSeekClient struct {
	apiKey  string
	baseURL string
	model   string
	timeout time.Duration
	http    *http.Client
}

// NewDeepSeekClient 构造 DeepSeek 客户端。timeout 为单次请求超时上限，同时作为兜底硬超时挂在 http.Client 上。
func NewDeepSeekClient(apiKey, baseURL, model string, timeout time.Duration) *DeepSeekClient {
	return &DeepSeekClient{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		timeout: timeout,
		http:    &http.Client{Timeout: timeout},
	}
}

// Timeout 返回单次调用的超时上限，供调用方构造带 deadline 的 context。
func (c *DeepSeekClient) Timeout() time.Duration {
	return c.timeout
}

// Stream 以流式方式调用大模型：DeepSeek 每生成一段文本就回调 onDelta。
//
// onDelta 返回 error 时（例如下游客户端已断开、写回失败）立即停止读取并把该 error 透传出去，
// 避免在客户端早退后还空转把整段响应读完。ctx 取消时底层 HTTP 请求也会被中断。
func (c *DeepSeekClient) Stream(ctx context.Context, messages []Message, onDelta func(delta string) error) error {
	if c == nil || c.apiKey == "" {
		return ErrNotConfigured
	}

	body, err := json.Marshal(chatCompletionRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   true,
	})
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("调用大模型失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 出错时响应通常是普通 JSON（非 SSE），读一段带上下文便于排查。
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
		return fmt.Errorf("大模型返回非 200 状态: %d, body: %s", resp.StatusCode, raw)
	}

	// SSE 逐行读：每行形如 `data: {json}`，中间可能有空行/注释行，流结束是 `data: [DONE]`。
	scanner := bufio.NewScanner(resp.Body)
	// 放大单行缓冲上限，防止某一帧过长触发 bufio.ErrTooLong。
	scanner.Buffer(make([]byte, 0, 64*1024), maxResponseBytes)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue // 跳过空行、注释行(: keep-alive)等
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			break
		}

		var chunk chatCompletionChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			return fmt.Errorf("解析流式分片失败: %w", err)
		}
		if chunk.Error != nil {
			return fmt.Errorf("大模型返回错误: %s", chunk.Error.Message)
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta.Content
		if delta == "" {
			continue // 首帧常只有 role 没有内容
		}
		if err := onDelta(delta); err != nil {
			return err // 下游写回失败/客户端断开，主动停止
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取流式响应失败: %w", err)
	}
	return nil
}
