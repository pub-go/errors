package errors

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"

	"code.gopub.tech/errors/fmtfwd"
	"code.gopub.tech/errors/pretty"
)

// F is alias of Formattable
// 将一个错误包装为 fmt.Formatter
func F(err error) FormattableError {
	return Formattable(err)
}

// Formattable 将一个错误包装为 fmt.Formatter
// 标准库中的 wrapErrors, joinerrors 并没有实现 fmt.Formatter 接口
// 使用这个帮助函数可以让那些错误类型也支持智能打印功能（自动解析 cause 等）
func Formattable(err error) FormattableError {
	if err == nil {
		return nil
	}
	return &errorFormatter{err}
}

// Detail 先将 err 包装为 Formattable 再使用 %+v 格式化为字符串
func Detail(err error) string {
	return fmt.Sprintf("%+v", Formattable(err))
}

// FormatError 能够识别格式化动词智能打印。
// 当一个错误类型在实现 fmt.Formatter 接口时，可以直接转发给本函数。
//
// 如果传入的错误类型实现了 errors.Formatter 接口，
// FormatError 会使用‘按 s, verb 配置好的 Printer’
// 调用它的 FormatError 方法。
//
// 否则，如果错误类型是一个包装类型（实现了 Cause/Unwrap）
// FormatError 会输出包装错误的附加信息，并递归地处理其内部错误。
//
// 其他情况，直接打印 Error() 文本。
func FormatError(err error, s fmt.State, verb rune) {
	p := state{State: s}
	switch {
	case verb == 'v' && s.Flag('+') && !s.Flag('#'):
		// 用 %+v 格式将每个错误输出到 p.buf
		// 使用递归解析得到错误树
		p.entry = p.buildTree(err, true)
		// 将错误树输出
		p.formatTree(err)
		// 得到了输出字符串，再看是否有宽度精度这些要求
		p.finishDisplay(verb)

	case verb == 'v' && s.Flag('#'):
		// %#v 语意为`输出 Go 语言表示`
		if es, ok := err.(fmt.GoStringer); ok {
			io.WriteString(&p.finalBuf, es.GoString())
		} else {
			fmt.Fprintf(&p.finalBuf, "%# v", pretty.Formatter(err))
		}
		p.finishDisplay(verb)

	case verb == 's' ||
		(verb == 'v' && !s.Flag('#')) ||
		(verb == 'x' || verb == 'X' || verb == 'q'):
		// 不需要详情
		p.entry = p.buildTree(err, false)
		// 输出一行即可
		p.printErrorString()
		// 再处理 %q %x 这种要求
		p.finishDisplay(verb)

	default:
		// 不支持的格式化动词
		p.finalBuf.WriteString("%!")
		p.finalBuf.WriteRune(verb)
		p.finalBuf.WriteByte('(')
		switch {
		case err != nil:
			p.finalBuf.WriteString(reflect.TypeOf(err).String())
		default:
			p.finalBuf.WriteString("<nil>")
		}
		p.finalBuf.WriteByte(')')
		io.Copy(s, &p.finalBuf)
	}
}

