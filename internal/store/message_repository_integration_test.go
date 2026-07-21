package store_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"healthAgent/internal/config"
	"healthAgent/internal/service"
	"healthAgent/internal/store"
)

func TestPostgresMessageRepositoryAppendsUserMessagesIdempotently(t *testing.T) {
	if os.Getenv("INTERVIEW_AGENT_INTEGRATION_TEST") != "1" {
		t.Skip("set INTERVIEW_AGENT_INTEGRATION_TEST=1 to run PostgreSQL integration tests")
	}

	cfg, err := config.Load("../../config.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	pool, err := store.NewPostgres(cfg.Postgres)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	const (
		userID    = "usr_message_repository_test"
		sessionID = "session_00000000000000000000000000000073"
	)
	ctx := context.Background()
	cleanupMessageRepositoryTest(t, pool, userID)
	defer cleanupMessageRepositoryTest(t, pool, userID)

	if _, err := pool.Exec(ctx, `
		INSERT INTO agent_user (user_id, user_type, status)
		VALUES ($1, 0, 0)`, userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO agent_memory_session (session_id, user_id)
		VALUES ($1, $2)`, sessionID, userID); err != nil {
		t.Fatalf("insert session: %v", err)
	}

	repository := store.NewPostgresMessageRepository(pool)
	messageService := service.NewMessageService(repository)
	firstRequest := service.AppendUserMessageRequest{
		UserID:          userID,
		SessionID:       sessionID,
		ClientMessageID: "00000000-0000-4000-8000-000000000073",
		Content:         "first message",
	}
	first, err := messageService.AppendUserMessage(ctx, firstRequest)
	if err != nil {
		t.Fatalf("append first message: %v", err)
	}
	if !first.Created || first.Message.Seq != 1 {
		t.Fatalf("first result = %+v, want created seq 1", first)
	}

	retried, err := messageService.AppendUserMessage(ctx, firstRequest)
	if err != nil {
		t.Fatalf("retry first message: %v", err)
	}
	if retried.Created || retried.Message.MessageID != first.Message.MessageID {
		t.Fatalf("retry result = %+v, want existing message %s", retried, first.Message.MessageID)
	}

	conflicting := firstRequest
	conflicting.Content = "changed content"
	if _, err := messageService.AppendUserMessage(ctx, conflicting); !errors.Is(err, service.ErrClientMessageConflict) {
		t.Fatalf("conflicting retry error = %v, want ErrClientMessageConflict", err)
	}

	const concurrentMessages = 8
	seqs := make(chan int64, concurrentMessages)
	errorsChannel := make(chan error, concurrentMessages)
	var waitGroup sync.WaitGroup
	for index := 0; index < concurrentMessages; index++ {
		waitGroup.Add(1)
		go func(messageIndex int) {
			defer waitGroup.Done()
			result, appendErr := messageService.AppendUserMessage(ctx, service.AppendUserMessageRequest{
				UserID:          userID,
				SessionID:       sessionID,
				ClientMessageID: fmt.Sprintf("00000000-0000-4000-8000-%012d", messageIndex+100),
				Content:         fmt.Sprintf("message %d", messageIndex),
			})
			if appendErr != nil {
				errorsChannel <- appendErr
				return
			}
			seqs <- result.Message.Seq
		}(index)
	}
	waitGroup.Wait()
	close(seqs)
	close(errorsChannel)
	for appendErr := range errorsChannel {
		t.Errorf("concurrent append: %v", appendErr)
	}
	if t.Failed() {
		return
	}

	actualSeqs := []int64{first.Message.Seq}
	for seq := range seqs {
		actualSeqs = append(actualSeqs, seq)
	}
	slices.Sort(actualSeqs)
	for index, seq := range actualSeqs {
		if seq != int64(index+1) {
			t.Fatalf("seqs = %v, want continuous 1..%d", actualSeqs, concurrentMessages+1)
		}
	}

	var messageCount int
	var storedMessages int
	if err := pool.QueryRow(ctx, `
		SELECT message_count,
		       (SELECT COUNT(*) FROM agent_memory_episodic WHERE user_id = $2 AND session_id = $1)
		FROM agent_memory_session
		WHERE session_id = $1 AND user_id = $2`, sessionID, userID).Scan(&messageCount, &storedMessages); err != nil {
		t.Fatalf("query final counts: %v", err)
	}
	if messageCount != concurrentMessages+1 || storedMessages != concurrentMessages+1 {
		t.Fatalf("message_count=%d stored=%d, want %d", messageCount, storedMessages, concurrentMessages+1)
	}

	_, err = messageService.AppendUserMessage(ctx, service.AppendUserMessageRequest{
		UserID:          "usr_other",
		SessionID:       sessionID,
		ClientMessageID: "00000000-0000-4000-8000-000000000999",
		Content:         "not owned",
	})
	if !errors.Is(err, service.ErrSessionNotFound) {
		t.Fatalf("foreign user error = %v, want ErrSessionNotFound", err)
	}
}

func TestPostgresMessageRepositoryLoadsRecentCompletedHistory(t *testing.T) {
	if os.Getenv("INTERVIEW_AGENT_INTEGRATION_TEST") != "1" {
		t.Skip("set INTERVIEW_AGENT_INTEGRATION_TEST=1 to run PostgreSQL integration tests")
	}

	cfg, err := config.Load("../../config.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	pool, err := store.NewPostgres(cfg.Postgres)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	const (
		userID         = "usr_message_history_test"
		sessionID      = "session_00000000000000000000000000000074"
		deletedSession = "session_00000000000000000000000000000076"
	)
	ctx := context.Background()
	cleanupMessageRepositoryTest(t, pool, userID)
	defer cleanupMessageRepositoryTest(t, pool, userID)

	if _, err := pool.Exec(ctx, `
		INSERT INTO agent_user (user_id, user_type, status)
		VALUES ($1, 0, 0)`, userID); err != nil {
		t.Fatalf("insert history user: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO agent_memory_session (session_id, user_id, deleted_at)
		VALUES ($1, $3, NULL), ($2, $3, now())`, sessionID, deletedSession, userID); err != nil {
		t.Fatalf("insert history session: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO agent_memory_episodic
			(message_id, session_id, user_id, agent_id, seq, role, status, content, deleted_at)
		VALUES
			(gen_random_uuid(), $1, $2, 'interview-agent', 1, 'user',      'completed', 'old user',          NULL),
			(gen_random_uuid(), $1, $2, 'interview-agent', 2, 'assistant', 'completed', 'old assistant',     NULL),
			(gen_random_uuid(), $1, $2, 'interview-agent', 3, 'system',    'completed', 'hidden system',     NULL),
			(gen_random_uuid(), $1, $2, 'interview-agent', 4, 'user',      'failed',    'hidden failed',     NULL),
			(gen_random_uuid(), $1, $2, 'interview-agent', 5, 'assistant', 'completed', 'hidden deleted',    now()),
			(gen_random_uuid(), $1, $2, 'interview-agent', 6, 'user',      'completed', 'latest user',       NULL),
			(gen_random_uuid(), $3, $2, 'interview-agent', 1, 'user',      'completed', 'hidden session',    NULL)`, sessionID, userID, deletedSession); err != nil {
		t.Fatalf("insert history fixtures: %v", err)
	}

	history, err := store.NewPostgresMessageRepository(pool).LoadRecent(ctx, userID, sessionID, 2)
	if err != nil {
		t.Fatalf("LoadRecent() error = %v", err)
	}
	want := []service.ConversationMessage{
		{Seq: 2, Role: "assistant", Content: "old assistant"},
		{Seq: 6, Role: "user", Content: "latest user"},
	}
	if len(history) != len(want) {
		t.Fatalf("LoadRecent() = %+v, want %+v", history, want)
	}
	for index := range want {
		if history[index] != want[index] {
			t.Fatalf("LoadRecent()[%d] = %+v, want %+v", index, history[index], want[index])
		}
	}

	foreignHistory, err := store.NewPostgresMessageRepository(pool).LoadRecent(ctx, "usr_other", sessionID, 10)
	if err != nil {
		t.Fatalf("LoadRecent() foreign error = %v", err)
	}
	if len(foreignHistory) != 0 {
		t.Fatalf("LoadRecent() foreign = %+v, want empty", foreignHistory)
	}

	messages, err := store.NewPostgresMessageRepository(pool).ListMessages(ctx, userID, sessionID)
	if err != nil {
		t.Fatalf("ListMessages() error = %v", err)
	}
	wantMessages := []service.SessionMessage{
		{Role: "user", Content: "old user", Seq: 1},
		{Role: "assistant", Content: "old assistant", Seq: 2},
		{Role: "user", Content: "latest user", Seq: 6},
	}
	if len(messages) != len(wantMessages) {
		t.Fatalf("ListMessages() = %+v, want %+v", messages, wantMessages)
	}
	for index, wantMessage := range wantMessages {
		if messages[index].Role != wantMessage.Role || messages[index].Content != wantMessage.Content || messages[index].Seq != wantMessage.Seq {
			t.Fatalf("ListMessages()[%d] = %+v, want role=%q content=%q seq=%d", index, messages[index], wantMessage.Role, wantMessage.Content, wantMessage.Seq)
		}
	}

	foreignMessages, err := store.NewPostgresMessageRepository(pool).ListMessages(ctx, "usr_other", sessionID)
	if err != nil {
		t.Fatalf("ListMessages() foreign error = %v", err)
	}
	if len(foreignMessages) != 0 {
		t.Fatalf("ListMessages() foreign = %+v, want empty", foreignMessages)
	}

	deletedSessionMessages, err := store.NewPostgresMessageRepository(pool).ListMessages(ctx, userID, deletedSession)
	if err != nil {
		t.Fatalf("ListMessages() deleted session error = %v", err)
	}
	if len(deletedSessionMessages) != 0 {
		t.Fatalf("ListMessages() deleted session = %+v, want empty", deletedSessionMessages)
	}
}

func TestPostgresMessageRepositoryAppendsAssistantInSequence(t *testing.T) {
	if os.Getenv("INTERVIEW_AGENT_INTEGRATION_TEST") != "1" {
		t.Skip("set INTERVIEW_AGENT_INTEGRATION_TEST=1 to run PostgreSQL integration tests")
	}

	cfg, err := config.Load("../../config.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	pool, err := store.NewPostgres(cfg.Postgres)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	const (
		userID    = "usr_assistant_repository_test"
		sessionID = "session_00000000000000000000000000000075"
	)
	ctx := context.Background()
	cleanupMessageRepositoryTest(t, pool, userID)
	defer cleanupMessageRepositoryTest(t, pool, userID)

	if _, err := pool.Exec(ctx, `
		INSERT INTO agent_user (user_id, user_type, status)
		VALUES ($1, 0, 0)`, userID); err != nil {
		t.Fatalf("insert assistant test user: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO agent_memory_session (session_id, user_id)
		VALUES ($1, $2)`, sessionID, userID); err != nil {
		t.Fatalf("insert assistant test session: %v", err)
	}

	messageService := service.NewMessageService(store.NewPostgresMessageRepository(pool))
	userResult, err := messageService.AppendUserMessage(ctx, service.AppendUserMessageRequest{
		UserID:          userID,
		SessionID:       sessionID,
		ClientMessageID: "00000000-0000-4000-8000-000000000075",
		Content:         "user question",
	})
	if err != nil {
		t.Fatalf("append user message: %v", err)
	}
	assistant, err := messageService.AppendAssistantMessage(ctx, service.AppendAssistantMessageRequest{
		UserID:    userID,
		SessionID: sessionID,
		Content:   "assistant answer",
	})
	if err != nil {
		t.Fatalf("append assistant message: %v", err)
	}
	if userResult.Message.Seq != 1 || assistant.Seq != 2 {
		t.Fatalf("user=%+v assistant=%+v, want seq 1/2", userResult.Message, assistant)
	}

	var messageCount int
	var assistantClientMessageID *string
	var assistantParentMessageID *string
	if err := pool.QueryRow(ctx, `
		SELECT session.message_count, episodic.client_message_id::text, episodic.parent_message_id::text
		FROM agent_memory_session AS session
		JOIN agent_memory_episodic AS episodic
		  ON episodic.session_id = session.session_id
		 AND episodic.user_id = session.user_id
		WHERE session.session_id = $1
		  AND session.user_id = $2
		  AND episodic.message_id = $3`, sessionID, userID, assistant.MessageID).Scan(&messageCount, &assistantClientMessageID, &assistantParentMessageID); err != nil {
		t.Fatalf("query assistant persistence: %v", err)
	}
	if messageCount != 2 || assistantClientMessageID != nil || assistantParentMessageID != nil {
		t.Fatalf("message_count=%d client_message_id=%v parent_message_id=%v, want 2, NULL and NULL", messageCount, assistantClientMessageID, assistantParentMessageID)
	}
}

func cleanupMessageRepositoryTest(t *testing.T, pool *pgxpool.Pool, userID string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := pool.Exec(ctx, `DELETE FROM agent_memory_episodic WHERE user_id = $1`, userID); err != nil {
		t.Fatalf("cleanup messages: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM agent_memory_session WHERE user_id = $1`, userID); err != nil {
		t.Fatalf("cleanup sessions: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM agent_user WHERE user_id = $1`, userID); err != nil {
		t.Fatalf("cleanup user: %v", err)
	}
}
