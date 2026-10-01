# IAM 권한

claws는 AWS 리소스에 접근하기 위해 적절한 IAM 권한이 필요합니다. 필요한 권한은 탐색하려는 서비스에 따라 다릅니다.

## 최소 권한

기본적인 읽기 전용 탐색을 위해서는 접근하려는 서비스의 `Describe*`, `List*`, `Get*` 권한이 필요합니다.

## AI 채팅 (선택 사항)

AI 채팅 기능(`A` 키)은 Amazon Bedrock을 사용합니다. 이 기능을 활성화하려면 다음 권한이 필요합니다:

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

**참고**: AWS Marketplace 권한은 계정에서 처음 모델을 사용할 때 필요합니다. 모델이 이미 활성화되어 있는 경우 `bedrock:InvokeModelWithResponseStream` 권한만 필요합니다.

## 인라인 메트릭 (선택 사항)

인라인 CloudWatch 메트릭을 표시하려면(`M` 키로 전환) 다음 권한이 필요합니다:

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

메트릭은 기본적으로 비활성화되어 있습니다. 활성화하면 claws는 지원되는 리소스(EC2, RDS, Lambda)의 최근 1시간 메트릭을 가져옵니다.

## 리소스 액션

일부 리소스 액션에는 추가 권한이 필요합니다:

| 액션 | 필요한 권한 |
|--------|---------------------|
| EC2 시작/중지 | `ec2:StartInstances`, `ec2:StopInstances` |
| 리소스 삭제 | `<service>:Delete*` |
| SSO 로그인 | `sso:*` (SSO 프로필용) |
| SQS 메시지 보기 (DLQ) | `sqs:ReceiveMessage`, `sqs:ListDeadLetterSourceQueues` |
| SQS DLQ 리드라이브 | `sqs:StartMessageMoveTask`, `sqs:ListMessageMoveTasks`, `sqs:CancelMessageMoveTask` |

## 권장 정책

메트릭과 AI 채팅을 포함한 전체 읽기 전용 접근의 경우:

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

전체 접근 권한이 필요한 경우 `ReadOnlyAccess` 또는 `ViewOnlyAccess`와 같은 AWS 관리형 정책을 사용하십시오.

## 읽기 전용 모드

`--read-only` 플래그를 사용하거나 `CLAWS_READ_ONLY=1`을 설정하면 IAM 권한에 관계없이 모든 파괴적 액션이 비활성화됩니다.
