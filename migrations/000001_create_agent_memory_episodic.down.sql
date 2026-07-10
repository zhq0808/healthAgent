-- 回滚 000001：删表会连带删除其触发器；函数单独删。
DROP TABLE IF EXISTS agent_memory_episodic;
DROP FUNCTION IF EXISTS set_updated_at();
