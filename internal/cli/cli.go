package cli

import (
	"fmt"
	"io"
	"runtime/debug"

	"github.com/delriscotechnologies/ghosttag/internal/assessment"
	"github.com/delriscotechnologies/ghosttag/internal/inspect"
	"github.com/delriscotechnologies/ghosttag/internal/report"
)

// Version may be replaced at build time. Standard Go module installations use
// the version recorded in the binary's build information.
var Version = "dev"

// Run executes the command and returns a process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 {
		switch args[0] {
		case "-h", "--help":
			writeUsage(stdout)
			return 0
		case "--version":
			fmt.Fprintf(stdout, "ghosttag %s\n", displayVersion())
			return 0
		}
	}

	if len(args) != 1 {
		fmt.Fprintln(stderr, "ghosttag: expected exactly one JPEG or PNG file")
		writeUsage(stderr)
		return 2
	}

	result, err := inspect.File(args[0])
	if err != nil {
		fmt.Fprintf(stderr, "ghosttag: %v\n", err)
		return 1
	}

	result.Assessment = assessment.Evaluate(result.Metadata)
	if err := report.WriteText(stdout, result); err != nil {
		fmt.Fprintf(stderr, "ghosttag: write report: %v\n", err)
		return 1
	}

	return 0
}

func displayVersion() string {
	if Version != "" && Version != "dev" {
		return Version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" || info.Main.Version == "(devel)" {
		return "dev"
	}
	return info.Main.Version
}

func writeUsage(writer io.Writer) {
	fmt.Fprintln(writer, "Usage: ghosttag <image.jpg|image.jpeg|image.png>")
	fmt.Fprintln(writer, "       ghosttag --help")
	fmt.Fprintln(writer, "       ghosttag --version")
}
