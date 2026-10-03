package utilities

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
)

const (
	DEBUG = "DEBUG"
	INFO  = "INFO"
	WARN  = "WARN"
	ERROR = "ERROR"
)

type logEntry struct {
	ReqID      string      `json:"req_id"`
	Level      string      `json:"level"`
	Path       string      `json:"path"`
	Function   string      `json:"function"`
	Payload    interface{} `json:"payload,omitempty"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data,omitempty"`
	File       string      `json:"file,omitempty"`
	Line       int         `json:"line,omitempty"`
	CallerFunc string      `json:"caller_func,omitempty"`
	StackTrace string      `json:"stack_trace,omitempty"`
	Time       string      `json:"time"`
}

func getLevelPriority(level string) int {
	switch level {
	case DEBUG:
		return 1
	case INFO:
		return 2
	case WARN:
		return 3
	case ERROR:
		return 4
	default:
		return 2 // Default to INFO
	}
}

// Log records structured JSON log messages with caller info, dynamic redaction, and stack traces.
// It supports both:
// - Log(level, path, functionName, payload, message, data)
// - Log(reqId, level, path, functionName, payload, message, data)
func Log(arg1 string, arg2 string, arg3 string, arg4 interface{}, arg5 interface{}, arg6 interface{}, optionalArg7 ...interface{}) {
	var reqId, level, path, functionName, message string
	var payload, data interface{}

	if len(optionalArg7) > 0 {
		// 7 arguments: reqId, level, path, functionName, payload, message, data
		reqId = arg1
		level = arg2
		path = arg3
		if s, ok := arg4.(string); ok {
			functionName = s
		} else {
			functionName = fmt.Sprintf("%v", arg4)
		}
		payload = arg5
		if s, ok := arg6.(string); ok {
			message = s
		} else {
			message = fmt.Sprintf("%v", arg6)
		}
		data = optionalArg7[0]
	} else {
		// 6 arguments: level, path, functionName, payload, message, data
		reqId = ""
		level = arg1
		path = arg2
		functionName = arg3
		payload = arg4
		if s, ok := arg5.(string); ok {
			message = s
		} else {
			message = fmt.Sprintf("%v", arg5)
		}
		data = arg6
	}

	// Read LOG_LEVEL from environment directly to avoid circular dependency with configs package
	configLevel := os.Getenv("LOG_LEVEL")
	if configLevel == "" {
		configLevel = INFO
	}

	if getLevelPriority(level) < getLevelPriority(configLevel) {
		return
	}

	// Dynamic caller resolution
	var callerFile string
	var callerLine int
	var callerFunc string
	if pc, filePath, lineNum, ok := runtime.Caller(1); ok {
		callerFile = filepath.Base(filePath)
		callerLine = lineNum
		if fn := runtime.FuncForPC(pc); fn != nil {
			callerFunc = fn.Name()
		}
	}

	// Capture stack trace for errors
	var stackTrace string
	if level == ERROR {
		stackTrace = string(debug.Stack())
	}

	entry := logEntry{
		ReqID:      reqId,
		Level:      level,
		Path:       path,
		Function:   functionName,
		Payload:    prepareValue(payload),
		Message:    message,
		Data:       prepareValue(data),
		File:       callerFile,
		Line:       callerLine,
		CallerFunc: callerFunc,
		StackTrace: stackTrace,
		Time:       time.Now().Format("2006-01-02 15:04:05.000"),
	}

	bz, err := json.Marshal(entry)
	if err != nil {
		// Fallback to plain print if JSON marshaling fails
		fmt.Fprintf(os.Stderr, `{"req_id":%q,"level":"ERROR","message":"failed to marshal log entry","error":%q,"time":%q}`+"\n",
			reqId, err.Error(), time.Now().Format("2006-01-02 15:04:05.000"))
		return
	}

	// Output exactly a single JSON line to stdout
	fmt.Fprintln(os.Stdout, string(bz))
}

func prepareValue(input interface{}) interface{} {
	if input == nil {
		return nil
	}
	// Check if it is a single error
	if err, ok := input.(error); ok {
		return err.Error()
	}

	// Convert to JSON and back to easily traverse and redact
	bz, err := json.Marshal(input)
	if err != nil {
		// If marshaling fails (e.g. channels, unsafe types), return formatted string
		return fmt.Sprintf("%v", input)
	}

	var parsed interface{}
	if err := json.Unmarshal(bz, &parsed); err != nil {
		return fmt.Sprintf("%v", input)
	}

	return redactValue(parsed)
}

func redactValue(val interface{}) interface{} {
	switch m := val.(type) {
	case map[string]interface{}:
		newMap := make(map[string]interface{}, len(m))
		for k, v := range m {
			lowerK := strings.ToLower(k)
			if strings.Contains(lowerK, "password") ||
				strings.Contains(lowerK, "pass") ||
				strings.Contains(lowerK, "token") ||
				strings.Contains(lowerK, "secret") ||
				strings.Contains(lowerK, "privatekey") ||
				strings.Contains(lowerK, "key") ||
				strings.Contains(lowerK, "auth") {
				newMap[k] = "[REDACTED]"
			} else {
				newMap[k] = redactValue(v)
			}
		}
		return newMap
	case []interface{}:
		newSlice := make([]interface{}, len(m))
		for i, v := range m {
			newSlice[i] = redactValue(v)
		}
		return newSlice
	default:
		return val
	}
}
