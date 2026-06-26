package app

import (
	"fmt"
	"strings"
)

type flagSpec struct {
	Short string
	Long  string
}

func parseStringFlags(args []string, defaults map[string]string, specs ...flagSpec) (map[string]string, error) {
	values := make(map[string]string, len(defaults))
	for name, value := range defaults {
		values[name] = value
	}
	lookup := make(map[string]string, len(specs)*2)
	for _, spec := range specs {
		lookup[spec.Short] = spec.Long
		lookup[spec.Long] = spec.Long
	}
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if strings.HasPrefix(arg, "--") && strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			name, ok := lookup[parts[0]]
			if !ok {
				return nil, fmt.Errorf("unknown flag: %s", parts[0])
			}
			values[name] = parts[1]
			continue
		}
		name, ok := lookup[arg]
		if !ok {
			if strings.HasPrefix(arg, "-") {
				return nil, fmt.Errorf("unknown flag: %s", arg)
			}
			return nil, fmt.Errorf("unexpected argument: %s", arg)
		}
		if index+1 >= len(args) {
			return nil, fmt.Errorf("%s requires a value", arg)
		}
		values[name] = args[index+1]
		index++
	}
	return values, nil
}
