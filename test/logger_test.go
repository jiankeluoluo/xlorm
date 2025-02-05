package test

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"xlorm/db"

	"github.com/stretchr/testify/assert"
)

// getDBConfig 获取测试数据库配置
func getDBConfig(logLevel string) *db.Config {
	return &db.Config{
		Host:            "localhost",
		Port:            3306,
		Username:        "root",
		Password:        "root",
		DBName:          "test_db",
		Database:        "test_db",
		Charset:         "utf8mb4",
		TablePrefix:     "test_",
		LogDir:          "./logs",
		LogLevel:        logLevel,
		Debug:           true,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnTimeout:     5 * time.Second,
		ConnMaxLifetime: 30 * time.Minute,
	}
}

// TestLoggerBasicFunctionality 测试日志基本功能
func TestLoggerBasicFunctionality(t *testing.T) {
	// 创建临时日志目录
	logDir := filepath.Join(os.TempDir(), "xlorm_test_logs")
	err := os.MkdirAll(logDir, 0755)
	if err != nil {
		t.Fatalf("创建日志目录失败: %v", err)
	}
	defer os.RemoveAll(logDir)

	cfg := getDBConfig("debug")
	cfg.LogDir = logDir
	cfg.Debug = true

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("初始化数据库失败: %v", err)
	}
	defer database.Close()

	// 清理并重建测试表
	_, err = database.Exec("DROP TABLE IF EXISTS users")
	if err != nil {
		t.Fatalf("删除测试表失败: %v", err)
	}

	_, err = database.Exec(`
		CREATE TABLE users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			age INT NOT NULL,
			status INT DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("创建测试表失败: %v", err)
	}

	// 测试插入操作
	testUser := map[string]interface{}{
		"name": "log_test_user",
		"age":  25,
	}

	_, err = database.M("users").Insert(testUser)
	if err != nil {
		t.Fatalf("插入数据失败: %v", err)
	}

	// 检查日志文件是否生成
	logFiles, err := os.ReadDir(logDir)
	if err != nil {
		t.Fatalf("读取日志目录失败: %v", err)
	}

	if len(logFiles) == 0 {
		t.Fatal("未生成日志文件")
	}

	// 检查日志文件内容
	for _, file := range logFiles {
		if !strings.HasSuffix(file.Name(), ".log") {
			continue
		}

		content, err := os.ReadFile(filepath.Join(logDir, file.Name()))
		if err != nil {
			t.Fatalf("读取日志文件失败: %v", err)
		}

		if len(content) == 0 {
			t.Errorf("日志文件 %s 为空", file.Name())
		}

		// 检查日志内容是否包含关键信息
		logContent := string(content)
		if !strings.Contains(logContent, "INSERT") {
			t.Errorf("日志文件未记录插入操作")
		}
	}
}

// TestLoggerCustomOutput 测试自定义日志输出
func TestLoggerCustomOutput(t *testing.T) {
	// 创建缓冲区来捕获日志输出
	var logBuffer bytes.Buffer
	_ = log.New(&logBuffer, "XLORM: ", log.Ldate|log.Ltime|log.Lshortfile)

	cfg := getDBConfig("debug")
	cfg.Debug = true

	database, err := db.New(cfg)
	if err != nil {
		t.Fatalf("初始化数据库失败: %v", err)
	}
	defer database.Close()

	// 清理并重建测试表
	_, err = database.Exec("DROP TABLE IF EXISTS users")
	if err != nil {
		t.Fatalf("删除测试表失败: %v", err)
	}

	_, err = database.Exec(`
		CREATE TABLE users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			age INT NOT NULL,
			status INT DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("创建测试表失败: %v", err)
	}

	// 测试插入操作
	testUser := map[string]interface{}{
		"name": "custom_log_user",
		"age":  25,
	}

	_, err = database.M("users").Insert(testUser)
	if err != nil {
		t.Fatalf("插入数据失败: %v", err)
	}

	// 检查自定义日志输出
	logOutput := logBuffer.String()
	if logOutput == "" {
		t.Fatal("未生成自定义日志输出")
	}

	// 验证日志输出的关键信息
	if !strings.Contains(logOutput, "XLORM:") {
		t.Error("自定义日志前缀未正确设置")
	}

	if !strings.Contains(logOutput, "INSERT") {
		t.Error("自定义日志未记录插入操作")
	}
}

