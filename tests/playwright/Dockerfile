# Playwright E2E Tests Docker Image
FROM mcr.microsoft.com/playwright:v1.40.0-jammy

# Set working directory
WORKDIR /tests

# Copy package files
COPY package.json ./

# Install dependencies
RUN npm install

# Copy test files
COPY . .

# Install Playwright browsers (already in base image, but ensure latest)
RUN npx playwright install

# Default command: run all tests
CMD ["npx", "playwright", "test"]

