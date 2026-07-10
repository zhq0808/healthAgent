-- 回滚 000004：删表会连带删除其触发器和索引；set_updated_at() 函数由 000001 拥有。
DROP TABLE IF EXISTS agent_user;