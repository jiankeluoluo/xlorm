package db

import (
	"sync"
	"sync/atomic"
	"time"
)

// 对象池定义
var tablePool = sync.Pool{
	New: func() interface{} {
		return &Table{
			where: make([]string, 0, 4),
			args:  make([]interface{}, 0, 4),
			joins: make([]string, 0, 2),
		}
	},
}

var builderPool = sync.Pool{
	New: func() interface{} {
		return &Builder{
			fields: make([]string, 0, 8),
			where:  make([]string, 0, 4),
			args:   make([]interface{}, 0, 4),
			joins:  make([]string, 0, 2),
		}
	},
}

// 改用原子值存储指针
var poolStats atomic.Value

// PoolStats 连接池状态结构体
type PoolStats struct {
	WaitDuration       time.Duration // 等待总时长
	MaxIdleClosed      int64         // 因超过最大空闲时间而关闭的连接数
	MaxLifetimeClosed  int64         // 因超过生命周期而关闭的连接数
	WaitCount          int64         // 等待连接的次数
	MaxOpenConnections int           // 最大打开连接数
	OpenConnections    int           // 当前打开的连接数
	InUse              int           // 当前正在使用的连接数
	Idle               int           // 当前空闲的连接数
}
