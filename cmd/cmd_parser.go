package cmd

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	Wrapper        = `"`
	WrapperEscaper = `\"`
	Separator      = `\s`
)

func genPattern(wrapper string) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf(`[^%s\s]+(%s[^%s]*%s\S*)+|[^%s\s]+|(%s[^%s]*%s\S*)+`, wrapper,
		wrapper, wrapper, wrapper, wrapper, wrapper, wrapper, wrapper))
}

func splitCliArgs(cli string) (args []string) {
	re := genPattern(Wrapper)
	cmdArgs := strings.Replace(cli, WrapperEscaper, "\x00", -1)
	args = re.FindAllString(cmdArgs, -1)
	for i, arg := range args {
		if strings.HasPrefix(args[i], `"`) && strings.HasSuffix(args[i], `"`) {
			arg = strings.Trim(args[i], `"`)
		}

		args[i] = strings.Replace(arg, "\x00", Wrapper, -1)
	}

	return
}
