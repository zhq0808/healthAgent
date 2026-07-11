package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"healthAgent/internal/service"
)

// PostgresMessageRepository 使用共享连接池持久化对话消息。
type PostgresMessageRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresMessageRepository(pool *pgxpool.Pool) *PostgresMessageRepository {
	return &PostgresMessageRepository{pool: pool}
}

// AppendUserMessage 在短事务内串行化同一 session 的 seq 分配，并保证客户端消息幂等。
func (r *PostgresMessageRepository) AppendUserMessage(ctx context.Context, request service.AppendUserMessageRequest) (service.AppendUserMessageResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return service.AppendUserMessageResult{}, fmt.Errorf("开启用户消息事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	result, err := appendUserMessageTx(ctx, tx, request)
	if err != nil {
		return service.AppendUserMessageResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return service.AppendUserMessageResult{}, fmt.Errorf("提交用户消息事务失败: %w", err)
	}
	return result, nil
}

// appendUserMessageTx 在调用方提供的短事务中锁定 Session、分配 seq，并幂等写入用户消息。
// 调用方负责提交或回滚事务。
func appendUserMessageTx(ctx context.Context, tx pgx.Tx, request service.AppendUserMessageRequest) (service.AppendUserMessageResult, error) {
	var messageCount int
	err := tx.QueryRow(ctx, `
		SELECT message_count
		FROM agent_memory_session
		WHERE session_id = $1
		  AND user_id = $2
		  AND status = 'active'
		  AND deleted_at IS NULL
		FOR UPDATE`, request.SessionID, request.UserID).Scan(&messageCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return service.AppendUserMessageResult{}, service.ErrSessionNotFound
	}
	if err != nil {
		return service.AppendUserMessageResult{}, fmt.Errorf("锁定会话失败: %w", err)
	}

	existing, err := scanUserMessage(tx.QueryRow(ctx, `
		SELECT id, user_id, session_id, client_message_id::text, seq,
		       COALESCE(content, ''), COALESCE(trace_id, ''), created_at
		FROM agent_memory_episodic
		WHERE user_id = $1
		  AND session_id = $2
		  AND client_message_id = $3
		  AND role = 'user'`, request.UserID, request.SessionID, request.ClientMessageID))
	if err == nil {
		return service.AppendUserMessageResult{Message: existing, Created: false}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return service.AppendUserMessageResult{}, fmt.Errorf("查询幂等用户消息失败: %w", err)
	}

	nextSeq := messageCount + 1
	message, err := scanUserMessage(tx.QueryRow(ctx, `
		INSERT INTO agent_memory_episodic (
			session_id, user_id, agent_id, seq, role, status,
			content, client_message_id, trace_id
		)
		VALUES ($1, $2, $3, $4, 'user', 'completed', $5, $6, $7)
		RETURNING id, user_id, session_id, client_message_id::text, seq,
		          COALESCE(content, ''), COALESCE(trace_id, ''), created_at`,
		request.SessionID,
		request.UserID,
		service.HealthAgentID,
		nextSeq,
		request.Content,
		request.ClientMessageID,
		request.TraceID,
	))
	if err != nil {
		return service.AppendUserMessageResult{}, fmt.Errorf("写入用户消息失败: %w", err)
	}

	command, err := tx.Exec(ctx, `
		UPDATE agent_memory_session
		SET message_count = $3,
		    last_message_at = $4
		WHERE session_id = $1
		  AND user_id = $2`, request.SessionID, request.UserID, nextSeq, message.CreatedAt)
	if err != nil {
		return service.AppendUserMessageResult{}, fmt.Errorf("更新会话消息计数失败: %w", err)
	}
	if command.RowsAffected() != 1 {
		return service.AppendUserMessageResult{}, fmt.Errorf("更新会话消息计数影响行数异常: %d", command.RowsAffected())
	}

	return service.AppendUserMessageResult{Message: message, Created: true}, nil
}

// AppendAssistantMessage 将完整回复追加到同一 session 的消息流水中。
func (r *PostgresMessageRepository) AppendAssistantMessage(ctx context.Context, request service.AppendAssistantMessageRequest) (service.AssistantMessage, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return service.AssistantMessage{}, fmt.Errorf("开启 assistant 消息事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	message, err := appendAssistantMessageTx(ctx, tx, request)
	if err != nil {
		return service.AssistantMessage{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return service.AssistantMessage{}, fmt.Errorf("提交 assistant 消息事务失败: %w", err)
	}
	return message, nil
}

// appendAssistantMessageTx 在调用方提供的短事务中追加 assistant 消息并更新 Session 计数。
// ParentID 为 0 时保持 NULL；调用方负责提交或回滚事务。
func appendAssistantMessageTx(ctx context.Context, tx pgx.Tx, request service.AppendAssistantMessageRequest) (service.AssistantMessage, error) {
	var messageCount int
	err := tx.QueryRow(ctx, `
		SELECT message_count
		FROM agent_memory_session
		WHERE session_id = $1
		  AND user_id = $2
		  AND status = 'active'
		  AND deleted_at IS NULL
		FOR UPDATE`, request.SessionID, request.UserID).Scan(&messageCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return service.AssistantMessage{}, service.ErrSessionNotFound
	}
	if err != nil {
		return service.AssistantMessage{}, fmt.Errorf("锁定会话失败: %w", err)
	}

	nextSeq := messageCount + 1
	message, err := scanAssistantMessage(tx.QueryRow(ctx, `
		INSERT INTO agent_memory_episodic (
			session_id, user_id, agent_id, seq, parent_id, role, status,
			content, trace_id
		)
		VALUES ($1, $2, $3, $4, NULLIF($5, 0), 'assistant', 'completed', $6, $7)
		RETURNING id, user_id, session_id, seq,
		          COALESCE(content, ''), COALESCE(trace_id, ''), created_at`,
		request.SessionID,
		request.UserID,
		service.HealthAgentID,
		nextSeq,
		request.ParentID,
		request.Content,
		request.TraceID,
	))
	if err != nil {
		return service.AssistantMessage{}, fmt.Errorf("写入 assistant 消息失败: %w", err)
	}

	command, err := tx.Exec(ctx, `
		UPDATE agent_memory_session
		SET message_count = $3,
		    last_message_at = $4
		WHERE session_id = $1
		  AND user_id = $2`, request.SessionID, request.UserID, nextSeq, message.CreatedAt)
	if err != nil {
		return service.AssistantMessage{}, fmt.Errorf("更新会话消息计数失败: %w", err)
	}
	if command.RowsAffected() != 1 {
		return service.AssistantMessage{}, fmt.Errorf("更新会话消息计数影响行数异常: %d", command.RowsAffected())
	}

	return message, nil
}

// LoadRecent 读取最近的 completed 对话消息，再按 seq 正序返回给上层组装上下文。
func (r *PostgresMessageRepository) LoadRecent(ctx context.Context, userID, sessionID string, limit int) ([]service.ConversationMessage, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT seq, role, content
		FROM (
			SELECT seq, role, content
			FROM agent_memory_episodic
			WHERE user_id = $1
			  AND session_id = $2
			  AND status = 'completed'
			  AND deleted_at IS NULL
			  AND role IN ('user', 'assistant')
			  AND content IS NOT NULL
			ORDER BY seq DESC
			LIMIT $3
		) AS recent
		ORDER BY seq ASC`, userID, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("读取最近对话历史失败: %w", err)
	}
	defer rows.Close()

	messages := make([]service.ConversationMessage, 0, limit)
	for rows.Next() {
		var message service.ConversationMessage
		if err := rows.Scan(&message.Seq, &message.Role, &message.Content); err != nil {
			return nil, fmt.Errorf("扫描对话历史失败: %w", err)
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历对话历史失败: %w", err)
	}
	return messages, nil
}

// FindAssistantReplyByID 按 completed turn 保存的结果消息 ID 查询 assistant 回复。
func (r *PostgresMessageRepository) FindAssistantReplyByID(ctx context.Context, userID, sessionID string, messageID int64) (service.AssistantMessage, bool, error) {
	message, err := scanAssistantMessage(r.pool.QueryRow(ctx, `
		SELECT id, user_id, session_id, seq, COALESCE(content, ''), COALESCE(trace_id, ''), created_at
		FROM agent_memory_episodic
		WHERE id = $1
		  AND user_id = $2
		  AND session_id = $3
		  AND role = 'assistant'
		  AND status = 'completed'
		  AND deleted_at IS NULL`, messageID, userID, sessionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return service.AssistantMessage{}, false, nil
	}
	if err != nil {
		return service.AssistantMessage{}, false, fmt.Errorf("查询 assistant 回放消息失败: %w", err)
	}
	return message, true, nil
}

// ListMessages 按 seq 升序返回该会话已完成、未删除的 user/assistant 消息，供"切换会话后
// 加载完整历史"这个场景使用。userID 是调用方已经校验过归属的可信用户，这里的 WHERE 条件
// 再带一层 user_id 过滤，是防止上层校验被绕过的第二道防线，不是唯一防线。
func (r *PostgresMessageRepository) ListMessages(ctx context.Context, userID, sessionID string) ([]service.SessionMessage, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT message.id, message.role, COALESCE(message.content, ''), message.seq, message.created_at
		FROM agent_memory_episodic AS message
		JOIN agent_memory_session AS session
		  ON session.session_id = message.session_id
		 AND session.user_id = message.user_id
		WHERE message.user_id = $1
		  AND message.session_id = $2
		  AND session.deleted_at IS NULL
		  AND message.status = 'completed'
		  AND message.deleted_at IS NULL
		  AND message.role IN ('user', 'assistant')
		ORDER BY message.seq ASC`, userID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("查询会话消息失败: %w", err)
	}
	defer rows.Close()

	messages := make([]service.SessionMessage, 0)
	for rows.Next() {
		var message service.SessionMessage
		if err := rows.Scan(&message.ID, &message.Role, &message.Content, &message.Seq, &message.CreatedAt); err != nil {
			return nil, fmt.Errorf("扫描会话消息失败: %w", err)
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历会话消息失败: %w", err)
	}
	return messages, nil
}

func scanUserMessage(row pgx.Row) (service.UserMessage, error) {
	var message service.UserMessage
	err := row.Scan(
		&message.ID,
		&message.UserID,
		&message.SessionID,
		&message.ClientMessageID,
		&message.Seq,
		&message.Content,
		&message.TraceID,
		&message.CreatedAt,
	)
	return message, err
}

func scanAssistantMessage(row pgx.Row) (service.AssistantMessage, error) {
	var message service.AssistantMessage
	err := row.Scan(
		&message.ID,
		&message.UserID,
		&message.SessionID,
		&message.Seq,
		&message.Content,
		&message.TraceID,
		&message.CreatedAt,
	)
	return message, err
}
