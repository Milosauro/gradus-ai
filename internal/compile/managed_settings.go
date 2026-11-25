package compile

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/Milosauro/gradus-ai/internal/resolve"
)

// RenderClaudeManagedSettings generates an enterprise Claude Code managed-settings.json
// payload from the resolved pyramid (hooks, permissions, forbidden MCP servers, and
// bypass-permissions lock when anti_tamper.mode is deny, §20).
func RenderClaudeManagedSettings(view resolve.ResolvedView, antiTamperMode string) ([]byte, error) {
	settings := make(map[string]any)

	hooksMap := ensureStringMap(settings, "hooks")
	ensureHookDispatcher(hooksMap, "PreToolUse", "Bash|Edit|Write|MultiEdit", "gradusctl guard")
	ensureHookDispatcher(hooksMap, "SessionStart", "", "gradusctl status --brief --hook")

	var denyList, askList []string
	for _, r := range view.Rules {
		if r.Waived || r.Level == "user" || r.Enforcement != "permission" || r.Guard == nil {
			continue
		}
		entry := formatClaudePermission(r.Guard.Action, r.Guard.Pattern)
		switch r.Guard.Decision {
		case "deny":
			denyList = append(denyList, entry)
		case "ask":
			askList = append(askList, entry)
		}
	}
	if len(denyList) > 0 || len(askList) > 0 {
		permMap := ensureStringMap(settings, "permissions")
		if len(denyList) > 0 {
			permMap["deny"] = mergeUniqueStrings(nil, denyList)
		}
		if len(askList) > 0 {
			permMap["ask"] = mergeUniqueStrings(nil, askList)
		}
	}

	if view.Toolchain != nil && len(view.Toolchain.Forbidden) > 0 {
		var mcpIDs []string
		seen := make(map[string]bool)
		for _, item := range view.Toolchain.Forbidden {
			if strings.EqualFold(strings.TrimSpace(item.Type), "mcp") {
				id := strings.TrimSpace(item.ID)
				if id != "" && !seen[id] {
					seen[id] = true
					mcpIDs = append(mcpIDs, id)
				}
			}
		}
		sort.Strings(mcpIDs)
		if len(mcpIDs) > 0 {
			denied := make([]map[string]string, 0, len(mcpIDs))
			for _, id := range mcpIDs {
				denied = append(denied, map[string]string{"serverName": id})
			}
			settings["deniedMcpServers"] = denied
		}
	}

	if strings.EqualFold(strings.TrimSpace(antiTamperMode), "deny") {
		settings["disableBypassPermissionsMode"] = "disable"
	}

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}
