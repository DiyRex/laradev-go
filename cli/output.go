package cli

import "fmt"

const (
	rst    = "\033[0m"
	bold   = "\033[1m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	blue   = "\033[34m"
	cyan   = "\033[36m"
	white  = "\033[97m"
	gray   = "\033[90m"
)

func Banner() {
	fmt.Println()
	fmt.Printf("  %s%sLaraDev%s %s| Environment Manager%s\n", blue, bold, rst, gray, rst)
}

func Info(msg string)    { fmt.Printf("  %s*%s %s\n", blue, rst, msg) }
func Success(msg string) { fmt.Printf("  %sOK%s %s\n", green, rst, msg) }
func Warn(msg string)    { fmt.Printf("  %s!!%s %s\n", yellow, rst, msg) }
func Error(msg string)   { fmt.Printf("  %sERR%s %s\n", red, rst, msg) }
func Step(msg string)    { fmt.Printf("  %s->%s %s\n", cyan, rst, msg) }
func Dimmed(msg string)  { fmt.Printf("  %s%s%s\n", gray, msg, rst) }
