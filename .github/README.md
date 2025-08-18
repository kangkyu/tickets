# GitHub Actions Setup Guide

This guide explains how to set up the GitHub Actions workflows for automated backend deployment.

## Workflows

### 1. Backend CI/CD Pipeline (`backend-ci-cd.yml`)

**Triggers:**
- Push to `main` or `master` branch (backend changes only)
- Pull requests to `main` or `master` branch
- Manual workflow dispatch

**Jobs:**
1. **Test** - Runs Go tests and builds binary
2. **Build and Push** - Builds Docker image and pushes to ECR
3. **Deploy** - Deploys to ECS (main/master only)

**ECR Repository:** `800097198265.dkr.ecr.us-east-1.amazonaws.com/uma-tickets-staging/backend`

**ECS Configuration:**
- **Cluster**: `uma-tickets-staging-cluster`
- **Service**: `uma-tickets-staging-backend`

## Required GitHub Secrets

You need to add these secrets in your GitHub repository:

### AWS Credentials
```
AWS_ACCESS_KEY_ID=your_aws_access_key
AWS_SECRET_ACCESS_KEY=your_aws_secret_access_key
```

### How to Add Secrets:
1. Go to your GitHub repository
2. Click **Settings** → **Secrets and variables** → **Actions**
3. Click **New repository secret**
4. Add each secret with the exact names above

## AWS IAM Permissions

The AWS user needs these permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ecr:GetAuthorizationToken",
        "ecr:BatchCheckLayerAvailability",
        "ecr:GetDownloadUrlForLayer",
        "ecr:BatchGetImage",
        "ecr:PutImage",
        "ecr:InitiateLayerUpload",
        "ecr:UploadLayerPart",
        "ecr:CompleteLayerUpload"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "ecs:UpdateService",
        "ecs:DescribeServices",
        "ecs:DescribeTasks",
        "ecs:ListTasks"
      ],
      "Resource": "*"
    }
  ]
}
```

## How It Works

### 1. **Test Stage**
- Runs on Ubuntu (x86_64)
- Installs Go 1.22
- Runs `go test ./...`
- Builds binary to verify compilation

### 2. **Build Stage**
- Uses Docker Buildx for multi-platform support
- Builds specifically for `linux/amd64` (AWS ECS compatible)
- Pushes to ECR with multiple tags:
  - SHA-based tag (e.g., `abc123`)
  - Branch name tag (e.g., `main`)
  - `latest` tag

### 3. **Deploy Stage**
- Only runs on main/master pushes
- Forces new ECS deployment
- Waits for deployment to complete
- Verifies service is running

## Benefits of This Approach

✅ **No more cross-compilation issues** - Built on Linux x86_64  
✅ **Automated testing** - Catches issues before deployment  
✅ **Consistent builds** - Same environment every time  
✅ **Easy rollbacks** - Can deploy specific image tags  
✅ **Production ready** - Built specifically for AWS ECS  

## Manual Deployment

To manually trigger a deployment:

1. Go to **Actions** tab in GitHub
2. Select **Backend CI/CD Pipeline**
3. Click **Run workflow**
4. Choose branch and click **Run workflow**

## Troubleshooting

### Common Issues:

1. **AWS credentials error**
   - Verify secrets are set correctly
   - Check IAM permissions

2. **ECR push failed**
   - Ensure ECR repository exists
   - Check ECR permissions

3. **ECS deployment failed**
   - Verify cluster and service names
   - Check ECS permissions

### Logs:
- Check the **Actions** tab for detailed logs
- Each step shows output and any errors
- Failed jobs can be re-run individually

## Next Steps

1. **Add the required secrets** to your GitHub repository
2. **Push to main/master** to trigger the first deployment
3. **Monitor the Actions tab** to see the workflow in action
4. **Check ECS** to verify the deployment

The workflow will automatically build and deploy your backend every time you push changes to the main branch!
