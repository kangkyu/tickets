# Tickets

Deploy backend first, and then frontend (AWS Amplify)

```sh
terraform apply -var-file="terraform.tfvars"

cd backend
docker build --platform linux/amd64 -t 800097198265.dkr.ecr.us-east-1.amazonaws.com/uma-tickets-staging/backend:latest .

# as needed, reauthenticate
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin 800097198265.dkr.ecr.us-east-1.amazonaws.com

docker push 800097198265.dkr.ecr.us-east-1.amazonaws.com/uma-tickets-staging/backend:latest

aws ecs update-service --cluster uma-tickets-staging-cluster --service uma-tickets-staging-backend --force-new-deployment --region us-east-1

aws logs tail "/ecs/uma-tickets-staging-backend" --region us-east-1 --since 10m

# deploys to Amplify frontend automatically
git push origin master
```

### TODO
- Add certificate for HTTPS backend
- Pay and get paid on testnet