// buildTree 通过 `Unwrap()error` / `Unwrap()[]error` 递归构造错误树
func (s *state) buildTree(err error, withDetail bool) *formatEntry {
	// 每一轮递归 初始化
	s.simpleBuf = bytes.Buffer{}
	s.detailBuf = bytes.Buffer{}

	// 包装单个错误时: `prefix: cause`,
	// 包装错误的 simple 缓冲区是 `prefix`，
	// 不能省略 cause 的 simple 输出
	// 包装多个错误时: `cause1\ncause2`,
	// 包装错误的 simple 缓冲区是 `cause1\ncause2`，
	// 应该省略 cause 的 simple 输出
	var ignoreCause bool
	switch v := err.(type) {
	case ErrorPrinter:
		if e := v.PrintError((*printer)(s)); e == nil {
			// 返回的 next error 为 nil 代表需要忽略 cause
			ignoreCause = true
		}
	default:
		ignoreCause = s.formatDirect(err)
	}

	entry := s.buildEntry(err)
	entry.ignoreCause = ignoreCause
	if st, ok := GetStackTrace(err); ok {
		entry.stackTrace = st
	}

	if cause := UnwrapOnce(err); cause != nil {
		child := s.buildTree(cause, withDetail)
		child.parent = entry
		entry.wraps = append(entry.wraps, child)
	}

	var (
		causes = UnwrapMulti(err)
		count  = len(causes)
		wraps  []*formatEntry
	)
	for i := count - 1; i >= 0; i-- { // 倒序进递归方法里
		// 为了输出时 上面的堆栈可以省略 下面的堆栈更完整
		child := s.buildTree(causes[i], withDetail)
		child.parent = entry
		wraps = append(wraps, child)
	}
	for i := count - 1; i >= 0; i-- { // 再倒序把顺序正过来
		// 输出时直接按顺序从上往下
		entry.wraps = append(entry.wraps, wraps[i])
	}

	if len(entry.stackTrace) > 0 { // 重复堆栈优化输出
		last := entry.stackTrace
		if nst, ok := ElideSharedStackTraceSuffix(s.lastStack, entry.stackTrace); ok {
			entry.stackTrace = nst
			entry.elidedStackTrace = true
		}
		s.lastStack = last
	}
	return entry
}

// formatDirect 直接比较 Error() 字符串看是否有相同后缀
func (s *state) formatDirect(err error) (ignoreCause bool) {
	var pref string
	if cause := UnwrapOnce(err); cause != nil {
		var isPrefix bool
		if pref, isPrefix = extractPrefix(err, cause); !isPrefix {
			ignoreCause = true
		}
	} else {
		pref = err.Error()
	}
	if len(pref) > 0 {
		s.Write([]byte(pref))
	}
	if causes := UnwrapMulti(err); len(causes) > 0 {
		// 如果包装了多个错误，则总是认为需要省略子错误
		// 直接使用包装错误即可
		ignoreCause = true
	}
	return
}

// extractPrefix 如果 err 的错误信息，以 cause 的错误信息为后缀
// 就将 err 中独有的前缀提取出来。
//
// 示例
//
//	err=`prefix: cause`
//	cause=`cause`
//	return=`prefix`, true
func extractPrefix(err, cause error) (msg string, isPrefix bool) {
	causeSuffix := cause.Error()
	errMsg := err.Error()
	if strings.HasSuffix(errMsg, causeSuffix) {
		prefix := errMsg[:len(errMsg)-len(causeSuffix)]
		if len(prefix) == 0 {
			return "", true
		}
		if strings.HasSuffix(prefix, ": ") {
			return prefix[:len(prefix)-2], true
		}
	}
	return errMsg, false
}

// buildEntry 收集简单消息、详细消息
func (s *state) buildEntry(err error) *formatEntry {
	entry := &formatEntry{err: err}
	entry.simple = s.simpleBuf.Bytes()
	entry.detail = s.detailBuf.Bytes()
	return entry
}

// formatTree 遍历错误树，格式化为树形
func (s *state) formatTree(err error) {
	s.printErrorString()
	var errType = map[int]string{}
	s.print(0, s.entry, " │ ", errType)
	s.finalBuf.WriteString("\nError types:")
	count := len(errType)
	for i := 1; i <= count; i++ {
		fmt.Fprintf(&s.finalBuf, " (%d) %s", i, errType[i])
	}
}

// printErrorString 输出简单模型的错误信息
// 这里不直接使用 `err.Error()` 是因为，包装错误的 `Error()`
// 通常会实现为 `return fmt.Sprint(err)` 从而调用到
// `Format` 中的 `FormatError` 会触发递归。
func (s *state) printErrorString() {
	entry := s.entry
	for entry != nil {
		if s.finalBuf.Len() > 0 && len(entry.simple) > 0 {
			s.finalBuf.WriteString(": ")
		}
		if len(entry.simple) > 0 {
			s.finalBuf.Write(entry.simple)
		}
		if entry.ignoreCause {
			break
		}
		if len(entry.wraps) == 1 {
			entry = entry.wraps[0]
		} else {
			break
		}
	}
}

