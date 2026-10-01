package messages

import (
	"bytes"
	"encoding/json"
	"maps"
	"slices"
	"strings"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
	"github.com/clawscli/claws/internal/sanitize"
)

const bodyPreviewLength = 120

// MessageRenderer renders SQS messages
type MessageRenderer struct {
	render.BaseRenderer
}

// NewMessageRenderer creates a new MessageRenderer
func NewMessageRenderer() render.Renderer {
	return &MessageRenderer{
		BaseRenderer: render.BaseRenderer{
			Service:  "sqs",
			Resource: "messages",
			Cols: []render.Column{
				{Name: "MESSAGE ID", Width: 38, Getter: func(r dao.Resource) string { return r.GetID() }, Priority: 0},
				{Name: "SENT", Width: 20, Getter: getSent, Priority: 1},
				{Name: "RECEIVES", Width: 9, Getter: getReceives, Priority: 2},
				{Name: "GROUP", Width: 16, Getter: getGroup, Priority: 4},
				{Name: "BODY", Width: 60, Getter: getBody, Priority: 3},
			},
		},
	}
}

func getSent(r dao.Resource) string {
	m, ok := r.(*MessageResource)
	if !ok || m.SentAt.IsZero() {
		return ""
	}
	return m.SentAt.Format("2006-01-02 15:04:05")
}

func getReceives(r dao.Resource) string {
	if m, ok := r.(*MessageResource); ok {
		return m.ReceiveCount()
	}
	return ""
}

func getGroup(r dao.Resource) string {
	if m, ok := r.(*MessageResource); ok && m.MessageGroupId() != "" {
		return sanitize.TerminalText(m.MessageGroupId())
	}
	return "-"
}

func getBody(r dao.Resource) string {
	m, ok := r.(*MessageResource)
	if !ok {
		return ""
	}
	return bodyPreview(m.Body)
}

func bodyPreview(body string) string {
	preview := strings.Join(strings.Fields(sanitize.MultilineTerminalText(body)), " ")

	runes := []rune(preview)
	if len(runes) > bodyPreviewLength {
		return string(runes[:bodyPreviewLength]) + "…"
	}
	return preview
}

// RenderDetail renders detailed message information
func (r *MessageRenderer) RenderDetail(resource dao.Resource) string {
	m, ok := resource.(*MessageResource)
	if !ok {
		return ""
	}

	d := render.NewDetailBuilder()

	d.Title("SQS Message", m.GetID())

	d.Section("Basic Information")
	d.Field("Message ID", m.GetID())
	d.Field("Queue URL", m.QueueUrl)
	if sent := getSent(m); sent != "" {
		d.Field("Sent", sent)
	}
	d.FieldNonEmpty("Receive Count", m.ReceiveCount())

	d.Section("System Attributes")
	for _, key := range slices.Sorted(maps.Keys(m.Attributes)) {
		d.Field(key, sanitize.TerminalText(m.Attributes[key]))
	}

	if len(m.MessageAttributes) > 0 {
		d.Section("Message Attributes")

		for _, key := range slices.Sorted(maps.Keys(m.MessageAttributes)) {
			attr := m.MessageAttributes[key]
			value := ""
			if attr.StringValue != nil {
				value = *attr.StringValue
			} else if attr.BinaryValue != nil {
				value = "(binary)"
			}
			d.Field(sanitize.TerminalText(key), sanitize.TerminalText(value))
		}
	}

	d.Section("Body")
	d.Line(formatBody(m.Body))

	return d.String()
}

// RenderSummary returns summary fields for the header panel
func (r *MessageRenderer) RenderSummary(resource dao.Resource) []render.SummaryField {
	m, ok := resource.(*MessageResource)
	if !ok {
		return r.BaseRenderer.RenderSummary(resource)
	}

	fields := []render.SummaryField{
		{Label: "Message ID", Value: m.GetID()},
		{Label: "Sent", Value: getSent(m)},
		{Label: "Receive Count", Value: m.ReceiveCount()},
	}

	if group := m.MessageGroupId(); group != "" {
		fields = append(fields, render.SummaryField{Label: "Message Group", Value: sanitize.TerminalText(group)})
	}

	return fields
}

func formatBody(body string) string {
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(body), "", "  "); err == nil {
		body = buf.String()
	}
	return sanitize.MultilineTerminalText(body)
}
