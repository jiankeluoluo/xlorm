package db

import (
	"sync"
	"sync/atomic"
	"time"
)

// Metrics 性能指标结构体
type Metrics struct {
	dbname         string
	queryDurations sync.Map
	affectedRows   atomic.Int64
	totalQueries   atomic.Int64
	slowQueries    atomic.Int64
	errors         atomic.Int64
}

// AsyncMetrics 异步性能指标结构体
type AsyncMetrics struct {
	metrics  chan func(*Metrics)
	stopChan chan struct{}
	wg       sync.WaitGroup
	*Metrics
	droppedMetrics atomic.Uint64 //丢弃的指标数量
}

// NewMetrics 创建新的性能指标实例
func NewMetrics(dbname string) *Metrics {
	return &Metrics{dbname: dbname}
}

// NewAsyncMetrics 创建新的异步性能指标实例
func NewAsyncMetrics(dbname string) *AsyncMetrics {
	am := &AsyncMetrics{
		metrics:  make(chan func(*Metrics), 1000), // 设置合理的缓冲大小
		stopChan: make(chan struct{}),
		Metrics:  NewMetrics(dbname),
	}
	am.start()
	return am
}

// GetDBMetrics 获取性能指标统计
func (m *Metrics) GetDBMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})
	metrics["db_name"] = m.dbname
	// 收集查询时间统计
	queryStats := make(map[string]interface{})
	m.queryDurations.Range(func(key, value interface{}) bool {
		durations := value.([]time.Duration)
		var total time.Duration
		for _, d := range durations {
			total += d
		}
		queryStats[key.(string)] = map[string]interface{}{
			"count":        len(durations),
			"total_time":   total,
			"average_time": total / time.Duration(len(durations)),
		}
		return true
	})

	metrics["query_stats"] = queryStats
	metrics["total_affected_rows"] = m.affectedRows.Load()
	metrics["total_queries"] = m.totalQueries.Load()
	metrics["slow_queries"] = m.slowQueries.Load()
	metrics["total_errors"] = m.errors.Load()

	return metrics
}

// ResetDBMetrics 重置性能指标
func (m *Metrics) ResetDBMetrics() {
	m.queryDurations = sync.Map{}
	m.affectedRows.Store(0)
	m.totalQueries.Store(0)
	m.slowQueries.Store(0)
	m.errors.Store(0)
}

// RecordQueryDuration 记录查询耗时
func (m *Metrics) RecordQueryDuration(queryType string, duration time.Duration) {
	if queryType == "" {
		queryType = "unknown"
	}
	m.totalQueries.Add(1)
	if durations, ok := m.queryDurations.Load(queryType); ok {
		durs := durations.([]time.Duration)
		durs = append(durs, duration)
		m.queryDurations.Store(queryType, durs)
	} else {
		m.queryDurations.Store(queryType, []time.Duration{duration})
	}
}

// RecordAffectedRows 记录影响的行数
func (m *Metrics) RecordAffectedRows(rows int64) {
	m.affectedRows.Add(rows)
}

// RecordError 记录错误
func (m *Metrics) RecordError() {
	m.errors.Add(1)
}

// RecordSlowQuery 记录慢查询
func (m *Metrics) RecordSlowQuery() {
	m.slowQueries.Add(1)
}

func (am *AsyncMetrics) Stop() {
	close(am.stopChan)
	am.wg.Wait()
}

func (am *AsyncMetrics) RecordQueryDuration(queryType string, duration time.Duration) {
	select {
	case am.metrics <- func(m *Metrics) {
		m.RecordQueryDuration(queryType, duration)
	}:
	default:
		am.droppedMetrics.Add(1)
		am.logDroppedMetric(queryType)
	}
}

func (am *AsyncMetrics) RecordError() {
	select {
	case am.metrics <- func(m *Metrics) {
		m.RecordError()
	}:
	default:
	}
}

func (am *AsyncMetrics) RecordSlowQuery() {
	select {
	case am.metrics <- func(m *Metrics) {
		m.RecordSlowQuery()
	}:
	default:
	}
}

// 提供一个方法获取丢弃的指标数量
func (am *AsyncMetrics) GetDroppedMetricsCount() uint64 {
	return am.droppedMetrics.Load()
}

func (am *AsyncMetrics) start() {
	am.wg.Add(1)
	go func() {
		defer am.wg.Done()
		for {
			select {
			case fn := <-am.metrics:
				fn(am.Metrics)
			case <-am.stopChan:
				return
			}
		}
	}()
}

func (am *AsyncMetrics) logDroppedMetric(metricType string) {
	// 记录日志或触发告警
}
