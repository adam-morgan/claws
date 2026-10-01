package messages

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/dao"
	apperrors "github.com/clawscli/claws/internal/errors"
)

const (
	maxMessages         = 50
	maxEmptyReceives    = 3
	receiveBatchSize    = 10
	maxReceiveCallLimit = 15
)

// MessageDAO peeks at messages in an SQS queue without hiding them from consumers
type MessageDAO struct {
	dao.BaseDAO
	client *sqs.Client
}

// NewMessageDAO creates a new MessageDAO
func NewMessageDAO(ctx context.Context) (dao.DAO, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "new "+ServiceResourcePath+" dao")
	}

	return &MessageDAO{
		BaseDAO: dao.NewBaseDAO("sqs", "messages"),
		client:  sqs.NewFromConfig(cfg),
	}, nil
}

func (d *MessageDAO) List(ctx context.Context) ([]dao.Resource, error) {
	queueUrl := dao.GetFilterFromContext(ctx, "QueueUrl")
	if queueUrl == "" {
		return nil, fmt.Errorf("queue URL filter required")
	}

	var messages []types.Message
	seen := make(map[string]struct{})
	emptyReceives := 0

	for call := 0; call < maxReceiveCallLimit && len(messages) < maxMessages && emptyReceives < maxEmptyReceives; call++ {
		// VisibilityTimeout 0 keeps messages visible to real consumers while we peek.
		output, err := d.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:                    &queueUrl,
			MaxNumberOfMessages:         receiveBatchSize,
			VisibilityTimeout:           0,
			WaitTimeSeconds:             0,
			MessageSystemAttributeNames: []types.MessageSystemAttributeName{types.MessageSystemAttributeNameAll},
			MessageAttributeNames:       []string{"All"},
		})
		if err != nil {
			return nil, apperrors.Wrap(err, "receive messages")
		}

		fresh := collectNew(output.Messages, seen)
		messages = append(messages, fresh...)

		if len(fresh) == 0 {
			emptyReceives++
		} else {
			emptyReceives = 0
		}
	}

	sortNewestFirst(messages)

	if len(messages) > maxMessages {
		messages = messages[:maxMessages]
	}

	resources := make([]dao.Resource, len(messages))
	for i, msg := range messages {
		resources[i] = NewMessageResource(msg, queueUrl)
	}

	return resources, nil
}

func (d *MessageDAO) Get(ctx context.Context, id string) (dao.Resource, error) {
	resources, err := d.List(ctx)
	if err != nil {
		return nil, err
	}

	for _, r := range resources {
		if r.GetID() == id {
			return r, nil
		}
	}

	return nil, fmt.Errorf("message %s not found", id)
}

func (d *MessageDAO) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("delete not supported for sqs messages")
}

func collectNew(batch []types.Message, seen map[string]struct{}) []types.Message {
	var fresh []types.Message
	for _, msg := range batch {
		id := appaws.Str(msg.MessageId)
		if _, ok := seen[id]; ok {
			continue
		}

		seen[id] = struct{}{}
		fresh = append(fresh, msg)
	}
	return fresh
}

func sortNewestFirst(messages []types.Message) {
	sort.SliceStable(messages, func(i, j int) bool {
		return sentTimestamp(messages[i]).After(sentTimestamp(messages[j]))
	})
}

func sentTimestamp(msg types.Message) time.Time {
	ms, err := strconv.ParseInt(msg.Attributes[string(types.MessageSystemAttributeNameSentTimestamp)], 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.UnixMilli(ms)
}

// MessageResource wraps a received SQS message
type MessageResource struct {
	dao.BaseResource
	QueueUrl          string
	Body              string
	Attributes        map[string]string
	MessageAttributes map[string]types.MessageAttributeValue
	SentAt            time.Time
}

// NewMessageResource creates a new MessageResource
func NewMessageResource(msg types.Message, queueUrl string) *MessageResource {
	id := appaws.Str(msg.MessageId)

	return &MessageResource{
		BaseResource: dao.BaseResource{
			ID:   id,
			Name: id,
			Data: msg,
		},
		QueueUrl:          queueUrl,
		Body:              appaws.Str(msg.Body),
		Attributes:        msg.Attributes,
		MessageAttributes: msg.MessageAttributes,
		SentAt:            sentTimestamp(msg),
	}
}

// ReceiveCount returns how many times the message has been received
func (r *MessageResource) ReceiveCount() string {
	return r.Attributes[string(types.MessageSystemAttributeNameApproximateReceiveCount)]
}

// MessageGroupId returns the FIFO message group ID
func (r *MessageResource) MessageGroupId() string {
	return r.Attributes[string(types.MessageSystemAttributeNameMessageGroupId)]
}
