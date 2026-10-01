# IAM 权限

claws 需要适当的 IAM 权限才能访问 AWS 资源。所需权限取决于您要浏览的服务。

## 最低权限

进行基本的只读浏览时，claws 需要您要访问的服务的 `Describe*`、`List*` 和 `Get*` 权限。

## AI 聊天（可选）

AI 聊天功能（`A` 键）使用 Amazon Bedrock。要启用此功能，需要以下权限：

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "bedrock:InvokeModelWithResponseStream",
      "Resource": "arn:aws:bedrock:*::foundation-model/*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "aws-marketplace:Subscribe",
        "aws-marketplace:ViewSubscriptions"
      ],
      "Resource": "*"
    }
  ]
}
```

**注意**：AWS Marketplace 权限仅在账户中首次使用模型时需要。如果模型已启用，则只需要 `bedrock:InvokeModelWithResponseStream` 权限。

## 内联指标（可选）

要显示内联 CloudWatch 指标（使用 `M` 键切换），需要以下权限：

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "cloudwatch:GetMetricData",
      "Resource": "*"
    }
  ]
}
```

指标默认处于禁用状态。启用后，claws 会获取受支持资源（EC2、RDS、Lambda）最近一小时的指标数据。

## 资源操作

部分资源操作需要额外的权限：

| 操作 | 所需权限 |
|------|----------|
| 启动/停止 EC2 | `ec2:StartInstances`、`ec2:StopInstances` |
| 删除资源 | `<service>:Delete*` |
| SSO 登录 | `sso:*`（用于 SSO 配置文件） |
| 查看 SQS 消息（DLQ） | `sqs:ReceiveMessage`, `sqs:ListDeadLetterSourceQueues` |
| SQS DLQ 重新驱动 | `sqs:StartMessageMoveTask`, `sqs:ListMessageMoveTasks`, `sqs:CancelMessageMoveTask` |

## 推荐策略

包含指标和 AI 聊天的完整只读访问权限：

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:Describe*",
        "rds:Describe*",
        "lambda:List*",
        "lambda:Get*",
        "s3:List*",
        "s3:GetBucket*",
        "cloudwatch:GetMetricData",
        "iam:List*",
        "iam:Get*"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": "bedrock:InvokeModelWithResponseStream",
      "Resource": "arn:aws:bedrock:*::foundation-model/*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "aws-marketplace:Subscribe",
        "aws-marketplace:ViewSubscriptions"
      ],
      "Resource": "*"
    }
  ]
}
```

如需完整访问权限，可使用 `ReadOnlyAccess` 或 `ViewOnlyAccess` 等 AWS 托管策略。

## 只读模式

使用 `claws --read-only` 运行，或设置 `CLAWS_READ_ONLY=1`，即可禁用所有破坏性操作，不受 IAM 权限影响。
