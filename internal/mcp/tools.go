package mcp

func tools() []tool {
	return []tool{
		newTool(
			"scaffold_query",
			"Query Scaffold Agent",
			"Return compact support, workflow, or managed-project facts. Use this first to avoid reading implementation files.",
			objectSchema(map[string]any{
				"topic":        enumString("support", "workflow", "project"),
				"project_root": stringProperty("Absolute project root; required when topic is project."),
			}, []string{"topic"}),
			true, false, true,
		),
		newTool(
			"scaffold_plan",
			"Plan Project Changes",
			"Validate a Blueprint, resolve capabilities, generate outputs, and store an immutable plan without changing generated project files.",
			objectSchema(map[string]any{
				"project_root":   stringProperty("Absolute project root."),
				"blueprint_path": stringProperty("Canonical project-relative Blueprint path using forward slashes."),
				"action":         enumString("create", "modify", "extend", "reduce", "repair", "upgrade"),
			}, []string{"project_root", "blueprint_path", "action"}),
			false, false, true,
		),
		newTool(
			"scaffold_preview",
			"Preview Project Plan",
			"Read one compact page of a stored plan and return the apply_token required to apply that exact plan.",
			pagedSchema(map[string]any{
				"project_root": stringProperty("Absolute project root."),
				"plan_id":      stringProperty("Content-addressed plan ID."),
			}, []string{"project_root", "plan_id"}),
			true, false, true,
		),
		newTool(
			"scaffold_apply",
			"Apply Project Plan",
			"Apply one previously previewed immutable plan transactionally. Refuses stale, user-modified, or unowned files.",
			objectSchema(map[string]any{
				"project_root": stringProperty("Absolute project root."),
				"plan_id":      stringProperty("Content-addressed plan ID."),
				"apply_token":  stringProperty("Token returned by scaffold_preview for this exact plan."),
			}, []string{"project_root", "plan_id", "apply_token"}),
			false, true, true,
		),
		newTool(
			"scaffold_verify",
			"Verify Managed Project",
			"Hash every Engine-managed file, store the complete findings, and return the first compact result page.",
			objectSchema(map[string]any{
				"project_root": stringProperty("Absolute project root."),
				"limit":        integerProperty(1, 100),
			}, []string{"project_root"}),
			false, false, true,
		),
		newTool(
			"scaffold_result",
			"Read Stored Result",
			"Read one bounded page from a stored validation, preview, or verification result instead of loading it all into context.",
			pagedSchema(map[string]any{
				"project_root": stringProperty("Absolute project root."),
				"result_id":    stringProperty("Content-addressed result ID."),
			}, []string{"project_root", "result_id"}),
			true, false, true,
		),
	}
}

func newTool(name, title, description string, inputSchema map[string]any, readOnly, destructive, idempotent bool) tool {
	return tool{
		Name:        name,
		Title:       title,
		Description: description,
		InputSchema: inputSchema,
		Annotations: toolAnnotations{
			Title:           title,
			ReadOnlyHint:    readOnly,
			DestructiveHint: destructive,
			IdempotentHint:  idempotent,
			OpenWorldHint:   false,
		},
	}
}

func objectSchema(properties map[string]any, required []string) map[string]any {
	return map[string]any{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"type":                 "object",
		"additionalProperties": false,
		"properties":           properties,
		"required":             required,
	}
}

func pagedSchema(properties map[string]any, required []string) map[string]any {
	properties["cursor"] = stringProperty("Opaque cursor returned by the previous page.")
	properties["limit"] = integerProperty(1, 100)
	return objectSchema(properties, required)
}

func enumString(values ...string) map[string]any {
	items := make([]any, 0, len(values))
	for _, value := range values {
		items = append(items, value)
	}
	return map[string]any{"type": "string", "enum": items}
}

func stringProperty(description string) map[string]any {
	return map[string]any{"type": "string", "minLength": 1, "description": description}
}

func integerProperty(minimum, maximum int) map[string]any {
	return map[string]any{"type": "integer", "minimum": minimum, "maximum": maximum}
}
