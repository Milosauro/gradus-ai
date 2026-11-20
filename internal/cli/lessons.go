package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/Milosauro/gradus-ai/internal/state"
)

func newLessonsCmd(jsonOutput *bool) *cobra.Command {
	var dir string
	var lessonsDir string

	cmd := &cobra.Command{
		Use:   "lessons",
		Short: "Manage local redacted session lessons (~/.gradus/lessons/)",
	}
	cmd.PersistentFlags().StringVar(&dir, "dir", ".", "Target repository directory")
	cmd.PersistentFlags().StringVar(&lessonsDir, "lessons-dir", "", "Override lessons storage directory")

	// add
	var kind, statement, level, enforcement string
	var evidenceItems []string
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a redacted lesson record",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			storeDir, err := state.LessonsDir(dir, lessonsDir)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}
			rec, path, err := state.AddLesson(storeDir, state.LessonRecord{
				Kind:                 kind,
				Statement:            statement,
				Evidence:             evidenceItems,
				SuggestedLevel:       level,
				SuggestedEnforcement: enforcement,
			}, time.Now().UTC())
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}
			if *jsonOutput {
				return encodeJSON(cmd, map[string]any{
					"lesson": rec,
					"path":   path,
				})
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "added lesson %s (%s) -> %s\n", rec.ID, rec.Kind, path)
			return err
		},
	}
	addCmd.Flags().StringVar(&kind, "kind", "correction", "Lesson kind (correction|repeated-mistake|preference|frustration)")
	addCmd.Flags().StringVar(&statement, "statement", "", "Lesson statement (secrets/PII are automatically redacted)")
	addCmd.Flags().StringArrayVar(&evidenceItems, "evidence", nil, "Evidence snippet (redacted before write)")
	addCmd.Flags().StringVar(&level, "level", "", "Suggested target level")
	addCmd.Flags().StringVar(&enforcement, "enforcement", "", "Suggested enforcement mode")

	// list
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List stored lessons for the project",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			storeDir, err := state.LessonsDir(dir, lessonsDir)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}
			items, err := state.ListLessons(storeDir)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}
			if *jsonOutput {
				return encodeJSON(cmd, map[string]any{"lessons": items})
			}
			if len(items) == 0 {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "No lessons recorded.")
				return err
			}
			for _, it := range items {
				fmt.Fprintf(cmd.OutOrStdout(), "%s [%s] (%dx) %s\n", it.ID, it.Status, it.Occurrences, it.Statement)
			}
			return nil
		},
	}

	// update
	var id, status string
	var increment bool
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update a lesson status or increment its occurrence count",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = "."
			}
			storeDir, err := state.LessonsDir(dir, lessonsDir)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}
			rec, err := state.UpdateLesson(storeDir, id, status, increment)
			if err != nil {
				return &ExitErrorWithCode{Code: ExitError, Err: err}
			}
			if *jsonOutput {
				return encodeJSON(cmd, map[string]any{"lesson": rec})
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "updated lesson %s [%s] (%dx)\n", rec.ID, rec.Status, rec.Occurrences)
			return err
		},
	}
	updateCmd.Flags().StringVar(&id, "id", "", "Lesson ID")
	updateCmd.Flags().StringVar(&status, "status", "", "New status (open|proposed|promoted|dismissed)")
	updateCmd.Flags().BoolVar(&increment, "increment", false, "Increment occurrence counter")

	cmd.AddCommand(addCmd, listCmd, updateCmd)
	return cmd
}

var _ = os.ErrNotExist
