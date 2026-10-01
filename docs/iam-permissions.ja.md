# IAM権限

clawsはAWSリソースにアクセスするために適切なIAM権限が必要です。必要な権限は、閲覧するサービスによって異なります。

## 最小権限

基本的な読み取り専用の閲覧には、アクセスするサービスの`Describe*`、`List*`、`Get*`権限が必要です。

## AIチャット（オプション）

AIチャット機能（`A`キー）はAmazon Bedrockを使用します。この機能を有効にするには、以下の権限が必要です：

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

**注意**: AWS Marketplace権限は、アカウントで初めてモデルを使用する際に必要です。モデルが既に有効化されている場合は、`bedrock:InvokeModelWithResponseStream`権限のみ必要です。

## インラインメトリクス（オプション）

インラインCloudWatchメトリクスを表示するには（`M`キーで切り替え）、以下の権限が必要です：

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

メトリクスはデフォルトで無効です。有効にすると、clawsは対応リソース（EC2、RDS、Lambda）の直近1時間のメトリクスを取得します。

## リソースアクション

一部のリソースアクションには追加の権限が必要です：

| アクション | 必要な権限 |
|--------|---------------------|
| EC2の起動/停止 | `ec2:StartInstances`, `ec2:StopInstances` |
| リソースの削除 | `<service>:Delete*` |
| SSOログイン | `sso:*`（SSOプロファイル用） |
| SQSメッセージの表示（DLQ） | `sqs:ReceiveMessage`, `sqs:ListDeadLetterSourceQueues` |
| SQS DLQリドライブ | `sqs:StartMessageMoveTask`, `sqs:ListMessageMoveTasks`, `sqs:CancelMessageMoveTask` |

## 推奨ポリシー

メトリクスとAIチャットを含む完全な読み取り専用アクセスの場合：

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

完全なアクセスには、`ReadOnlyAccess`や`ViewOnlyAccess`などのAWSマネージドポリシーを使用してください。

## 読み取り専用モード

`claws --read-only`で実行するか、`CLAWS_READ_ONLY=1`を設定すると、IAM権限に関係なくすべての破壊的アクションが無効になります。
