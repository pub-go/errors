package errors

import (
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

// callers 获取本函数调用者的调用者的堆栈信息
// runtime.Callers <- callers <- errors.New <- user
func callers() *stack {
	const numFrames = 32
	var pcs [numFrames]uintptr
	n := runtime.Callers(3, pcs[:])
	var st stack = pcs[0:n]
	return &st
}

// stack 堆栈信息
type stack []uintptr
type stackTraceProvider = interface{ StackTrace() []uintptr }

var _ stackTraceProvider = (*stack)(nil)

// StackTrace implements stackTraceProvider
func (s *stack) StackTrace() []uintptr {
	f := make([]uintptr, len(*s))
	for i := 0; i < len(f); i++ {
		f[i] = (*s)[i]
	}
	return f
}

var ptrType = reflect.TypeOf(uintptr(0))

// GetStackTrace 获取错误上附加的堆栈
// 兼容 pkg/errors, cockroachdb/errors (通过反射)
func GetStackTrace(err error) (st []uintptr, ok bool) {
	if se, ok := err.(stackTraceProvider); ok {
		f := se.StackTrace() // 本项目的 stack 直接调用 不用反射
		return f, true
	}
	rt := reflect.TypeOf(err)
	rm, has := rt.MethodByName("StackTrace")
	if !has {
		return // 没有这个方法 直接返回
	}
	rv := reflect.ValueOf(err)
	// 在不引入 pkg/errors, cockroachdb/error 依赖的情况下
	// 获取那些错误的堆栈，可以使用反射
	if method := rv.Method(rm.Index); method != (reflect.Value{}) {
		if result := method.Call(nil); len(result) == 1 {
			if result := result[0]; result.Kind() == reflect.Slice {
				count := result.Len()
				st = make([]uintptr, 0, count)
				for i := 0; i < count; i++ {
					rvItem := result.Index(i)
					if ok || rvItem.CanConvert(ptrType) {
						rvItem = rvItem.Convert(ptrType)
						ok = true
					}
					if ok {
						st = append(st, rvItem.Interface().(uintptr))
					}
				}
			}
		}
	}
	return
}

// StackDetail 获取堆栈详情
func StackDetail(st []uintptr) string {
	var sb strings.Builder
	for _, f := range st {
		sb.WriteString("\n")
		pc := f - 1
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			sb.WriteString(fn.Name())
			sb.WriteString("\n\t")
			file, line := fn.FileLine(pc)
			sb.WriteString(file)
			sb.WriteString(":")
			sb.WriteString(strconv.Itoa(line))
		} else {
			sb.WriteString("unknown\n\tunknown:0")
		}
	}
	return sb.String()
}

var _ error = (*withStack)(nil)
var _ ErrorPrinter = (*withStack)(nil)
var _ fmt.Formatter = (*withStack)(nil)

type withStack struct {
	error
	*stack
}

func (w *withStack) Cause() error  { return w.error }
func (w *withStack) Unwrap() error { return w.error }

func (e *withStack) Format(s fmt.State, verb rune) {
	FormatError(e, s, verb)
}

func (e *withStack) PrintError(p Printer) (next error) {
	p.PrintDetail("attached stack trace")
	return e.error
}
