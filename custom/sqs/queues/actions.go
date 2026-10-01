package queues

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"

	sqsClient "github.com/clawscli/claws/custom/sqs"
	"github.com/clawscli/claws/internal/action"
	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	"github.com/clawscli/claws/internal/sanitize"
)

const moveTaskStatusRunning = "RUNNING"

func init() {
	// Register actions for SQS queues
	action.Global.Register("sqs", "queues", []action.Action{
		{
			Name:      "Purge Queue",
			Shortcut:  "p",
			Type:      action.ActionTypeAPI,
			Operation: "PurgeQueue",
			Confirm:   action.ConfirmSimple,
		},
		{
			Name:      "Send Test Message",
			Shortcut:  "s",
			Type:      action.ActionTypeAPI,
			Operation: "SendTestMessage",
			Confirm:   action.ConfirmSimple,
		},
		{
			Name:      "Start Redrive",
			Shortcut:  "r",
			Type:      action.ActionTypeAPI,
			Operation: "StartMessageMoveTask",
			Confirm:   action.ConfirmSimple,
			Filter:    isDLQ,
		},
		{
			Name:      "Redrive Status",
			Shortcut:  "R",
			Type:      action.ActionTypeAPI,
			Operation: "ListMessageMoveTasks",
			Filter:    isDLQ,
		},
		{
			Name:      "Cancel Redrive",
			Shortcut:  "c",
			Type:      action.ActionTypeAPI,
			Operation: "CancelMessageMoveTask",
			Confirm:   action.ConfirmSimple,
			Filter:    isDLQ,
		},
		{
			Name:      "Delete",
			Shortcut:  "D",
			Type:      action.ActionTypeAPI,
			Operation: "DeleteQueue",
			Confirm:   action.ConfirmDangerous,
		},
	})

	// Register executor
	action.RegisterExecutor("sqs", "queues", executeQueueAction)
}

// executeQueueAction executes an action on an SQS queue
func executeQueueAction(ctx context.Context, act action.Action, resource dao.Resource) action.ActionResult {
	switch act.Operation {
	case "PurgeQueue":
		return executePurgeQueue(ctx, resource)
	case "SendTestMessage":
		return executeSendTestMessage(ctx, resource)
	case "DeleteQueue":
		return executeDeleteQueue(ctx, resource)
	case "StartMessageMoveTask":
		return executeStartRedrive(ctx, resource)
	case "ListMessageMoveTasks":
		return executeRedriveStatus(ctx, resource)
	case "CancelMessageMoveTask":
		return executeCancelRedrive(ctx, resource)
	default:
		return action.UnknownOperationResult(act.Operation)
	}
}

func getSQSClient(ctx context.Context) (*sqs.Client, error) {
	return sqsClient.GetClient(ctx)
}

