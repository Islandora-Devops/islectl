package utils

import (
	"reflect"
	"testing"

	"github.com/spf13/cobra"
)

func TestGetContextFromArgs(t *testing.T) {
	rootCmd := &cobra.Command{Use: "root"}
	rootCmd.PersistentFlags().String("context", "default", "context flag")

	cmd := &cobra.Command{Use: "test"}
	rootCmd.AddCommand(cmd)

	tests := []struct {
		name         string
		args         []string
		expectedArgs []string
		expectedCtx  string
	}{
		{
			name:         "no context flag",
			args:         []string{"arg1", "arg2"},
			expectedArgs: []string{"arg1", "arg2"},
			expectedCtx:  "default",
		},
		{
			name:         "separate --context flag",
			args:         []string{"--context", "newcontext", "arg1"},
			expectedArgs: []string{"arg1"},
			expectedCtx:  "newcontext",
		},
		{
			name:         "equals --context flag",
			args:         []string{"--context=equalcontext", "arg1"},
			expectedArgs: []string{"arg1"},
			expectedCtx:  "equalcontext",
		},
		{
			name:         "multiple flags, last one wins",
			args:         []string{"arg0", "--context", "midcontext", "arg1", "--context=another", "arg2"},
			expectedArgs: []string{"arg0", "arg1", "arg2"},
			expectedCtx:  "another",
		},
		{
			name:         "context flag with extra quotes and spaces",
			args:         []string{"--context", ` " spacedcontext " `, "arg1"},
			expectedArgs: []string{"arg1"},
			expectedCtx:  "spacedcontext",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filteredArgs, ctx, err := GetContextFromArgs(cmd, tt.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(filteredArgs, tt.expectedArgs) {
				t.Errorf("expected filtered args %v, got %v", tt.expectedArgs, filteredArgs)
			}
			if ctx != tt.expectedCtx {
				t.Errorf("expected context %q, got %q", tt.expectedCtx, ctx)
			}
		})
	}
}
