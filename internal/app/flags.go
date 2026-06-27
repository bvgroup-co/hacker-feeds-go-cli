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

func parseProductDetailsFlags(args []string) (map[string]string, error) {
	values := map[string]string{"--url": "", "--slug": ""}
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if strings.HasPrefix(arg, "--") && strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			switch parts[0] {
			case "--url", "--slug":
				values[parts[0]] = parts[1]
			case "--count", "--past":
				return nil, fmt.Errorf("%s is not supported in product details mode", parts[0])
			default:
				return nil, fmt.Errorf("unknown flag: %s", parts[0])
			}
			continue
		}
		if arg == "--count" || arg == "-c" || arg == "--past" || arg == "-p" {
			return nil, fmt.Errorf("%s is not supported in product details mode", arg)
		}
		if arg != "--url" && arg != "--slug" {
			if strings.HasPrefix(arg, "-") {
				return nil, fmt.Errorf("unknown flag: %s", arg)
			}
			return nil, fmt.Errorf("unexpected argument: %s", arg)
		}
		if index+1 >= len(args) {
			return nil, fmt.Errorf("%s requires a value", arg)
		}
		values[arg] = args[index+1]
		index++
	}
	return values, nil
}

func hasAnyFlag(args []string, names ...string) bool {
	lookup := make(map[string]bool, len(names))
	for _, name := range names {
		lookup[name] = true
	}
	for _, arg := range args {
		name := arg
		if strings.HasPrefix(arg, "--") && strings.Contains(arg, "=") {
			name = strings.SplitN(arg, "=", 2)[0]
		}
		if lookup[name] {
			return true
		}
	}
	return false
}

func wantsHelp(args []string) bool {
	return len(args) > 0 && (args[0] == "--help" || args[0] == "-h" || args[0] == "help")
}
