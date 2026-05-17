// SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
// SPDX-License-Identifier: EUPL-1.2

//go:build with_mrs

package app

import (
	"flag"
	"fmt"
	"io"

	"gitflower/internal/git"
)

func cmdMR(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printMRUsage(stderr)
		return 1
	}
	switch args[0] {
	case "list":
		return cmdMRList(args[1:], stdout, stderr)
	case "-h", "--help":
		printMRUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "mr: unknown subcommand %q\n\n", args[0])
		printMRUsage(stderr)
		return 1
	}
}

func printMRUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: gitflower mr <subcommand>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Subcommands:")
	fmt.Fprintln(w, "  list     List active merge requests (refs/heads/mr/*)")
}

func cmdMRList(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("mr list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	archived := fs.Bool("archived", false, "list archived MRs instead (refs/heads/archive/mr/*)")
	all := fs.Bool("all", false, "list both active and archived MRs")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	var prefixes []string
	switch {
	case *all:
		prefixes = []string{"refs/heads/mr/", "refs/heads/archive/mr/"}
	case *archived:
		prefixes = []string{"refs/heads/archive/mr/"}
	default:
		prefixes = []string{"refs/heads/mr/"}
	}

	repo, err := git.Open("")
	if err != nil {
		fmt.Fprintf(stderr, "mr list: %v\n", err)
		return 1
	}
	refs, err := repo.ForEachRefPrefix(prefixes...)
	if err != nil {
		fmt.Fprintf(stderr, "mr list: %v\n", err)
		return 1
	}
	if len(refs) == 0 {
		return 0
	}

	// Compute padding for nicer alignment, capped to 60 cols.
	pad := 0
	for _, ri := range refs {
		if n := len(ri.Name); n > pad {
			pad = n
		}
	}
	if pad > 60 {
		pad = 60
	}
	for _, ri := range refs {
		fmt.Fprintf(stdout, "%-*s  %s\n", pad, ri.Name, ri.Subject)
	}
	return 0
}
