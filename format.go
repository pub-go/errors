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

var (
	detailSep = []byte("\n  | ")
	ptrType   = reflect.TypeOf(uintptr(0))
)

// Formattable 将一个错误包装为 fmt.Formatter
// 标准库中的 wrapErrors, joinerrors 并没有实现 fmt.Formatter 接口
// 使用这个帮助函数可以让那些错误类型也支持智能打印功能（自动解析 cause 等）
func Formattable(err error) FormattableError {
	if err == nil {
		return nil
	}
	return &errorFormatter{err}
}

// FormatError 能够识别格式化动词智能打印。
// 当一个错误类型在实现 fmt.Formatter 接口时，可以直接转发给本函数。
//
// 如果传入的错误类型实现了 errors.Formatter 接口，
// FormatError 会使用‘按 s, verb 配置好的 Printer’
// 调用它的 FormatError 方法。
//
// 否则，如果错误类型是一个包装类型（实现了 Cause/Unwrap）
// FormatError 会输出包装错误的 prefix 信息，并递归地处理其内部错误。
//
// 其他情况，打印 Error() 文本。
func FormatError(err error, s fmt.State, verb rune) {
	p := state{State: s}
	switch {
	case verb == 'v' && s.Flag('+') && !s.Flag('#'):
		// 用 %+v 格式将每个错误输出到 p.buf
		// 通过递归，得到从最内层错误开始的错误链
		p.formatRecursive(
			err,
			true,  // isOutermost
			true,  // withDetail
			false, // withDepth
			0,     // depth
		)
		// 将错误链中每一项错误输出
		p.formatEntries(err)
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
		p.formatRecursive(
			err,
			true,  // isOutermost
			false, // withDetail
			false, // withDepth
			0,     // depth
		)
		// 输出一行即可
		p.formatSingleLineOutput()
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

func (s *state) formatEntries(err error) {
	// 先输出
	//
	// <完整错误链信息>
	// (1) <错误详情>
	s.formatSingleLineOutput()
	s.finalBuf.WriteString("\n(1)")

	s.printEntry(s.entries[len(s.entries)-1])

	// 然后输出包装错误信息
	// Wraps: (N) <错误详情>
	for i, j := len(s.entries)-2, 2; i >= 0; i, j = i-1, j+1 {
		s.finalBuf.WriteByte('\n')
		// 缩进
		for m := 0; m < s.entries[i].depth-1; m += 1 {
			if m == s.entries[i].depth-2 {
				s.finalBuf.WriteString("└─ ")
			} else {
				s.finalBuf.WriteByte(' ')
				s.finalBuf.WriteByte(' ')
			}
		}
		fmt.Fprintf(&s.finalBuf, "Wraps: (%d)", j)
		entry := s.entries[i]
		s.printEntry(entry)
	}

	// 最后输出错误类型
	s.finalBuf.WriteString("\nError types:")
	for i, j := len(s.entries)-1, 1; i >= 0; i, j = i-1, j+1 {
		fmt.Fprintf(&s.finalBuf, " (%d) %T", j, s.entries[i].err)
	}
}

func (s *state) printEntry(entry formatEntry) {
	if len(entry.head) > 0 {
		if entry.head[0] != '\n' {
			s.finalBuf.WriteByte(' ')
		}
		if len(entry.head) > 0 {
			s.finalBuf.Write([]byte(entry.head))
		}
	}

	if len(entry.details) > 0 {
		if len(entry.head) == 0 {
			if entry.details[0] != '\n' {
				s.finalBuf.WriteByte(' ')
			}
		}
		s.finalBuf.Write(entry.details)
	}

	if entry.stackTrace != nil {
		s.finalBuf.WriteString("\n  -- stack trace:")
		s.finalBuf.WriteString(strings.ReplaceAll(
			StackDetail(entry.stackTrace),
			"\n", string(detailSep)))
		if entry.elidedStackTrace {
			fmt.Fprintf(&s.finalBuf, "%s[...repeated from below...]", detailSep)
		}
	}
}

func (s *state) formatSingleLineOutput() {
	// 第一个是 root cause 最后一个是最外层的 wrap error
	for i := len(s.entries) - 1; i >= 0; i-- {
		entry := &s.entries[i]
		if entry.elideShort {
			continue
		}
		if s.finalBuf.Len() > 0 && len(entry.head) > 0 {
			s.finalBuf.WriteString(": ")
		}
		if len(entry.head) > 0 {
			s.finalBuf.Write([]byte(entry.head))
		}
	}
}

// formatRecursive 对错误递归 Unwrap 处理 得到错误链。
// `withDepth` 和 `depth` 用于标记 multi-cause 错误的子树，
// 以便在输出时能够添加适当的缩进；
// 遇到多错误的实例时, withDepth 就会设置为 true.
func (s *state) formatRecursive(
	err error,
	isOutermost,
	withDetail bool,
	withDepth bool,
	depth int,
) int {
	cause := UnwrapOnce(err)
	numChildren := 0
	if cause != nil { // 立即开始递归
		// isOutermost 传 false 不是最外层了
		numChildren += s.formatRecursive(cause, false, withDetail, withDepth, depth+1)
	}
	causes := UnwrapMulti(err)
	for _, c := range causes {
		numChildren += s.formatRecursive(c, false, withDetail, true, depth+1)
	}

	// 每一轮递归 初始化
	s.wantDetail = withDetail
	s.needNewline = 0
	s.notEmpty = false
	s.hasDetail = false
	s.headBuf = nil

	seenTrace := false

	switch v := err.(type) {
	case Formatter: // 本项目的错误都实现了这个接口
		// 也实现了 fmt.Formatter 接口，但其实现就是调用本方法
		// 所以需要先处理 Formatter 以免发生无限递归
		if desiredShortening := v.FormatError((*printer)(s)); desiredShortening == nil {
			s.elideShortChildren(numChildren)
		}

	case fmt.Formatter:
		if !isOutermost && cause == nil {
			// 触发错误的格式化输出 会调用到 state 重写的 Write 方法
			v.Format(s, 'v') // 在 Write 中会改变 state 的一些状态
			if st, ok := GetStackTrace(err); ok {
				seenTrace = true
				s.lastStack = st
			}
		} else {
			if elideCauseMsg := s.formatSimple(err, cause); elideCauseMsg {
				s.elideShortChildren(numChildren)
			}
		}
	default:
		elideChildren := s.formatSimple(err, cause)
		if len(causes) > 0 {
			// 如果是 multi-cause 错误，总是设为 true
			elideChildren = true
		}
		if elideChildren {
			s.elideShortChildren(numChildren)
		}
	}

	entry := s.collectEntry(err, withDepth, depth)

	if !seenTrace {
		if st, ok := GetStackTrace(err); ok {
			entry.stackTrace, entry.elidedStackTrace = ElideSharedStackTraceSuffix(s.lastStack, st)
			s.lastStack = entry.stackTrace
		}
	}

	s.entries = append(s.entries, entry)
	s.buf = bytes.Buffer{}

	return numChildren + 1
}

func (s *state) elideShortChildren(newEntries int) {
	for i := 0; i < newEntries; i++ {
		s.entries[len(s.entries)-1-i].elideShort = true
	}
}

func (s *state) collectEntry(err error, withDepth bool, depth int) formatEntry {
	entry := formatEntry{err: err}

	if s.wantDetail {
		// The buffer has been populated as a result of formatting with
		// %+v. In that case, if the printer has separated detail
		// from non-detail, we can use the split.
		if s.hasDetail {
			entry.head = s.headBuf
			entry.details = s.buf.Bytes()
		} else {
			entry.head = s.buf.Bytes()
		}
	} else {
		entry.head = s.headBuf
		if len(entry.head) > 0 && entry.head[len(entry.head)-1] != '\n' &&
			s.buf.Len() > 0 && s.buf.Bytes()[0] != '\n' {
			entry.head = append(entry.head, '\n')
		}
		entry.head = append(entry.head, s.buf.Bytes()...)
	}
	if withDepth {
		entry.depth = depth
	}
	return entry
}

func (s *state) formatSimple(err, cause error) bool {
	var pref string
	elideCauses := false
	if cause != nil {
		var messageType MessageType
		pref, messageType = extractPrefix(err, cause)
		if messageType == FullMessage {
			elideCauses = true
		}
	} else {
		pref = err.Error()
	}
	if len(pref) > 0 {
		s.Write([]byte(pref))
	}
	return elideCauses
}

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
	// 错误链
	entries []formatEntry
	// 递归收集错误链详情时使用的缓冲区
	buf bytes.Buffer
	// 当一个错误在 FormatError 中调用了 p.Detail() 时
	// 当前 buf 的内容就会拷贝到 headBuf 中，
	// 同时重新初始化一个 buf
	headBuf []byte

	// 记录最近一次的堆栈
	lastStack StackTrace

	// 记录是否有详情，当调用了 p.Detail() 后置为 true
	// 用于 formatEntry 时决定如何使用 buf/headBuf
	hasDetail bool
	// 当使用 %+v 输出时，表示希望输出详情；
	// 其他情况都是 false, 此时调用 p.Detail() 将得到 false.
	wantDetail bool

	// 记录错误输出时 是否已经输出了字符
	// 用于记录换行时是否需要添加竖线/缩进
	notEmpty bool
	// 错误输出时 遇到换行时会插入竖线分隔符
	// 几个换行就需要插入多个分隔符
	needNewline int
}

type formatEntry struct {
	err error
	// 简单模式输出时的内容
	head []byte
	// 错误详情内容
	details []byte
	// 是否需要在输出时省略 head 内容
	elideShort bool
	// 堆栈
	stackTrace StackTrace
	// 堆栈是否和其他 entry 有重复
	elidedStackTrace bool
	// 如果 wrap 了多个错误会用到
	depth int
}

// Write implements fmt.State.
// 重写 Write 方法：错误链中的错误执行 Format 方法时，
// 将劫持其输出到 buf 缓冲区中（而不要立即输出到 fmt.State）
func (s *state) Write(b []byte) (n int, err error) {
	if len(b) == 0 {
		return 0, nil
	}
	k := 0

	sep := detailSep // "\n  | "
	if !s.wantDetail {
		sep = []byte("\n")
	}

	for i, c := range b {
		if c == '\n' {
			// 遇到换行了，把之前的字符先输出到 buf 中
			s.buf.Write(b[k:i])
			// 不输出换行本身 在下一个字符到来时再处理(加上分隔符)
			k = i + 1
			// 如果是连续换行 需要记录一下个数
			s.needNewline++
			if s.wantDetail {
				s.switchOver()
			}
		} else {
			if s.needNewline > 0 && s.notEmpty {
				// 输出换行
				for i := 0; i < s.needNewline-1; i++ {
					s.buf.Write(detailSep[:len(sep)-1])
				}
				s.buf.Write(sep)
				s.needNewline = 0
			}
			s.notEmpty = true
		}
	}
	s.buf.Write(b[k:])
	return len(b), nil
}

func (p *state) detail() bool {
	if !p.wantDetail {
		return false
	}
	if p.notEmpty {
		p.needNewline = 1
	}
	p.switchOver()
	return true
}

func (p *state) switchOver() {
	if p.hasDetail {
		return
	}
	p.headBuf = p.buf.Bytes()
	p.buf = bytes.Buffer{}
	p.notEmpty = false
	p.hasDetail = true
}

type printer state

// Detail implements Printer.
func (s *printer) Detail() bool {
	return ((*state)(s)).detail()
}

// Print implements Printer.
func (s *printer) Print(args ...any) {
	s.enhanceArgs(args)
	fmt.Fprint((*state)(s), args...)
}

// Printf implements Printer.
func (s *printer) Printf(format string, args ...any) {
	s.enhanceArgs(args)
	fmt.Fprintf((*state)(s), format, args...)
}

func (s *printer) enhanceArgs(args []interface{}) {
	prevStack := s.lastStack
	lastSeen := prevStack
	for i := range args {
		/*
			if st, ok := GetStackTrace(args[i]); ok {
				//stackTrace, _ := ElideSharedStackTraceSuffix(prevStack, st)
				//_ = stackTrace
				// args[i] = StackDetail(stackTrace)
				lastSeen = st
			}
		*/
		if err, ok := args[i].(error); ok {
			args[i] = &errorFormatter{err}
		}
	}
	s.lastStack = lastSeen
}

// ElideSharedStackTraceSuffix 省略共同的堆栈
func ElideSharedStackTraceSuffix(prevStack, newStack StackTrace) (StackTrace, bool) {
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

type MessageType int

const (
	Prefix MessageType = iota
	FullMessage
)

func extractPrefix(err, cause error) (string, MessageType) {
	causeSuffix := cause.Error()
	errMsg := err.Error()
	if strings.HasSuffix(errMsg, causeSuffix) {
		prefix := errMsg[:len(errMsg)-len(causeSuffix)]
		if len(prefix) == 0 {
			return "", Prefix
		}
		if strings.HasSuffix(prefix, ": ") {
			return prefix[:len(prefix)-2], Prefix
		}
	}
	return errMsg, FullMessage
}
