-- 回滚 000008：删表会连带删除其索引和触发器；set_updated_at() 函数由 000001 拥有，不在此删。
DROP TABLE IF EXISTS agent_turn_lease;
