package scriptkit

import (
	"fmt"
	"os"
	"strings"
)

type Script struct {
	path       string
	template   string
	permission os.FileMode
	args       map[string]any
}

func New(path string, options ...ScriptOption) *Script {
	sc := Script{}
	sc.path = path
	sc.args = make(map[string]any)
	sc.permission = 0777
	for _, modify := range options {
		modify(&sc)
	}
	return &sc
}

/*
sc.Move() error
sc.Run() error
sc.Validate() error
*/

type ScriptOption func(*Script)

func WithTemplate(template string) ScriptOption {
	return func(s *Script) {
		s.template = template
	}
}

func WithArgs(args ...ScriptArgument) ScriptOption {
	return func(s *Script) {
		for _, arg := range args {
			s.args[argKeyFormat(arg.key)] = arg.value
		}
	}
}

type ScriptArgument struct {
	key   string
	value any
}

func ScriptArg(key string, value any) ScriptArgument {
	return ScriptArgument{key: key, value: value}
}

func argKeyFormat(key string) string {
	key = strings.TrimPrefix(key, "|=")
	key = strings.TrimSuffix(key, "=|")
	return fmt.Sprintf("|=%s=|", key)
}
