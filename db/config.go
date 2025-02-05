package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Config 数据库配置结构体
type Config struct {
	DBName             string        //数据库别名称、用于区分不同数据库
	Driver             string        // 数据库驱动
	Host               string        // 主机地址
	Username           string        // 用户名
	Password           string        // 密码
	Database           string        // 数据库名称
	Charset            string        // 字符集
	TablePrefix        string        // 表前缀
	LogDir             string        // 日志目录
	LogLevel           string        // 日志级别（支持：debug|info|warn|error）
	ConnMaxLifetime    time.Duration // 连接最大生命周期
	ConnMaxIdleTime    time.Duration // 连接最大空闲时间
	ConnTimeout        time.Duration // 连接超时时间
	ReadTimeout        time.Duration // 读取超时时间
	WriteTimeout       time.Duration // 写入超时时间
	SlowQueryTime      time.Duration // 慢查询阈值
	PoolStatsInterval  time.Duration // 连接池统计频率
	Port               int
	LogBufferSize      int  // 日志缓冲区数量（默认5000）
	MaxOpenConns       int  // 最大打开连接数（默认0）
	MaxIdleConns       int  // 最大空闲连接数（默认0）
	LogRotationMaxAge  int  // 日志保留天数，默认30天
	LogRotationEnabled bool // 是否启用日志轮转
	EnablePoolStats    bool // 是否启用性能指标（默认false）
	Debug              bool // 是否开启调试模式（默认false）
}

// New 创建新的数据库连接
func New(cfg *Config) (*XgDB, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("数据库参数配置有误: %v", err)
	}
	// 设置默认值
	if cfg.DBName == "" {
		cfg.DBName = "master"
	}
	if cfg.Driver == "" {
		cfg.Driver = "MySQL"
	}
	if cfg.Charset == "" {
		cfg.Charset = "utf8mb4"
	}
	if cfg.MaxOpenConns == 0 {
		cfg.MaxOpenConns = 10
	}
	if cfg.MaxIdleConns == 0 {
		cfg.MaxIdleConns = 5
	}
	if cfg.ConnMaxLifetime == 0 {
		cfg.ConnMaxLifetime = time.Hour * 1
	}
	if cfg.ConnMaxIdleTime == 0 {
		cfg.ConnMaxIdleTime = time.Minute * 30
	}
	if cfg.ConnTimeout == 0 {
		cfg.ConnTimeout = time.Second * 1
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = time.Second * 30
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = time.Second * 30
	}
	if cfg.SlowQueryTime == 0 {
		cfg.SlowQueryTime = time.Second * 1
	}
	if cfg.EnablePoolStats {
		if cfg.PoolStatsInterval == 0 || cfg.PoolStatsInterval < time.Second {
			cfg.PoolStatsInterval = 60 * time.Second // 默认60秒
		}
	}
	if cfg.LogDir == "" {
		cfg.LogDir = "./logs"
	}

	// 设置日志保留天数的默认值
	if cfg.LogRotationMaxAge <= 0 {
		cfg.LogRotationMaxAge = 30 // 默认保留30天
	}

	if cfg.LogBufferSize == 0 {
		cfg.LogBufferSize = 5000
	}

	// 构建 DSN
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local&timeout=%s&readTimeout=%s&writeTimeout=%s",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
		cfg.Charset,
		safeTimeout(cfg.ConnTimeout),  // 带最小值的超时
		safeTimeout(cfg.ReadTimeout),  // 带最小值的读超时
		safeTimeout(cfg.WriteTimeout), // 带最小值的写超时
	)

	// 连接数据库
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %v", err)
	}

	// 设置连接池
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("测试数据库连接失败: %v", err)
	}

	logLevelVar := new(slog.LevelVar)
	logLevel, err := parseLogLevel(cfg.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("日志级别设置失败: %v", err)
	}
	logLevelVar.Set(logLevel)

	// 创建异步处理器
	asyncHandler := NewAsyncLogger(NewRotatingFileHandler(
		cfg.LogDir,
		"db",
		time.Duration(cfg.LogRotationMaxAge)*24*time.Hour,
		logLevelVar,
		cfg.LogRotationEnabled,
	).handler, cfg.LogBufferSize)

	// 创建 XgDB 实例
	xdb := &XgDB{
		ctxMu:              new(sync.RWMutex),
		ctx:                ctx,
		cancel:             cancel,
		DBName:             cfg.DBName,
		DB:                 db,
		tablePre:           cfg.TablePrefix,
		asyncMetrics:       NewAsyncMetrics(cfg.DBName),
		structFieldsCache:  NewShardedCache(),
		placeholderCache:   NewShardedCache(),
		logger:             slog.New(asyncHandler),
		LogLevelVar:        logLevelVar,
		startTime:          time.Now(),
		poolStatsStop:      make(chan struct{}),
		PoolStatsInterval:  cfg.PoolStatsInterval,
		poolStatsMutex:     new(sync.Mutex), // 互斥锁保护
		poolStatsTicker:    nil,             // 统计定时器
		slowQueryThreshold: cfg.SlowQueryTime,
		debug:              cfg.Debug,
	}

	// 启动连接池统计信息收集
	if cfg.EnablePoolStats {
		xdb.poolStatsEnabled.Store(true)
		go xdb.collectPoolStats(cfg.PoolStatsInterval)
	}

	// 启动连接探活
	go xdb.startKeepAlive()

	return xdb, nil
}

// Validate 验证配置
func (cfg *Config) Validate() error {
	if cfg == nil {
		return errors.New("配置不能为空")
	}
	if cfg.Host == "" {
		return errors.New("数据库主机不能为空")
	}
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return errors.New("无效端口号")
	}
	if cfg.Username == "" {
		return errors.New("数据库用户名不能为空")
	}
	if cfg.Database == "" {
		return errors.New("数据库名称不能为空")
	}
	if cfg.LogLevel == "" {
		return errors.New("日志等级不能为空")
	}
	if _, err := parseLogLevel(cfg.LogLevel); err != nil {
		return err
	}
	if cfg.LogDir == "" {
		return errors.New("日志目录不能为空")
	}
	return nil
}
