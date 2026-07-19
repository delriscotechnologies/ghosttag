package cli

import (
	"fmt"
	"io"
	"runtime/debug"
	"strings"
	"unicode"

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
			if err := writeUsage(stdout); err != nil {
				writeDiagnostic(stderr, "write help: %v", err)
				return 1
			}
			return 0
		case "--version":
			if _, err := fmt.Fprintf(stdout, "ghosttag %s\n", displayVersion()); err != nil {
				writeDiagnostic(stderr, "write version: %v", err)
				return 1
			}
			return 0
		}
	}

	if len(args) != 1 {
		if _, err := fmt.Fprintln(stderr, "ghosttag: expected exactly one JPEG or PNG file"); err != nil {
			return 1
		}
		if err := writeUsage(stderr); err != nil {
			return 1
		}
		return 2
	}

	result, err := inspect.File(args[0])
	if err != nil {
		writeDiagnostic(stderr, "%v", err)
		return 1
	}

	result.Assessment = assessment.Evaluate(result.Metadata)
	if err := report.WriteText(stdout, result); err != nil {
		writeDiagnostic(stderr, "write report: %v", err)
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

func writeUsage(writer io.Writer) error {
	for _, line := range []string{
		"Usage: ghosttag <image.jpg|image.jpeg|image.png>",
		"       ghosttag --help",
		"       ghosttag --version",
	} {
		if _, err := fmt.Fprintln(writer, line); err != nil {
			return err
		}
	}
	return nil
}

func writeDiagnostic(writer io.Writer, format string, args ...any) {
	message := sanitizeDiagnostic(fmt.Sprintf(format, args...))
	_, _ = fmt.Fprintf(writer, "ghosttag: %s\n", message)
}

func sanitizeDiagnostic(value string) string {
	var builder strings.Builder
	for _, character := range value {
		if unicode.IsControl(character) || unicode.In(character, unicode.Cf) {
			builder.WriteByte(' ')
			continue
		}
		builder.WriteRune(character)
	}
	return strings.Join(strings.Fields(builder.String()), " ")
}
