package store_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"healthAgent/internal/config"
	"healthAgent/internal/service"
	"healthAgent/internal/store"
)

func TestPostgresMessageRepositoryAppendsUserMessagesIdempotently(t *testing.T) {
	if os.Getenv("HEALTH_AGENT_INTEGRATION_TEST") != "1" {
		t.Skip("set HEALTH_AGENT_INTEGRATION_TEST=1 to run PostgreSQL integration tests")
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
		TraceID:         "trace-first",
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
	if retried.Created || retried.Message.ID != first.Message.ID {
		t.Fatalf("retry result = %+v, want existing message %d", retried, first.Message.ID)
	}

	conflicting := firstRequest
	conflicting.Content = "changed content"
	if _, err := messageService.AppendUserMessage(ctx, conflicting); !errors.Is(err, service.ErrClientMessageConflict) {
		t.Fatalf("conflicting retry error = %v, want ErrClientMessageConflict", err)
	}

	const concurrentMessages = 8
	seqs := make(chan int, concurrentMessages)
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
				TraceID:         fmt.Sprintf("trace-%d", messageIndex),
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

	actualSeqs := []int{first.Message.Seq}
	for seq := range seqs {
		actualSeqs = append(actualSeqs, seq)
	}
	sort.Ints(actualSeqs)
	for index, seq := range actualSeqs {
		if seq != index+1 {
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
