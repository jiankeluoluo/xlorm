package db

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Cache 缓存接口定义
type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, expiration time.Duration) error
	Delete(key string) error
}

// 分片锁缓存
type ShardedCache struct {
	shards [16]struct {
		sync.RWMutex
		m      map[string][]string
		hits   atomic.Uint64
		misses atomic.Uint64
	}
}

func NewShardedCache() *ShardedCache {
	c := &ShardedCache{}
	for i := range c.shards {
		c.shards[i].m = make(map[string][]string)
	}
	return c
}

func (c *ShardedCache) Get(key string) ([]string, bool) {
	idx := c.hash(key)
	shard := &c.shards[idx]

	shard.RLock()
	defer shard.RUnlock()

	v, ok := shard.m[key]
	if ok {
		shard.hits.Add(1)
		return v, true
	}

	shard.misses.Add(1)
	return nil, false
}

func (c *ShardedCache) Set(key string, value []string) {
	idx := c.hash(key)
	shard := &c.shards[idx]

	shard.Lock()
	defer shard.Unlock()

	if shard.m == nil {
		shard.m = make(map[string][]string)
	}
	shard.m[key] = value
}

// 获取缓存统计信息
func (c *ShardedCache) Stats() map[string]uint64 {
	stats := make(map[string]uint64)
	for i := range c.shards {
		stats[fmt.Sprintf("shard_%d_hits", i)] = c.shards[i].hits.Load()
		stats[fmt.Sprintf("shard_%d_misses", i)] = c.shards[i].misses.Load()
	}
	return stats
}

// Clear 清理所有缓存并重置统计信息
func (c *ShardedCache) Clear() {
	for i := range c.shards {
		shard := &c.shards[i]
		shard.Lock()
		shard.m = make(map[string][]string) // 重新创建一个新的空映射
		shard.hits.Store(0)
		shard.misses.Store(0)
		shard.Unlock()
	}
}

// hash 计算键的哈希值，用于确定分片索引
func (c *ShardedCache) hash(key string) uint32 {
	// 使用 FNV-1a 哈希算法
	h := uint32(2166136261)
	for i := 0; i < len(key); i++ {
		h ^= uint32(key[i])
		h *= 16777619
	}
	return h % 16 // 返回 0-15 的索引
}
