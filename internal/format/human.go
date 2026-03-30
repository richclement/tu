package format

import (
	"bytes"
	"fmt"
	"text/tabwriter"

	"github.com/richclement/tu/internal/report"
)

func Human(scanReport report.ScanReport) []byte {
	if len(scanReport.Results) == 0 {
		if scanReport.Threshold != nil {
			return []byte("No entries matched threshold.\n")
		}

		return []byte("No files found.\n")
	}

	var buf bytes.Buffer
	writer := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)

	fmt.Fprintln(writer, "TOKENS\tMETHOD\tSTATUS\tPATH")
	for _, result := range scanReport.Results {
		fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\n",
			formatNullableInt(result.Tokens),
			formatNullableMethod(result.Method),
			formatHumanStatus(result),
			result.Path,
		)
	}

	_ = writer.Flush()

	return buf.Bytes()
}

func formatHumanStatus(result report.Result) string {
	status := string(result.Status)
	if result.Reason == nil {
		return status
	}

	return fmt.Sprintf("%s:%s", status, *result.Reason)
}
