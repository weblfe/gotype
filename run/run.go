package run

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type (
	Runner struct {
		bin          string
		err          *os.File
		output       *os.File
		input        *os.File
		flagHandlers map[string]func(string) (string, error)
	}

	Result struct {
		output string
		err    error
	}

	commandType string
	typeMatch   struct {
		T          commandType
		MatchWords []string
	}
)

const (
	builtInType              = `BUILTIN_TYPE_BIN` // 系统自带 type
	defaultBind              = `/usr/bin/type`
	TypeAlias    commandType = "alias"
	TypeKeyword  commandType = "keyword"
	TypeFunction commandType = "function"
	TypeBuiltin  commandType = "builtin"
	TypeFile     commandType = "file"
	TypeUnFound  commandType = "unfound"
)

var (
	types = []commandType{
		TypeAlias,
		TypeKeyword,
		TypeFunction,
		TypeBuiltin,
		TypeFile,
		TypeUnFound,
	}
	typesMatches = map[commandType]typeMatch{
		TypeAlias: {
			TypeAlias, []string{"alias"},
		},
		TypeKeyword: {
			TypeKeyword, []string{"word"},
		},
		TypeFunction: {
			TypeFunction, []string{"shell function", "function"},
		},
		TypeBuiltin: {
			TypeBuiltin, []string{"builtin"},
		},
		TypeFile: {
			TypeFile, []string{"file", "is"},
		},
		TypeUnFound: {
			TypeUnFound, []string{"not", "found"},
		},
	}
)

func (ty commandType) String() string {
	return string(ty)
}

func (ty commandType) Eq(str string) bool {
	return strings.EqualFold(str, ty.String())
}

func (ty commandType) Match(info string) bool {
	if ty.Eq(info) {
		return true
	}
	if v, ok := typesMatches[ty]; ok {
		for _, w := range v.MatchWords {
			if strings.Contains(info, w) {
				return true
			}
		}
	}
	return false
}

var (
	shortFlagMap = map[string]string{
		"-t": "type",
		"-a": "all",
		"-p": "path",
	}
)

func NewResult() *Result {
	var r = new(Result)
	return r
}

func (rs *Result) Get() string {
	return rs.output
}

func (rs *Result) Deal(handler func(data string) bool) bool {
	if rs.HasErr() || handler == nil {
		return false
	}
	return handler(rs.Get())
}

func (rs *Result) HasErr() bool {
	if rs.err == nil {
		return false
	}
	return true
}

func (rs *Result) Err() error {
	return rs.err
}

// NewRunner 构造执行器
func NewRunner(err, input, output *os.File) *Runner {
	var runner = new(Runner)
	runner.init()
	runner.SetOut(err, input, output)
	return runner
}

func (r *Runner) init() {
	r.bin = GetEnvOr(builtInType, defaultBind)
	r.flagHandlers = r.createHandlers()
}

func (r *Runner) Bind(bin string) *Runner {
	if bin == "" {
		return r
	}
	if !filepath.IsAbs(bin) {
		p, err := filepath.Abs(bin)
		if err != nil {
			r.errLog("ERROR:", err.Error())
			return r
		}
		if _, err = os.Stat(p); err != nil {
			r.errLog("ERROR:", err.Error())
			return r
		}
		r.bin = p
	}
	return r
}

func (r *Runner) createHandlers() map[string]func(string) (string, error) {
	return map[string]func(string) (string, error){
		"type": r.parseType,
		"all":  r.parseAll,
		"path": r.parsePath,
	}
}

func (r *Runner) parseType(cmd string) (string, error) {
	var (
		args       = []string{`-a`, cmd}
		command    = r.command(args)
		bytes, err = command.Output()
	)
	if err != nil {
		return TypeUnFound.String(), nil
	}
	if len(bytes) <= 0 {
		return TypeUnFound.String(), nil
	}
	var lines = strings.Split(string(bytes), "\n")
	return r.getType(lines[0]).String(), nil
}

func (r *Runner) command(args []string) *exec.Cmd {
	var command = exec.Command(r.bin, args...)
	command.Env = os.Environ()
	return command
}

func (r *Runner) parseAll(cmd string) (string, error) {
	var (
		args       = []string{`-a`, cmd}
		command    = r.command(args)
		bytes, err = command.Output()
	)
	if err != nil {
		return cmd + ` not found`, nil
	}
	return string(bytes), nil
}

func (r *Runner) parsePath(cmd string) (string, error) {
	var (
		args       = []string{`-a`, cmd}
		command    = r.command(args)
		bytes, err = command.Output()
	)
	if err != nil {
		return cmd + ` not found`, nil
	}
	var (
		lines = strings.Split(string(bytes), "\n")
		ty    = r.getType(lines[0])
	)
	switch ty {
	case TypeFile:
		var (
			index = 1
			argc  = len(lines)
		)
		if argc < 2 {
			index = 0
		}
		var paths = strings.Split(lines[index], `is`)
		return strings.TrimSpace(paths[1]), nil
	}
	return ``, nil
}

func (r *Runner) getType(info string) commandType {
	for _, v := range types {
		if v.Match(info) {
			return v
		}
	}
	return TypeUnFound
}

func (r *Runner) SetOut(err, input, output *os.File) *Runner {
	r.err = err
	r.input = input
	r.output = output
	return r
}

func (r *Runner) Exec(flag string, cmd string) *Result {
	var rs = NewResult()
	if err := r.check(); err != nil {
		rs.err = err
		r.errLog("cmd err:", rs.err)
		return rs
	}
	if cmd == "" {
		return rs
	}
	if strings.HasPrefix(flag, "-") {
		flag = r.short2Long(flag)
	}
	if fn, ok := r.flagHandlers[flag]; ok {
		rs.output, rs.err = fn(cmd)
	} else {
		rs.err = errors.New(flag + `:flag undefined `)
	}
	var out = rs.Get()
	if strings.HasSuffix(out, "\n") {
		r.print(out)
	} else {
		r.println(out)
	}
	return rs
}

func (r *Runner) print(args ...interface{}) int {
	if r.output == nil {
		if n, err := fmt.Print(args...); err == nil {
			return n
		}
		return 0
	}
	if n, err := fmt.Fprint(r.output, args...); err == nil {
		return n
	}
	return 0
}

func (r *Runner) println(args ...interface{}) int {
	if r.output == nil {
		if n, err := fmt.Println(args...); err == nil {
			return n
		}
		return 0
	}
	if n, err := fmt.Fprintln(r.output, args...); err == nil {
		return n
	}
	return 0
}

func (r *Runner) errLog(args ...interface{}) int {
	if r.output == nil {
		if n, err := fmt.Println(args...); err == nil {
			return n
		}
		return 0
	}
	if n, err := fmt.Fprintln(r.err, args...); err == nil {
		return n
	}
	return 0
}

func (r *Runner) short2Long(flag string) string {
	if v, ok := shortFlagMap[strings.ToLower(flag)]; ok {
		return v
	}
	return flag
}

func (r *Runner) check() error {
	if r.flagHandlers == nil {
		r.flagHandlers = r.createHandlers()
	}
	if r.bin == "" {
		return errors.New(`miss init type built env:`+builtInType)
	}
	return nil
}

func GetEnvOr(key string, defaultValue ...string) string {
	defaultValue = append(defaultValue, "")
	var v = os.Getenv(key)
	if v == "" {
		return defaultValue[0]
	}
	return v
}
