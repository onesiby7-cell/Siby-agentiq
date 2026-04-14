package executor

import (
	"bufio"
	"fmt"
	"strings"
)

const (
	FilePrefix    = "FILE:"
	EndFileMarker = "END_FILE"
	CreatePrefix  = "CREATE:"
	ModifyPrefix  = "MODIFY:"
	DeletePrefix  = "DELETE:"
)

type FilePlan struct {
	Path    string
	Action  string
	Content string
}

func ParseFilePlan(llmResponse string) []FilePlan {
	var plans []FilePlan
	var current *FilePlan
	var inBlock bool
	var content strings.Builder

	scanner := bufio.NewScanner(strings.NewReader(llmResponse))
	for scanner.Scan() {
		line := scanner.Text()
		upper := strings.ToUpper(strings.TrimSpace(line))

		switch {
		case strings.HasPrefix(upper, FilePrefix):
			if current != nil && content.Len() > 0 {
				current.Content = content.String()
				plans = append(plans, *current)
			}
			path := strings.TrimPrefix(line, "FILE:")
			path = strings.TrimSpace(path)
			current = &FilePlan{Path: path, Action: "modify"}
			content.Reset()
			inBlock = true

		case strings.HasPrefix(upper, CreatePrefix):
			path := strings.TrimPrefix(line, "CREATE:")
			path = strings.TrimSpace(path)
			plans = append(plans, FilePlan{Path: path, Action: "create"})
			inBlock = false

		case strings.HasPrefix(upper, ModifyPrefix):
			path := strings.TrimPrefix(line, "MODIFY:")
			path = strings.TrimSpace(path)
			plans = append(plans, FilePlan{Path: path, Action: "modify"})
			inBlock = false

		case strings.HasPrefix(upper, DeletePrefix):
			path := strings.TrimPrefix(line, "DELETE:")
			path = strings.TrimSpace(path)
			plans = append(plans, FilePlan{Path: path, Action: "delete"})
			inBlock = false

		case strings.HasPrefix(upper, EndFileMarker):
			if current != nil {
				current.Content = content.String()
				plans = append(plans, *current)
				current = nil
			}
			content.Reset()
			inBlock = false

		case inBlock && current != nil:
			if content.Len() > 0 {
				content.WriteString("\n")
			}
			content.WriteString(line)
		}
	}

	if current != nil && content.Len() > 0 {
		current.Content = content.String()
		plans = append(plans, *current)
	}

	return plans
}

func FormatFilePlan(plans []FilePlan) string {
	var sb strings.Builder
	for _, p := range plans {
		switch p.Action {
		case "create":
			sb.WriteString(fmt.Sprintf("CREATE: %s\n", p.Path))
		case "modify":
			sb.WriteString(fmt.Sprintf("MODIFY: %s\n", p.Path))
		case "delete":
			sb.WriteString(fmt.Sprintf("DELETE: %s\n", p.Path))
		}
	}
	return sb.String()
}

func (p FilePlan) String() string {
	return fmt.Sprintf("[%s] %s (%d bytes)", p.Action, p.Path, len(p.Content))
}
