package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

var logFile *os.File
var debugMode = false

func Setup(logDir string, enableDebug bool) {
	debugMode = enableDebug

	if logFile != nil {
		_ = logFile.Close() // close the previous file if it exists
	}

	logPath := filepath.Join(logDir, "e6-cache.log")
	var err error
	logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("[FATAL] Could not open log file: %v", err)
	}

	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
	log.SetFlags(log.LstdFlags)
}

func Info(msg string, v ...any) {
	log.SetPrefix("[INFO] ")
	log.Println(formatMessage(msg, v...))
}

func Warn(msg string, v ...any) {
	log.SetPrefix("[WARN] ")
	log.Println(formatMessage(msg, v...))
}

func Error(msg string, v ...any) {
	log.SetPrefix("[ERROR] ")
	formatted := formatMessage(msg, v...)
	log.Println(formatted)
}

func Fatal(msg string, v ...any) {
	log.SetPrefix("[FATAL] ")
	log.Fatalln(formatMessage(msg, v...))
}

func Debug(msg string, v ...any) {
	if debugMode {
		log.SetPrefix("[DEBUG] ")
		log.Println(formatMessage(msg, v...))
	}
}

func formatMessage(msg string, v ...any) string {
	if len(v) == 0 {
		return msg
	}

	// Try to use Sprintf if format verbs exist
	formatted := fmt.Sprintf(msg, v...)
	if isFormatSafe(formatted) {
		return formatted
	}

	// Fallback: join everything manually
	return fmt.Sprint(append([]any{msg}, v...)...)
}

func isFormatSafe(s string) bool {
	return !containsAny(s, "%!(EXTRA", "%!(", "%!INVALID")
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if contains := (len(sub) > 0 && len(s) > 0 && containsStr(s, sub)); contains {
			return true
		}
	}
	return false
}

func containsStr(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (len(needle) == 0 || (len(haystack) > 0 && len(needle) > 0 && len(haystack) >= len(needle) && (string(haystack[0:len(needle)]) == needle || containsStr(haystack[1:], needle))))
}