func executePurgeQueue(ctx context.Context, resource dao.Resource) action.ActionResult {
	queue, ok := resource.(*QueueResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	client, err := getSQSClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	queueUrl := queue.URL
	queueName := queue.GetName()

	input := &sqs.PurgeQueueInput{
		QueueUrl: &queueUrl,
	}

	_, err = client.PurgeQueue(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("purge queue: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Purged all messages from %s", queueName),
	}
}

func executeSendTestMessage(ctx context.Context, resource dao.Resource) action.ActionResult {
	queue, ok := resource.(*QueueResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	client, err := getSQSClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	queueUrl := queue.URL
	queueName := queue.GetName()
	messageBody := `{"test": true, "source": "claws"}`

	input := &sqs.SendMessageInput{
		QueueUrl:    &queueUrl,
		MessageBody: &messageBody,
	}

	// Add message group ID for FIFO queues
	if queue.IsFIFO() {
		groupId := "claws-test"
		dedupId := fmt.Sprintf("claws-test-%d", time.Now().UnixNano())
		input.MessageGroupId = &groupId
		input.MessageDeduplicationId = &dedupId
	}

	output, err := client.SendMessage(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("send message: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Sent test message to %s (ID: %s)", queueName, appaws.Str(output.MessageId)),
	}
}

func executeDeleteQueue(ctx context.Context, resource dao.Resource) action.ActionResult {
	queue, ok := resource.(*QueueResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	client, err := getSQSClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	queueUrl := queue.URL
	queueName := queue.GetName()

	input := &sqs.DeleteQueueInput{
		QueueUrl: &queueUrl,
	}

	_, err = client.DeleteQueue(ctx, input)
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("delete queue: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Deleted queue %s", queueName),
	}
}

func isDLQ(resource dao.Resource) bool {
	queue, ok := resource.(*QueueResource)
	return ok && queue.IsDLQ
}

func executeStartRedrive(ctx context.Context, resource dao.Resource) action.ActionResult {
	queue, ok := resource.(*QueueResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	client, err := getSQSClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	sourceArn := queue.GetARN()

	output, err := client.StartMessageMoveTask(ctx, &sqs.StartMessageMoveTaskInput{
		SourceArn: &sourceArn,
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("start message move task: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Started redrive of %s to source queue(s) (task: %s)", queue.GetName(), appaws.Str(output.TaskHandle)),
	}
}

func executeRedriveStatus(ctx context.Context, resource dao.Resource) action.ActionResult {
	queue, ok := resource.(*QueueResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	client, err := getSQSClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	tasks, err := listMoveTasks(ctx, client, queue.GetARN())
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	return action.ActionResult{
		Success: true,
		Message: formatMoveTasks(queue.GetName(), tasks),
	}
}

func executeCancelRedrive(ctx context.Context, resource dao.Resource) action.ActionResult {
	queue, ok := resource.(*QueueResource)
	if !ok {
		return action.InvalidResourceResult()
	}

	client, err := getSQSClient(ctx)
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	tasks, err := listMoveTasks(ctx, client, queue.GetARN())
	if err != nil {
		return action.ActionResult{Success: false, Error: err}
	}

	running := findRunningTask(tasks)
	if running == nil {
		return action.ActionResult{Success: true, Message: fmt.Sprintf("No running redrive for %s", queue.GetName())}
	}

	output, err := client.CancelMessageMoveTask(ctx, &sqs.CancelMessageMoveTaskInput{
		TaskHandle: running.TaskHandle,
	})
	if err != nil {
		return action.ActionResult{Success: false, Error: fmt.Errorf("cancel message move task: %w", err)}
	}

	return action.ActionResult{
		Success: true,
		Message: fmt.Sprintf("Cancelled redrive of %s (%d messages moved)", queue.GetName(), output.ApproximateNumberOfMessagesMoved),
	}
}

func listMoveTasks(ctx context.Context, client *sqs.Client, sourceArn string) ([]types.ListMessageMoveTasksResultEntry, error) {
	output, err := client.ListMessageMoveTasks(ctx, &sqs.ListMessageMoveTasksInput{
		SourceArn:  &sourceArn,
		MaxResults: aws.Int32(10),
	})
	if err != nil {
		return nil, fmt.Errorf("list message move tasks: %w", err)
	}

	return output.Results, nil
}

func findRunningTask(tasks []types.ListMessageMoveTasksResultEntry) *types.ListMessageMoveTasksResultEntry {
	for i := range tasks {
		if appaws.Str(tasks[i].Status) == moveTaskStatusRunning && tasks[i].TaskHandle != nil {
			return &tasks[i]
		}
	}
	return nil
}

func formatMoveTasks(queueName string, tasks []types.ListMessageMoveTasksResultEntry) string {
	if len(tasks) == 0 {
		return fmt.Sprintf("No redrive tasks found for %s", queueName)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Redrive tasks for %s:\n", queueName)

	for _, task := range tasks {
		progress := fmt.Sprintf("%d moved", task.ApproximateNumberOfMessagesMoved)
		if task.ApproximateNumberOfMessagesToMove != nil {
			progress = fmt.Sprintf("%d/%d moved", task.ApproximateNumberOfMessagesMoved, *task.ApproximateNumberOfMessagesToMove)
		}

		destination := "source queue(s)"
		if task.DestinationArn != nil {
			destination = queueNameFromArn(*task.DestinationArn)
		}

		started := time.UnixMilli(task.StartedTimestamp).Format("2006-01-02 15:04:05")

		fmt.Fprintf(&sb, "\n%s  %s  started %s  -> %s",
			sanitize.TerminalText(appaws.Str(task.Status)), progress, started, sanitize.TerminalText(destination))

		if reason := appaws.Str(task.FailureReason); reason != "" {
			fmt.Fprintf(&sb, "\n  failure: %s", sanitize.TerminalText(reason))
		}
	}

	return sb.String()
}
