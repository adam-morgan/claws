package queues

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/render"
)

func newQueue(name string, attrs map[string]string) *QueueResource {
	if attrs == nil {
		attrs = map[string]string{}
	}

	attrs["QueueArn"] = "arn:aws:sqs:us-east-1:123456789012:" + name

	return NewQueueResource("https://sqs.us-east-1.amazonaws.com/123456789012/"+name, attrs)
}

func TestParseRedrivePolicy(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantArn    string
		wantMax    int
		wantParsed bool
	}{
		{"valid", `{"deadLetterTargetArn":"arn:aws:sqs:us-east-1:1:dlq","maxReceiveCount":5}`, "arn:aws:sqs:us-east-1:1:dlq", 5, true},
		{"empty", "", "", 0, false},
		{"invalid json", "{", "", 0, false},
		{"missing target", `{"maxReceiveCount":5}`, "", 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy, ok := parseRedrivePolicy(tt.raw)

			if ok != tt.wantParsed || policy.DeadLetterTargetArn != tt.wantArn || policy.MaxReceiveCount != tt.wantMax {
				t.Errorf("parseRedrivePolicy(%q) = %+v, %v", tt.raw, policy, ok)
			}
		})
	}
}

func TestDeadLetterTargetArn(t *testing.T) {
	q := newQueue("src", map[string]string{
		"RedrivePolicy": `{"deadLetterTargetArn":"arn:aws:sqs:us-east-1:123456789012:src-dlq","maxReceiveCount":3}`,
	})

	if got := q.DeadLetterTargetArn(); got != "arn:aws:sqs:us-east-1:123456789012:src-dlq" {
		t.Errorf("DeadLetterTargetArn() = %q", got)
	}
}

func TestMarkDLQs(t *testing.T) {
	src := newQueue("src", map[string]string{
		"RedrivePolicy": `{"deadLetterTargetArn":"arn:aws:sqs:us-east-1:123456789012:src-dlq","maxReceiveCount":3}`,
	})
	dlq := newQueue("src-dlq", nil)
	other := newQueue("other", nil)

	markDLQs([]dao.Resource{src, dlq, other})

	if src.IsDLQ || other.IsDLQ {
		t.Errorf("non-DLQ queues marked as DLQ: src=%v other=%v", src.IsDLQ, other.IsDLQ)
	}

	if !dlq.IsDLQ {
		t.Error("src-dlq should be marked as DLQ")
	}
}

func TestQueueNavigations(t *testing.T) {
	renderer := NewQueueRenderer().(*QueueRenderer)

	src := newQueue("src", map[string]string{
		"RedrivePolicy": `{"deadLetterTargetArn":"arn:aws:sqs:us-east-1:123456789012:src-dlq","maxReceiveCount":3}`,
	})
	navs := renderer.Navigations(src)

	if len(navs) != 1 || navs[0].Resource != "queues" || navs[0].FilterValue != "src-dlq" {
		t.Errorf("source queue navigations = %+v", navs)
	}

	dlq := newQueue("src-dlq", nil)
	dlq.IsDLQ = true
	navs = renderer.Navigations(dlq)

	if len(navs) != 1 || navs[0].Resource != "messages" || navs[0].FilterValue != dlq.URL {
		t.Errorf("DLQ navigations = %+v", navs)
	}

	if navs := renderer.Navigations(newQueue("plain", nil)); len(navs) != 0 {
		t.Errorf("plain queue navigations = %+v", navs)
	}

	var _ render.Navigator = renderer
}

func TestIsDLQFilter(t *testing.T) {
	dlq := newQueue("dlq", nil)
	dlq.IsDLQ = true

	if !isDLQ(dlq) {
		t.Error("isDLQ should be true for DLQ")
	}

	if isDLQ(newQueue("plain", nil)) {
		t.Error("isDLQ should be false for plain queue")
	}
}

func TestFormatMoveTasks(t *testing.T) {
	if got := formatMoveTasks("dlq", nil); !strings.Contains(got, "No redrive tasks") {
		t.Errorf("empty format = %q", got)
	}

	tasks := []types.ListMessageMoveTasksResultEntry{
		{
			Status:                            aws.String("RUNNING"),
			ApproximateNumberOfMessagesMoved:  4,
			ApproximateNumberOfMessagesToMove: aws.Int64(10),
			StartedTimestamp:                  1700000000000,
		},
		{
			Status:                           aws.String("FAILED"),
			ApproximateNumberOfMessagesMoved: 1,
			DestinationArn:                   aws.String("arn:aws:sqs:us-east-1:123456789012:custom"),
			FailureReason:                    aws.String("Access\x1b[31m denied"),
		},
	}

	got := formatMoveTasks("dlq", tasks)

	for _, want := range []string{"RUNNING", "4/10 moved", "source queue(s)", "FAILED", "1 moved", "-> custom", "failure: Access denied"} {
		if !strings.Contains(got, want) {
			t.Errorf("formatMoveTasks missing %q in:\n%s", want, got)
		}
	}

	if strings.Contains(got, "\x1b") {
		t.Error("formatMoveTasks output contains escape sequence")
	}
}

func TestFindRunningTask(t *testing.T) {
	tasks := []types.ListMessageMoveTasksResultEntry{
		{Status: aws.String("COMPLETED"), TaskHandle: aws.String("old")},
		{Status: aws.String("RUNNING"), TaskHandle: aws.String("live")},
	}

	if got := findRunningTask(tasks); got == nil || *got.TaskHandle != "live" {
		t.Errorf("findRunningTask() = %+v", got)
	}

	if got := findRunningTask(tasks[:1]); got != nil {
		t.Errorf("findRunningTask() without running = %+v", got)
	}
}
