package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var logLevelMap = map[string]slog.Level{
	"debug": slog.LevelDebug,
	"info":  slog.LevelInfo,
	"warn":  slog.LevelWarn,
	"error": slog.LevelError,
}

// AsyncLogger 异步日志处理器
type AsyncLogger struct {
	baseHandler slog.Handler       // 实际处理器
	ch          chan slog.Record   // 缓冲通道
	wg          *sync.WaitGroup    // 使用指针避免复制
	ctx         context.Context    // 上下文
	cancel      context.CancelFunc // 取消函数
	dropped     atomic.Uint64      // 丢弃的日志计数
	total       atomic.Uint64      // 总处理日志数
	errCh       chan error         // 错误通道
	closed      atomic.Bool        // 是否已关闭
}

// RotatingFileHandler 日志文件旋转处理器
type RotatingFileHandler struct {
	handler            slog.Handler // 实际处理器
	dir                string       // 日志目录
	baseFileName       string       // 基础文件名
	currentDate        string       // 当前日期
	currentFile        *os.File     // 当前日志文件
	mu                 sync.Mutex
	maxAge             time.Duration  // 日志文件最大保留时间
	logLevel           *slog.LevelVar // 日志级别
	logRotationEnabled bool           // 日志轮转是否启用
}

// NewAsyncLogger 创建异步日志处理器
func NewAsyncLogger(h slog.Handler, bufferSize int) *AsyncLogger {
	ctx, cancel := context.WithCancel(context.Background())
	al := &AsyncLogger{
		baseHandler: h,
		ch:          make(chan slog.Record, bufferSize),
		wg:          &sync.WaitGroup{}, // 使用指针初始化
		ctx:         ctx,
		cancel:      cancel,
		errCh:       make(chan error, 100), // 增加错误通道
	}

	// 启动处理协程
	al.wg.Add(1)
	go al.process()

	return al
}

// Enabled 实现 slog.Handler 接口
func (al *AsyncLogger) Enabled(ctx context.Context, level slog.Level) bool {
	return al.baseHandler.Enabled(ctx, level)
}

// Handle 实现 slog.Handler 接口
func (al *AsyncLogger) Handle(ctx context.Context, r slog.Record) error {
	// 快速检查是否已关闭
	if al.closed.Load() {
		return errors.New("日志处理器已关闭")
	}
	select {
	case al.ch <- r: // 尝试非阻塞写入
		al.total.Add(1)
		return nil
	case <-al.ctx.Done():
		return al.ctx.Err() // 已关闭
	default:
		al.dropped.Add(1)
		// 通道满时记录警告
		select {
		case al.errCh <- fmt.Errorf("日志通道已满，丢弃日志记录"):
		default:
			// 错误通道也满了，直接忽略
		}
		return nil
	}
}

// WithAttrs 实现 slog.Handler 接口
func (al *AsyncLogger) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &AsyncLogger{
		baseHandler: al.baseHandler.WithAttrs(attrs),
		ch:          al.ch,
		wg:          al.wg,
		ctx:         al.ctx,
		cancel:      al.cancel,
	}
}

// WithGroup 实现 slog.Handler 接口
func (al *AsyncLogger) WithGroup(name string) slog.Handler {
	return &AsyncLogger{
		baseHandler: al.baseHandler.WithGroup(name),
		ch:          al.ch,
		wg:          al.wg,
		ctx:         al.ctx,
		cancel:      al.cancel,
	}
}

func (al *AsyncLogger) Close() error {
	if al.closed.Load() {
		return nil
	}
	if !al.closed.CompareAndSwap(false, true) {
		return errors.New("日志处理器已关闭")
	}
	al.closed.Store(true)
	// 1. 关闭上下文和通道
	al.cancel()
	close(al.ch) // 关闭通道，通知处理协程退出
	// 2. 设置超时等待协程退出
	done := make(chan struct{})
	go func() {
		al.wg.Wait() // 等待处理协程完成
		close(done)
	}()

	// 3. 超时后强制清理剩余日志
	select {
	case <-done:
		err := al.collectErrors()
		// fmt.Println("日志处理器关闭成功，已丢弃剩余日志", err)
		return err
	case <-time.After(5 * time.Second):
		// 清空通道中未处理的日志
		for len(al.ch) > 0 {
			<-al.ch
		}
		fmt.Println("日志处理器关闭超时，已丢弃剩余日志")
	}
	return nil
}

// GetDroppedLogsCount 获取丢弃的日志数量
func (al *AsyncLogger) GetDroppedLogsCount() uint64 {
	return al.dropped.Load()
}

// GetTotalLogsCount 获取总处理日志数量
func (al *AsyncLogger) GetTotalLogsCount() uint64 {
	return al.total.Load()
}

// GetLogMetrics 获取当前日志状态
func (al *AsyncLogger) GetLogMetrics() map[string]uint64 {
	return map[string]uint64{
		"total_logs":    al.total.Load(),
		"dropped_logs":  al.dropped.Load(),
		"channel_depth": uint64(len(al.ch)),
	}
}