// TestLoggerLogLevel 测试日志级别
func TestLoggerLogLevel(t *testing.T) {
	testCases := []struct {
		name string
		// logLevel       db.LogLevel
		shouldLog      bool
		expectedPrefix string
	}{
		{
			name: "调试级别",
			// logLevel:       db.LogLevelDebug,
			shouldLog:      true,
			expectedPrefix: "[DEBUG]",
		},
		{
			name: "信息级别",
			// logLevel:       db.LogLevelInfo,
			shouldLog:      true,
			expectedPrefix: "[INFO]",
		},
		{
			name: "警告级别",
			// logLevel:       db.LogLevelWarn,
			shouldLog:      true,
			expectedPrefix: "[WARN]",
		},
		{
			name: "错误级别",
			// logLevel:       db.LogLevelError,
			shouldLog:      true,
			expectedPrefix: "[ERROR]",
		},
		{
			name: "关闭日志",
			// logLevel:       db.LogLevelOff,
			shouldLog:      false,
			expectedPrefix: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建缓冲区来捕获日志输出
			var logBuffer bytes.Buffer
			_ = log.New(&logBuffer, "", 0)

			cfg := getDBConfig("debug")
			cfg.Debug = true
			// cfg.LogLevel = tc.logLevel

			database, err := db.New(cfg)
			if err != nil {
				t.Fatalf("初始化数据库失败: %v", err)
			}
			defer database.Close()

			// 清理并重建测试表
			_, err = database.Exec("DROP TABLE IF EXISTS users")
			if err != nil {
				t.Fatalf("删除测试表失败: %v", err)
			}

			_, err = database.Exec(`
				CREATE TABLE users (
					id INT AUTO_INCREMENT PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					age INT NOT NULL,
					status INT DEFAULT 1,
					created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
				)
			`)
			if err != nil {
				t.Fatalf("创建测试表失败: %v", err)
			}

			// 测试插入操作
			testUser := map[string]interface{}{
				"name": "log_level_test_user",
				"age":  25,
			}

			_, err = database.M("users").Insert(testUser)
			if err != nil {
				t.Fatalf("插入数据失败: %v", err)
			}

			logOutput := logBuffer.String()

			if tc.shouldLog {
				if logOutput == "" {
					t.Fatalf("%s: 未生成日志输出", tc.name)
				}

				if tc.expectedPrefix != "" && !strings.Contains(logOutput, tc.expectedPrefix) {
					t.Errorf("%s: 日志前缀不正确，期望: %s", tc.name, tc.expectedPrefix)
				}
			} else {
				if logOutput != "" {
					t.Errorf("%s: 不应生成日志输出", tc.name)
				}
			}
		})
	}
}

func TestLogLevel(t *testing.T) {
	cfg := getDBConfig("warn")
	db, _ := db.New(cfg)

	// 初始级别应为warn
	assert.Equal(t, "warn", db.GetLogLevel())

	// 修改为debug
	_ = db.SetLogLevel("debug")
	assert.Equal(t, "debug", db.GetLogLevel())

	// 修改为error
	_ = db.SetLogLevel("error")
	assert.Equal(t, "error", db.GetLogLevel())

	// 测试无效日志级别
	_ = db.SetLogLevel("invalid")
	assert.Equal(t, "error", db.GetLogLevel(), "无效日志级别应保持上一个有效级别")
}

func BenchmarkLogging(b *testing.B) {
	cfg := getDBConfig("error")
	db, _ := db.New(cfg)

	b.Run("disabled", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			db.Logger().Debug("test") // 应被过滤
		}
	})

	b.Run("enabled", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			db.Logger().Error("test") // 应记录
		}
	})
}