// print 格式化输出每个错误节点
func (s *state) print(depth int, entry *formatEntry, prefix string, errType map[int]string) {
	index := len(errType) + 1 // 计数
	errType[index] = reflect.TypeOf(entry.err).String()

	if index == 1 { // 第一个特殊处理 不需要竖线开头
		fmt.Fprintf(&s.finalBuf, "\n(1)")
	} else {
		// entry.parent != nil must be true
		if len(entry.parent.wraps) > 1 { // 父错误包装了多个错误
			// 先往下看 包装多个错误时 递归调用本方法时 增加了缩进

			count := len(entry.parent.wraps)
			if entry.parent.wraps[count-1] == entry { // 现在是多个 cause 中的最后一个
				// `Next:`
				// ` │  │ Wraps:`
				// 改为
				// `Next:`
				// ` └─ Wraps:`
				fmt.Fprintf(&s.finalBuf, "\n%s Wraps: (%d)",
					prefix[:len(prefix)-len(" │  │ ")]+" └─", index)
				// 如果是最后一个节点
				// ` └─ Wraps: xxx`
				// ` │  │ xxx`
				// 前一条竖线就没必要再输出了
				// ` └─ Wraps: xx`
				// `    │ xxx`
				prefix = prefix[:len(prefix)-len(" │  │ ")] + "    │ "
			} else {
				// `Next:`
				// ` │  │ Wraps:`
				// 改为
				// `Next:`
				// ` ├─ Wraps:`
				fmt.Fprintf(&s.finalBuf, "\n%s Wraps: (%d)",
					prefix[:len(prefix)-len(" │  │ ")]+" ├─", index)
			}
		} else { // 父错误仅有一个 cause
			// ` | `
			// ` | Next:`
			// 改为
			// ` | `
			// `Next:` 减少一点缩进
			fmt.Fprintf(&s.finalBuf, "\n%sNext: (%d)", prefix[:len(prefix)-len(" │ ")], index)
		}
	}

	s.printOne(prefix, entry)

	childCount := len(entry.wraps)
	switch childCount {
	case 0:
		return
	case 1:
		entry = entry.wraps[0] // 包装单个错误 无需新增缩进
		s.print(depth, entry, prefix, errType)
	default:
		for _, entry := range entry.wraps {
			// 包装多个错误 添加缩进
			s.print(depth+1, entry, prefix+" │ ", errType)
		}
	}
}

// printOne 输出错误详细内容
func (s *state) printOne(prefix string, entry *formatEntry) {
	var sb strings.Builder // 先往缓冲区输出

	if len(entry.simple) > 0 {
		if entry.simple[0] != '\n' {
			// 在 `Wraps: (N)` 之后加一个空格
			sb.WriteByte(' ')
		}
		sb.Write(entry.simple)
	}
	if len(entry.detail) > 0 {
		if len(entry.simple) == 0 { // 空格还没加
			if entry.detail[0] != '\n' {
				// 在 `Wraps: (N)` 之后加一个空格
				sb.WriteByte(' ')
			}
		}
		sb.Write(entry.detail)
	}
	if entry.stackTrace != nil {
		if sb.String() == "" {
			sb.WriteString(" attached stack trace")
		}
		sb.WriteString("\n-- stack trace:")
		sb.WriteString(StackDetail(entry.stackTrace))
		if entry.elidedStackTrace {
			sb.WriteString("\n[...repeated from below...]")
		}
	}

	tmp := sb.String()
	if tmp == "" {
		s.finalBuf.WriteString(" " + reflect.TypeOf(entry.err).String())
		return
	}

	// 替换换行符号后再实际输出
	str := replacePrefix(tmp, prefix, len(entry.wraps) == 0)
	s.finalBuf.WriteString(str)
}

// replacePrefix 将内容中的换行符号替换为竖线前缀
func replacePrefix(origin string, linePrefix string, isLeaf bool) string {
	if isLeaf {
		i := strings.LastIndex(origin, "\n")
		if i >= 0 {
			// │ │  │
			// │ │ Next: (8) goErr
			// │ │  │ with
			// │ │  │ new line
			// replace to
			// │ │  │
			// │ │ Next: (8) goErr
			// │ │  │  with      加一个空格
			// │ │  └─ new line¸ 加一个拐角
			linePrefix = linePrefix[:len(linePrefix)-len("│ ")]
			lastPrefix := linePrefix + "└─ "
			linePrefix = linePrefix + "│  "
			origin = strings.ReplaceAll(
				origin[:i], "\n", "\n"+linePrefix) + // 中间的换行
				"\n" + lastPrefix + origin[i+1:] // 最后一个换行
		}
		return origin
	} else {
		return strings.ReplaceAll(origin, "\n", "\n"+linePrefix)
	}
}

