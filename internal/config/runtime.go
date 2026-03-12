package config

import "sort"

// ResolveRuntime merges a GlobalConfig and a ProjectConfig into a RuntimeConfig
// for the named app. If appName is empty and exactly one app is configured, that
// app is used automatically. If appName is empty and multiple apps are configured,
// AppAmbiguousError is returned. If appName is provided but not found,
// AppNotFoundError is returned.
func ResolveRuntime(global *GlobalConfig, project *ProjectConfig, appName string) (*RuntimeConfig, error) {
	inst, ok := global.Instances[project.Instance]
	if !ok {
		return nil, &InstanceNotFoundError{Name: project.Instance}
	}

	// Resolve which app to use.
	var resolvedName string
	var resolvedApp AppConfig

	switch {
	case appName == "" && len(project.Apps) == 1:
		// Auto-select the only app.
		for k, v := range project.Apps {
			resolvedName = k
			resolvedApp = v
		}

	case appName == "" && len(project.Apps) > 1:
		keys := sortedKeys(project.Apps)
		return nil, &AppAmbiguousError{AvailableApps: keys}

	default:
		// appName was provided — look it up.
		app, found := project.Apps[appName]
		if !found {
			keys := sortedKeys(project.Apps)
			return nil, &AppNotFoundError{Name: appName, AvailableApps: keys}
		}
		resolvedName = appName
		resolvedApp = app
	}

	allowed := make(map[string]bool, len(AllAgentCommands))
	for _, cmd := range AllAgentCommands {
		allowed[cmd] = true
	}

	// Three-layer deny merge: global -> project -> app.
	for _, cmd := range global.Defaults.Permissions.Deny {
		allowed[cmd] = false
	}
	for _, cmd := range project.Permissions.Deny {
		allowed[cmd] = false
	}
	for _, cmd := range resolvedApp.Permissions.Deny {
		allowed[cmd] = false
	}

	return &RuntimeConfig{
		InstanceName: project.Instance,
		InstanceURL:  inst.URL,
		Token:        inst.Token,
		AppUUID:      resolvedApp.UUID,
		AppName:      resolvedName,
		AllowedCmds:  allowed,
	}, nil
}

// sortedKeys returns a sorted slice of keys from an AppConfig map.
func sortedKeys(m map[string]AppConfig) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
