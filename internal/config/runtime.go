package config

// ResolveRuntime merges a GlobalConfig and a ProjectConfig into a RuntimeConfig.
// It looks up the instance referenced by the project config in the global config
// and builds the AllowedCmds map by applying both deny lists.
func ResolveRuntime(global *GlobalConfig, project *ProjectConfig) (*RuntimeConfig, error) {
	inst, ok := global.Instances[project.Instance]
	if !ok {
		return nil, &InstanceNotFoundError{Name: project.Instance}
	}

	// All known agent commands — keep in sync with permissions.AllAgentCommands.
	allCmds := []string{"deploy", "redeploy", "logs", "status", "list"}

	allowed := make(map[string]bool, len(allCmds))
	for _, cmd := range allCmds {
		allowed[cmd] = true
	}

	// Apply global deny list.
	for _, cmd := range global.Defaults.Permissions.Deny {
		allowed[cmd] = false
	}

	// Apply project deny list (can only further restrict, never escalate).
	for _, cmd := range project.Permissions.Deny {
		allowed[cmd] = false
	}

	return &RuntimeConfig{
		InstanceName: project.Instance,
		InstanceURL:  inst.URL,
		Token:        inst.Token,
		AppUUID:      project.AppUUID,
		AllowedCmds:  allowed,
	}, nil
}
