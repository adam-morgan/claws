package messages

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

func newMessage(id, sentMs, body string) types.Message {
	return types.Message{
		MessageId: aws.String(id),
		Body:      aws.String(body),
		Attributes: map[string]string{
			"SentTimestamp":           sentMs,
			"ApproximateReceiveCount": "2",
		},
	}
}

func TestCollectNew(t *testing.T) {
	seen := map[string]struct{}{}

	first := collectNew([]types.Message{newMessage("a", "1", ""), newMessage("b", "2", "")}, seen)
	second := collectNew([]types.Message{newMessage("b", "2", ""), newMessage("c", "3", "")}, seen)

	if len(first) != 2 || len(second) != 1 || *second[0].MessageId != "c" {
		t.Errorf("collectNew first=%d second=%+v", len(first), second)
	}
}

func TestSortNewestFirst(t *testing.T) {
	messages := []types.Message{
		newMessage("old", "1000", ""),
		newMessage("new", "3000", ""),
		newMessage("mid", "2000", ""),
		newMessage("unknown", "", ""),
	}

	sortNewestFirst(messages)

	var ids []string
	for _, m := range messages {
		ids = append(ids, *m.MessageId)
	}

	if got := strings.Join(ids, ","); got != "new,mid,old,unknown" {
		t.Errorf("sortNewestFirst order = %s", got)
	}
}

func TestBodyPreview(t *testing.T) {
	got := bodyPreview("line one\n\x1b[31mline\ttwo")

	if got != "line one line two" {
		t.Errorf("bodyPreview = %q", got)
	}

	long := bodyPreview(strings.Repeat("x", bodyPreviewLength+10))

	if !strings.HasSuffix(long, "…") || len([]rune(long)) != bodyPreviewLength+1 {
		t.Errorf("bodyPreview did not truncate: %d runes", len([]rune(long)))
	}
}

func TestRenderDetail(t *testing.T) {
	msg := newMessage("m-1", "1700000000000", `{"order":42,"note":"\u001b[31mred"}`)
	msg.Attributes["MessageGroupId"] = "group-a"
	msg.MessageAttributes = map[string]types.MessageAttributeValue{
		"trace": {DataType: aws.String("String"), StringValue: aws.String("abc")},
	}

	resource := NewMessageResource(msg, "https://sqs.us-east-1.amazonaws.com/1/dlq")
	detail := NewMessageRenderer().RenderDetail(resource)

	for _, want := range []string{"m-1", "Message Attributes", "trace", "abc", `"order": 42`, "group-a"} {
		if !strings.Contains(detail, want) {
			t.Errorf("RenderDetail missing %q", want)
		}
	}

	if got := getGroup(resource); got != "group-a" {
		t.Errorf("getGroup = %q", got)
	}
}

func TestFormatBodySanitizes(t *testing.T) {
	got := formatBody("plain\n\x1b]0;title\x07text")

	if strings.Contains(got, "\x1b") || !strings.Contains(got, "plain\ntext") {
		t.Errorf("formatBody = %q", got)
	}
}
