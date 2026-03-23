package export

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/marianogappa/predictions-tracker/domain"
)

type ExportData struct {
	Prediction domain.Prediction
	Events     []domain.Event
	Values     []domain.CandleValue
	ChartPNG   []byte // optional, embedded if non-nil
}

func RenderMarkdown(data ExportData) (string, error) {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# Prediction: %s\n\n", data.Prediction.Statement))

	b.WriteString("## Metadata\n\n")
	b.WriteString(fmt.Sprintf("- **ID:** %s\n", data.Prediction.ID))
	b.WriteString(fmt.Sprintf("- **Asset:** %s\n", data.Prediction.Asset))
	b.WriteString(fmt.Sprintf("- **State:** %s\n", data.Prediction.State))
	b.WriteString(fmt.Sprintf("- **Created:** %s\n", data.Prediction.CreatedAt.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("- **Deadline:** %s\n", data.Prediction.Deadline.Format(time.RFC3339)))
	if data.Prediction.AuthorName != "" {
		b.WriteString(fmt.Sprintf("- **Author:** %s\n", data.Prediction.AuthorName))
	}
	if data.Prediction.AuthorURL != "" {
		b.WriteString(fmt.Sprintf("- **Author URL:** %s\n", data.Prediction.AuthorURL))
	}
	if data.Prediction.SourceURL != "" {
		b.WriteString(fmt.Sprintf("- **Source:** %s\n", data.Prediction.SourceURL))
	}
	b.WriteString("\n")

	b.WriteString("## Rule\n\n```json\n")
	ruleJSON, _ := json.MarshalIndent(data.Prediction.Rule, "", "  ")
	b.WriteString(string(ruleJSON))
	b.WriteString("\n```\n\n")

	b.WriteString("## Evaluation Result\n\n")
	b.WriteString(fmt.Sprintf("**Final State:** %s\n\n", data.Prediction.State))
	if data.Prediction.UpdatedAt.After(data.Prediction.CreatedAt) {
		b.WriteString(fmt.Sprintf("**Finalized At:** %s\n\n", data.Prediction.UpdatedAt.Format(time.RFC3339)))
	}

	if len(data.Events) > 0 {
		b.WriteString("## Event Log\n\n")
		b.WriteString("| Timestamp | Type |\n")
		b.WriteString("|-----------|------|\n")
		for _, e := range data.Events {
			b.WriteString(fmt.Sprintf("| %s | %s |\n", e.Timestamp.Format(time.RFC3339), e.Type))
		}
		b.WriteString("\n")
	}

	if len(data.Values) > 0 {
		b.WriteString("## Price Data\n\n")
		b.WriteString("| Timestamp | Open | High | Low | Close |\n")
		b.WriteString("|-----------|------|------|-----|-------|\n")
		for _, v := range data.Values {
			ts := time.Unix(v.Timestamp, 0).UTC().Format(time.RFC3339)
			b.WriteString(fmt.Sprintf("| %s | %.2f | %.2f | %.2f | %.2f |\n", ts, v.Open, v.High, v.Low, v.Close))
		}
		b.WriteString("\n")
	}

	if len(data.ChartPNG) > 0 {
		b.WriteString("## Chart\n\n")
		encoded := base64.StdEncoding.EncodeToString(data.ChartPNG)
		b.WriteString(fmt.Sprintf("![Price Chart](data:image/png;base64,%s)\n\n", encoded))
	}

	b.WriteString("## Integrity\n\n")
	eventsJSON, _ := json.Marshal(data.Events)
	hash := sha256.Sum256(eventsJSON)
	b.WriteString(fmt.Sprintf("**Event Log SHA256:** `%x`\n", hash))

	return b.String(), nil
}
