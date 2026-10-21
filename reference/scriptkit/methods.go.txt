package scriptkit

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

func (sc *Script) parseArgs() []string {
	re := regexp.MustCompile(`(\|=.*=\|)`)
	return re.FindAllString(sc.template, -1)
}

func (sc *Script) Render() string {
	text := sc.template
	for key, arg := range sc.args {
		text = strings.ReplaceAll(text, key, fmt.Sprintf("%v", arg))
	}
	return text
}

// CreateScriptFile - create new or rewrite file associated with script.
func (sc *Script) CreateScriptFile() error {
	if err := sc.Validate(); err != nil {
		return fmt.Errorf("script validation failed: %v", err)
	}

	f, err := os.OpenFile(sc.path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, sc.permission)
	if err != nil {
		return fmt.Errorf("failed create file: %v", err)
	}
	defer f.Close()
	if _, err := f.WriteString(sc.Render()); err != nil {
		return fmt.Errorf("write to file failed: %v", err)
	}
	return nil
}

func (sc *Script) Move(newpath string) error {
	err := os.Rename(sc.path, newpath)
	if err != nil {
		return fmt.Errorf("failed to move script file: %v", err)
	}
	sc.path = newpath
	return nil
}

// Clean - delete file associated with script.
func (sc *Script) Clean() error {
	err := os.Remove(sc.path)
	if err != nil {
		return fmt.Errorf("failed to clean script file: %v", err)
	}
	return nil
}

// SetArg - Dynamicly update value of argument by key
func (sc *Script) SetArg(key string, value any) {
	sc.args[argKeyFormat(key)] = value
}

// SetPermission - set custom file rights.
func (sc *Script) SetPermission(perm os.FileMode) {
	sc.permission = perm
}

// Validate - check template/arguments matching.
func (sc *Script) Validate() error {
	keys := sc.parseArgs()
	for _, key := range keys {
		if !strings.Contains(sc.template, key) {
			return fmt.Errorf("parsed arg %v is missing in script args", key)
		}
	}
	return nil
}

func (sc *Script) Path() string {
	return sc.path
}
