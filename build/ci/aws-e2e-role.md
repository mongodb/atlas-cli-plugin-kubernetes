# AWS IAM Role for E2E Tests

The e2e tests (under `test/e2e/`) create and destroy real AWS VPCs via the EC2 API. Rather than using static AWS credentials, the Evergreen CI pipeline uses `ec2.assume_role` to obtain temporary credentials.

## Role ARN

```
arn:aws:iam::358363220050:role/atlas-cli-plugin-kubernetes-evergreen-role
```

Set as the Evergreen project variable `aws_role_arn` (plaintext, not a secret).

## Permissions Policy

The role grants the minimum EC2 permissions needed for the e2e VPC lifecycle:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "VisualEditor0",
            "Effect": "Allow",
            "Action": [
                "ec2:CreateVpc",
                "ec2:CreateTags",
                "ec2:ModifyVpcAttribute",
                "ec2:DeleteVpc",
                "ec2:DescribeVpcs"
            ],
            "Resource": "arn:aws:ec2:eu-south-2:358363220050:vpc/*"
        }
    ]
}
```

`ec2:CreateTags` is required because `CreateVpcInput` includes `TagSpecifications` (a `Name` tag). `ec2:DescribeVpcs` covers SDK internal calls. All actions are scoped to **`eu-south-2`**, the only region the tests operate in.

## Trust Policy

The role can only be assumed by the Evergreen production instance role, restricted by an external ID condition that matches the Evergreen project:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "AWS": "arn:aws:iam::557821124784:role/evergreen.role.production"
            },
            "Action": "sts:AssumeRole",
            "Condition": {
                "StringLike": {
                    "sts:ExternalId": "67695d6afadbc80007e0c945-*"
                }
            }
        }
    ]
}
```

The external ID prefix `67695d6afadbc80007e0c945` is the Evergreen project ID.

## Usage in CI

In `build/ci/evergreen.yml`, the `"e2e test"` function calls `ec2.assume_role` before running the tests. The temporary credentials (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`) are then passed to the test subprocess via `include_expansions_in_env`.