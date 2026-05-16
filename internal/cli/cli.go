package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"agent-recall/internal/config"
	"agent-recall/internal/hooks"
	"agent-recall/internal/install"
	"agent-recall/internal/mcp"
	"agent-recall/internal/search"
	"agent-recall/internal/store"
)

const version = "0.1.0"

func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		printUsage(stdout)
		return nil
	}

	switch args[0] {
	case "hook-sync":
		return runHook(args[1:], stdin, stderr, false)
	case "hook-flush":
		return runHook(args[1:], stdin, stderr, true)
	case "mcp":
		return runMCP(args[1:], stdin, stdout, stderr)
	case "recall":
		return runRecall(args[1:], stdout)
	case "status":
		return runStatus(args[1:], stdout)
	case "install":
		return runInstall(args[1:], stdout)
	case "version", "--version", "-v":
		fmt.Fprintln(stdout, version)
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `agent-recall records coding-agent session transcripts as external historical evidence.

Usage:
  agent-recall hook-sync [--store-dir DIR] [--strict]
  agent-recall hook-flush [--store-dir DIR] [--strict]
  agent-recall mcp [--store-dir DIR]
  agent-recall recall <query> [--store-dir DIR] [--json] [--limit N]
  agent-recall status [--store-dir DIR] [--json]
  agent-recall install claude-code [--dry-run] [--force] [--store-dir DIR]
`)
}

func runHook(args []string, stdin io.Reader, stderr io.Writer, flush bool) error {
	fs := flag.NewFlagSet("hook", flag.ContinueOnError)
	fs.SetOutput(stderr)
	storeDir := fs.String("store-dir", "", "store directory")
	strict := fs.Bool("strict", false, "return errors instead of soft-failing")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := hooks.Sync(stdin, hooks.SyncOptions{StoreDir: *storeDir, Flush: flush}); err != nil {
		fmt.Fprintf(stderr, "agent-recall: %v\n", err)
		if *strict {
			return err
		}
	}
	return nil
}

func runMCP(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	storeDir := fs.String("store-dir", "", "store directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	dir, err := config.StoreDir(*storeDir)
	if err != nil {
		return err
	}
	return mcp.Serve(stdin, stdout, stderr, dir)
}

func runRecall(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("recall", flag.ContinueOnError)
	storeDir := fs.String("store-dir", "", "store directory")
	jsonOut := fs.Bool("json", false, "print JSON")
	limit := fs.Int("limit", 10, "result limit")
	sessionID := fs.String("session-id", "", "session id")
	cwd := fs.String("cwd", "", "working directory filter")
	if err := fs.Parse(args); err != nil {
		return err
	}
	query := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if query == "" {
		return fmt.Errorf("recall requires a query")
	}
	dir, err := config.StoreDir(*storeDir)
	if err != nil {
		return err
	}
	res, err := search.Recall(dir, search.Query{Text: query, Limit: *limit, SessionID: *sessionID, CWD: *cwd})
	if err != nil {
		return err
	}
	return search.WriteResult(stdout, res, *jsonOut)
}

func runStatus(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	storeDir := fs.String("store-dir", "", "store directory")
	jsonOut := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	dir, err := config.StoreDir(*storeDir)
	if err != nil {
		return err
	}
	st, err := store.Status(dir)
	if err != nil {
		return err
	}
	return store.WriteStatus(stdout, st, *jsonOut)
}

func runInstall(args []string, stdout io.Writer) error {
	if len(args) == 0 || args[0] != "claude-code" {
		return fmt.Errorf("install target must be claude-code")
	}
	fs := flag.NewFlagSet("install claude-code", flag.ContinueOnError)
	storeDir := fs.String("store-dir", "", "store directory")
	dryRun := fs.Bool("dry-run", false, "show changes without writing")
	force := fs.Bool("force", false, "overwrite generated templates")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	return install.ClaudeCode(install.Options{StoreDir: *storeDir, DryRun: *dryRun, Force: *force}, stdout)
}