func (al *AsyncLogger) collectErrors() error {
	var errs []error
	for {
		select {
		case err := <-al.errCh:
			if err != nil {
				errs = append(errs, err)
			}
			return fmt.Errorf("错误: %v", errs)
		default:
			if len(errs) > 0 {
				return fmt.Errorf("日志处理错误: %v", errs)
			}
			return nil
		}
	}
}

// process 日志处理协程
func (al *AsyncLogger) process() {
	defer al.wg.Done()
	defer close(al.errCh)

	for {
		select {
		case r := <-al.ch:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := al.baseHandler.Handle(ctx, r)
			cancel()
			if err != nil {
				select {
				case al.errCh <- err:
				default:
					// 错误通道已满，忽略
					log.Printf("错误通道已满，丢弃错误: %v", err)
				}
			}

		case <-al.ctx.Done():
			// 设置处理剩余日志的超时
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			for len(al.ch) > 0 {
				select {
				case r := <-al.ch:
					_ = al.baseHandler.Handle(ctx, r)
				case <-ctx.Done():
					return // 超时强制退出
				}
			}
			return
		}
	}
}

func NewRotatingFileHandler(dir, baseFileName string, maxAge time.Duration, logLevel *slog.LevelVar, LogRotationEnabled bool) *RotatingFileHandler {
	r := &RotatingFileHandler{
		dir:                dir,
		baseFileName:       baseFileName,
		maxAge:             maxAge,
		logLevel:           logLevel,
		logRotationEnabled: LogRotationEnabled,
	}
	r.openNewFileIfNeeded()
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handler = slog.NewJSONHandler(r.currentFile, &slog.HandlerOptions{Level: r.logLevel})
	go r.startLogRotationCleanup()
	return r
}

// 实现 io.Writer 接口
func (r *RotatingFileHandler) Write(p []byte) (n int, err error) {
	r.openNewFileIfNeeded()
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.currentFile.Write(p)
}

// 实现 slog.Handler 接口
func (r *RotatingFileHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true // 根据实际需求调整
}

func (r *RotatingFileHandler) Handle(ctx context.Context, record slog.Record) error {
	return r.handler.Handle(ctx, record)
}

func (r *RotatingFileHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return r
}

func (r *RotatingFileHandler) WithGroup(name string) slog.Handler {
	return r
}

func (r *RotatingFileHandler) openNewFileIfNeeded() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 确保日志目录存在
	if err := os.MkdirAll(r.dir, 0755); err != nil {
		return err
	}

	// 创建日志处理器
	if r.logRotationEnabled {
		currentDate := time.Now().Format("2006-01-02")
		if currentDate != r.currentDate {
			// 关闭旧文件
			if r.currentFile != nil {
				_ = r.currentFile.Sync() // 强制刷新
				_ = r.currentFile.Close()
			}

			// 创建新文件
			filename := filepath.Join(r.dir, fmt.Sprintf("%s_%s.log", r.baseFileName, currentDate))
			file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}

			r.currentFile = file
			r.currentDate = currentDate
		}
		return nil
	}
	if r.currentFile != nil {
		return nil
	}
	// 非轮转模式下明确使用
	file, err := os.OpenFile(filepath.Join(r.dir, fmt.Sprintf("%s.log", r.baseFileName)), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("无法打开日志文件: %v", err)
	}
	r.currentFile = file
	r.currentDate = ""
	return nil
}

// startLogRotationCleanup 开始日志轮转清理
func (r *RotatingFileHandler) startLogRotationCleanup() {
	// 如果日志轮转未启用，直接返回
	if !r.logRotationEnabled {
		return
	}
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		r.cleanupOldLogs()
	}
}

// cleanupOldLogs 清理旧日志
func (r *RotatingFileHandler) cleanupOldLogs() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 如果日志轮转未启用，直接返回
	if !r.logRotationEnabled {
		return
	}

	files, err := os.ReadDir(r.dir)
	if err != nil {
		fmt.Printf("读取日志目录失败: %v\n", err)
		return
	}

	cutoffTime := time.Now().Add(-r.maxAge)
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), r.baseFileName) && strings.HasSuffix(file.Name(), ".log") {
			parts := strings.Split(file.Name(), "_")
			if len(parts) < 2 || !strings.HasSuffix(parts[1], ".log") {
				continue // 忽略格式不匹配的文件
			}
			// 检查日期部分是否为有效格式
			datePart := strings.TrimSuffix(parts[1], ".log")
			if _, err := time.Parse("2006-01-02", datePart); err != nil {
				continue
			}
			info, err := file.Info()
			if err != nil {
				continue
			}

			if info.ModTime().Before(cutoffTime) {
				os.Remove(filepath.Join(r.dir, file.Name()))
			}
		}
	}
}

func (r *RotatingFileHandler) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.currentFile != nil {
		return r.currentFile.Close()
	}
	return nil
}
