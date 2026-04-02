package format

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/richclement/tu/internal/report"
)

func JSON(scanReport report.ScanReport) ([]byte, error) {
	var buf bytes.Buffer

	scanReport.Results = ensureResults(scanReport.Results)
	scanReport.SymlinkMode = ensureSymlinkMode(scanReport.SymlinkMode)

	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(scanReport); err != nil {
		return nil, fmt.Errorf("encode json report: %w", err)
	}

	return buf.Bytes(), nil
}

func Plain(scanReport report.ScanReport) []byte {
	if len(scanReport.Results) == 0 {
		return nil
	}

	lines := make([]string, 0, len(scanReport.Results))
	for _, result := range scanReport.Results {
		lines = append(lines, fmt.Sprintf(
			"%s\t%s\t%s\t%s",
			formatNullableInt(result.Tokens),
			formatNullableMethod(result.Method),
			result.Status,
			escapePlainPath(result.Path),
		))
	}

	return []byte(strings.Join(lines, "\n") + "\n")
}

func CSV(scanReport report.ScanReport) ([]byte, error) {
	var buf bytes.Buffer

	writer := csv.NewWriter(&buf)
	if err := writer.Write([]string{"kind", "path", "tokens", "method", "provider", "status", "reason"}); err != nil {
		return nil, fmt.Errorf("write csv header: %w", err)
	}

	for _, result := range ensureResults(scanReport.Results) {
		if err := writer.Write([]string{
			string(result.Kind),
			result.Path,
			formatNullableInt(result.Tokens),
			formatNullableMethod(result.Method),
			formatNullableString(result.Provider),
			string(result.Status),
			formatNullableString(result.Reason),
		}); err != nil {
			return nil, fmt.Errorf("write csv row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("flush csv report: %w", err)
	}

	return buf.Bytes(), nil
}

func ensureResults(results []report.Result) []report.Result {
	if results == nil {
		return []report.Result{}
	}

	return results
}

func ensureSymlinkMode(mode report.SymlinkMode) report.SymlinkMode {
	if mode == "" {
		return report.SymlinkModePhysical
	}

	return mode
}

func formatNullableInt(value *int64) string {
	if value == nil {
		return ""
	}

	return fmt.Sprintf("%d", *value)
}

func formatNullableMethod(value *report.Method) string {
	if value == nil {
		return ""
	}

	return string(*value)
}

func formatNullableString(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func escapePlainPath(path string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		"\t", `\t`,
		"\n", `\n`,
		"\r", `\r`,
	)

	return replacer.Replace(path)
}