// finishDisplay 将 finalBuf 输出到 fmt.State
// 如果有 %q, %x, %X, 宽度, 精度等要求 在这里实现
func (p *state) finishDisplay(verb rune) {
	width, okW := p.Width()
	_, okP := p.Precision()

	direct := verb == 'v' || verb == 's'
	if !direct || (okW && width > 0) || okP {
		_, format := fmtfwd.MakeFormat(p, verb)
		fmt.Fprintf(p.State, format, p.finalBuf.String())
	} else {
		io.Copy(p.State, &p.finalBuf)
	}
}

type state struct {
	// state 组合了 fmt.State 但重写了 Write 方法
	fmt.State
	// 这个缓冲区是最终输出，会在最后时刻写入 fmt.State
	finalBuf bytes.Buffer
	// 错误树
	entry *formatEntry
	// 记录最近一次的堆栈
	lastStack []uintptr

	// 下面的字段会在每轮递归时初始化

	// 错误信息默认输出到这个缓冲区
	simpleBuf bytes.Buffer
	// 错误详情输出到这个缓冲区
	detailBuf bytes.Buffer
}

type formatEntry struct {
	err error
	// 简单模式输出时的内容
	simple []byte
	// 错误详情内容
	detail []byte
	// 在拼接整体简单消息时 输出完自己的 simple 后就结束，忽略 cause 的
	ignoreCause bool
	// 堆栈
	stackTrace []uintptr
	// 堆栈是否和其他 entry 有重复
	elidedStackTrace bool
	// 树形
	parent *formatEntry
	wraps  []*formatEntry
}

// String is used for debugging only.
func (e formatEntry) String() string {
	return fmt.Sprintf("entry{%T, %v, %q, %q}", e.err, e.elidedStackTrace, e.simple, e.detail)
}

// Write implements fmt.State.
// 重写 Write 方法：错误链中的错误执行 Format 方法时，
// 将劫持其输出到 buf 缓冲区中（而不要立即输出到 fmt.State）
func (s *state) Write(b []byte) (n int, err error) {
	if len(b) == 0 {
		return 0, nil
	}
	return s.simpleBuf.Write(b)
}

type printer state

// Print implements Printer.
func (s *printer) Print(args ...any) {
	s.enhanceArgs(args)
	fmt.Fprint(&s.simpleBuf, args...)
}

// Printf implements Printer.
func (s *printer) Printf(format string, args ...any) {
	s.enhanceArgs(args)
	fmt.Fprintf(&s.simpleBuf, format, args...)
}

// PrintDetail implements Printer.
func (s *printer) PrintDetail(args ...any) {
	s.enhanceArgs(args)
	fmt.Fprint(&s.detailBuf, args...)
}

// PrintDetailf implements Printer.
func (s *printer) PrintDetailf(format string, args ...any) {
	s.enhanceArgs(args)
	fmt.Fprintf(&s.detailBuf, format, args...)
}

func (s *printer) enhanceArgs(args []interface{}) {
	for i := range args {
		if err, ok := args[i].(error); ok {
			args[i] = &errorFormatter{err}
		}
	}
}

// ElideSharedStackTraceSuffix 省略共同的堆栈
func ElideSharedStackTraceSuffix(prevStack, newStack []uintptr) ([]uintptr, bool) {
	if len(prevStack) == 0 {
		return newStack, false
	}
	if len(newStack) == 0 {
		return newStack, false
	}

	var i, j int // Skip over the common suffix.
	for i, j = len(newStack)-1, len(prevStack)-1; i > 0 && j > 0; i, j = i-1, j-1 {
		if newStack[i] != prevStack[j] {
			break
		}
	}
	if i == 0 {
		i = 1 // Keep at least one entry.
	}
	return newStack[:i], i < len(newStack)-1
}
