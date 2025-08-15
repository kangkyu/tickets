# AWS Amplify Deployment Guide

## Current Issue
The frontend is showing "Welcome - Your app will appear here once you complete your first deployment" which indicates a build or deployment issue.

## Solution Steps

### 1. Verify Build Configuration
The `amplify.yml` file is correctly configured to:
- Install dependencies in the `frontend` directory
- Build the project using `npm run build`
- Deploy from `frontend/dist` directory

### 2. Check Build Logs in AWS Amplify Console
1. Go to your AWS Amplify Console
2. Select your app
3. Go to the "Build" tab
4. Check the latest build logs for errors

### 3. Common Issues and Fixes

#### Issue: Build fails during npm install
**Fix**: Ensure `package-lock.json` is committed to git
```bash
cd frontend
npm install
git add package-lock.json
git commit -m "Add package-lock.json"
git push
```

#### Issue: Build fails during npm run build
**Fix**: The build works locally, so this might be a dependency issue
```bash
cd frontend
rm -rf node_modules package-lock.json
npm install
npm run build
git add .
git commit -m "Fix dependencies"
git push
```

#### Issue: Build succeeds but deployment fails
**Fix**: Check if the `dist` directory contains the expected files
```bash
cd frontend
npm run build
ls -la dist/
# Should show: index.html and assets/ directory
```

### 4. Environment Variables
Update your backend API URL in AWS Amplify Console:
1. Go to App settings > Environment variables
2. Add: `VITE_API_BASE_URL` = `https://your-backend-url.com/api`

### 5. Manual Build Test
Test the build process locally to ensure it works:
```bash
cd frontend
npm install
npm run build
# Check if dist/ directory is created with index.html
```

### 6. Force Rebuild
In AWS Amplify Console:
1. Go to the "Build" tab
2. Click "Trigger build" to force a new build
3. Monitor the build logs for any errors

## Expected Result
After a successful build and deployment, you should see:
- The UMA Ticket Platform interface
- Event list page
- Navigation menu
- No more "Welcome" placeholder message

## Backend Connection
Once the frontend is deployed:
1. Update the `VITE_API_BASE_URL` environment variable
2. Ensure your backend is accessible from the internet
3. Test the API endpoints from the deployed frontend

## Troubleshooting Commands
```bash
# Check if build works locally
cd frontend
npm run build

# Check build output
ls -la dist/

# Test the built files locally
cd dist
python3 -m http.server 8000
# Open http://localhost:8000 in browser
```
