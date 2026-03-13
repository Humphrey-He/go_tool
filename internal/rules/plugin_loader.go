package rules

import (
	"errors"
	"fmt"
	"plugin"
	"runtime"
)

type PluginRegister func(reg *Registry) error

func LoadPlugins(reg *Registry, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	if runtime.GOOS == "windows" {
		return errors.New("plugin loading is not supported on windows")
	}

	for _, path := range paths {
		p, err := plugin.Open(path)
		if err != nil {
			return fmt.Errorf("open plugin %s: %w", path, err)
		}
		sym, err := p.Lookup("Register")
		if err != nil {
			return fmt.Errorf("lookup Register in %s: %w", path, err)
		}
		register, ok := sym.(func(*Registry) error)
		if !ok {
			if typed, ok := sym.(*PluginRegister); ok {
				register = *typed
			} else {
				return fmt.Errorf("invalid Register signature in %s", path)
			}
		}
		if err := register(reg); err != nil {
			return fmt.Errorf("register plugin %s: %w", path, err)
		}
	}
	return nil
}
