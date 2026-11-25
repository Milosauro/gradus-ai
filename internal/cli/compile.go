package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Milosauro/gradus-ai/internal/compile"
	"github.com/Milosauro/gradus-ai/internal/resolve"
	"github.com/Milosauro/gradus-ai/internal/schema"
	"github.com/Milosauro/gradus-ai/skills"
)

func newCompileCmd(jsonOutput *bool) *cobra.Command {
	var dir, agentProfile, managedSettingsPath, chatBundlePath string

	cmd := &cobra.Command{
		Use:   "compile",
		Short: "Compile the resolved pyramid into target agent instructions, skills, and hooks",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}

			cfgBytes, err := os.ReadFile(filepath.Join(dir, "gradus.yaml"))
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: fmt.Errorf("failed to read gradus.yaml: %w", err)}
			}
			cfgRes, err := schema.ParsePyramidConfig(cfgBytes)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			resolved, err := resolve.Resolve(dir, resolve.Options{
				AgentProfile: agentProfile,
			})
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			if len(resolved.View.Conflicts) > 0 {
				return &ExitErrorWithCode{
					Code: ExitViolation,
					Err:  fmt.Errorf("compilation aborted: %d blocking conflict(s) detected", len(resolved.View.Conflicts)),
				}
			}

			res, err := compile.ApplyCompile(dir, resolved.View, cfgRes.Value.Targets, skills.FS)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}

			var writtenManagedSettings string
			if managedSettingsPath != "" {
				target := managedSettingsPath
				if !filepath.IsAbs(target) {
					target = filepath.Join(dir, target)
				}
				antiTamperMode := "ask"
				if cfgRes.Value.AntiTamper != nil && cfgRes.Value.AntiTamper.Mode != "" {
					antiTamperMode = cfgRes.Value.AntiTamper.Mode
				}
				msBytes, err := compile.RenderClaudeManagedSettings(resolved.View, antiTamperMode)
				if err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: err}
				}
				if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: err}
				}
				if err := os.WriteFile(target, msBytes, 0o644); err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: err}
				}
				writtenManagedSettings = target
			}

			var chatBundleRes *compile.ChatBundleResult
			if chatBundlePath != "" {
				target := chatBundlePath
				if !filepath.IsAbs(target) {
					target = filepath.Join(dir, target)
				}
				cb, err := compile.WriteChatBundleZIP(target, resolved.View, skills.FS)
				if err != nil {
					return &ExitErrorWithCode{Code: ExitError, Err: err}
				}
				chatBundleRes = &cb
			}

			if *jsonOutput {
				payload := struct {
					compile.CompileResult
					ManagedSettings string                    `json:"managed_settings,omitempty"`
					ChatBundle      *compile.ChatBundleResult `json:"chat_bundle,omitempty"`
				}{
					CompileResult:   res,
					ManagedSettings: writtenManagedSettings,
					ChatBundle:      chatBundleRes,
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(payload)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Compiled targets %v (view %s)\n", res.Targets, res.ViewHash)
			fmt.Fprintf(out, "Changed: %d, Unchanged: %d, Removed: %d\n",
				len(res.ChangedFiles), len(res.UnchangedFiles), len(res.RemovedFiles))
			if writtenManagedSettings != "" {
				fmt.Fprintf(out, "Managed settings: %s\n", writtenManagedSettings)
			}
			if chatBundleRes != nil {
				fmt.Fprintf(out, "Chat bundle: %s (%d rule(s), %d skill(s))\n",
					chatBundleRes.Path, chatBundleRes.RuleCount, len(chatBundleRes.Skills))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Target repository directory")
	cmd.Flags().StringVar(&agentProfile, "agent-profile", "", "Team agent profile directory replacing personal user rules")
	cmd.Flags().StringVar(&managedSettingsPath, "managed-settings", "", "Optional output path for enterprise Claude Code managed-settings.json")
	cmd.Flags().StringVar(&chatBundlePath, "chat-bundle", "", "Optional output ZIP path for portable chat-agent skills and shared rules")
	return cmd
}
