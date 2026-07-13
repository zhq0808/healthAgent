package service

import (
	"fmt"

	"github.com/google/uuid"
)

// NewMessageID 生成一个 RFC 9562 UUIDv7 字符串，作为一条消息的唯一业务身份。
//
// 为什么用 UUIDv7 而不是自增 BIGINT 或 UUIDv4：
//   - 高 48 位是毫秒级时间戳，天然按生成时间近似有序，写入 B-tree 有较好局部性，
//     不像 UUIDv4 那样随机分散导致页分裂；
//   - 低位是随机数，保证跨进程/跨库唯一，拆库拆服务时不需要再做 ID 迁移；
//   - 由后端生成，业务身份在写库前就已确定，日志和跨表关联可以立即使用。
//
// 数据库对 message_id 仍保留 UNIQUE 约束作为最终兜底；调用方在命中唯一冲突时
// 应重新生成并有限重试（见 store 层的保存点重试）。
func NewMessageID() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("生成 UUIDv7 message_id 失败: %w", err)
	}
	return id.String(), nil
}
