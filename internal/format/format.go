package format

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/richclement/tu/internal/report"
)

func JSON(scanReport report.ScanReport) ([]byte, error) {
	var buf bytes.Buffer

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
			"%s\t%d\t%s\t%s\t%s",
			formatNullableInt(result.Tokens),
			result.Bytes,
			formatNullableMethod(result.Method),
			result.Status,
			result.Path,
		))
	}

	return []byte(strings.Join(lines, "\n") + "\n")
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
