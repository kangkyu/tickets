# Tickets by UMA

An e-ticketing platform for virtual events with Lightning Network payments via the UMA (Universal Money Address) protocol.

## Goal

Enable event organizers to sell tickets priced in Bitcoin (satoshis) and let attendees pay using UMA-compatible wallets (e.g. test.uma.me). The app uses **UMA Request** to push payment requests directly to the buyer's wallet.

## Features

### For Attendees
- Browse and search active events
- Purchase tickets with a UMA address (e.g. `$jimmy@test.uma.me`)
- Approve the payment in their UMA wallet when prompted
- View ticket status (pending, confirmed, expired)
- Download ticket QR code for event entry

### For Admins
- Create and manage virtual events (title, date, capacity, price in sats, stream URL)
- Monitor pending payments and retry failed ones
- View Lightning node balance
- Admin access controlled by email whitelist

### Payment Flow (UMA Request)
1. Buyer purchases a ticket on the website, providing their UMA address (e.g. `$jimmy@test.uma.me`)
2. Backend creates a UMA Invoice with the ticket amount and a callback URL
3. Backend discovers the buyer's VASP by fetching `https://test.uma.me/.well-known/uma-configuration`
4. Backend sends the UMA Invoice to the VASP's `uma_request_endpoint`
5. Buyer sees the payment request in their UMA wallet (test.uma.me) and approves it
6. The VASP (test.uma.me) POSTs a pay request to our callback URL
7. Our callback handler creates a Lightning invoice and returns it
8. The VASP pays the Lightning invoice
9. Lightspark webhook fires (`PAYMENT_FINISHED`) with an `IncomingPayment`
10. Backend matches the bolt11 to the ticket's payment record and marks the ticket as paid

In test mode, `CreateTestModePayment` can simulate step 5-8 for local development.

### Tech Stack
- **Frontend:** React (Vite), Tailwind CSS, deployed on AWS Amplify
- **Backend:** Go, Gorilla Mux, PostgreSQL, deployed on ECS Fargate
- **Payments:** Lightspark SDK, UMA Go SDK, UMA Request protocol
- **Infra:** Terraform (VPC, ALB, ECS, RDS, ECR, Amplify)

## Deploy

Backend first, then frontend (Amplify deploys on `git push`):

```sh
terraform apply -var-file="terraform.tfvars"

cd backend
docker build --platform linux/amd64 -t 800097198265.dkr.ecr.us-east-1.amazonaws.com/uma-tickets-staging/backend:latest .

# reauthenticate if needed
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin 800097198265.dkr.ecr.us-east-1.amazonaws.com

docker push 800097198265.dkr.ecr.us-east-1.amazonaws.com/uma-tickets-staging/backend:latest

aws ecs update-service --cluster uma-tickets-staging-cluster --service uma-tickets-staging-backend --force-new-deployment --region us-east-1

# check logs
aws logs tail "/ecs/uma-tickets-staging-backend" --region us-east-1 --since 10m

# frontend deploys automatically
git push origin master
```
