# Payment Routing Layer (Juspay Hyperswitch)

This is an end-to-end payment routing service built on top of [Juspay Hyperswitch](https://hyperswitch.io/). It provides a robust backend layer with:

- **Smart Routing & Fallbacks:** Delegated to Hyperswitch (configured via their dashboard) to route transactions across PSPs based on success rate and latency.
- **Idempotency:** Redis-based idempotency to prevent duplicate payments.
- **Reconciliation:** A local PostgreSQL database tracks transactions and a `/reconcile` endpoint synchronizes status with Hyperswitch.
- **Webhooks:** Handler for Hyperswitch webhooks to asynchronously update transaction statuses.
- **Dockerized:** Fully containerized with Docker and Docker Compose.

## Architecture

1. **Node.js (Express, TypeScript):** The API server.
2. **PostgreSQL (Prisma ORM):** Stores local transaction records.
3. **Redis:** Manages API idempotency keys (caching responses for duplicate requests).
4. **Hyperswitch SDK/API:** The underlying payment orchestrator.

## Prerequisites & Setup

Ensure you have Docker and Docker Compose installed.

### 1. Environment Variables & API Keys

Before starting the application, you **MUST** configure your API keys.
Rename `.env.example` to `.env` and fill in the missing values.

```bash
cp .env.example .env
```

You need to add the following keys to your `.env` file:

- `HYPERSWITCH_API_KEY`: Obtain this from your [Hyperswitch Dashboard](https://app.hyperswitch.io/). Go to Settings -> API Keys.
- `HYPERSWITCH_WEBHOOK_SECRET`: Obtain this from the Webhooks section in the Hyperswitch Dashboard after configuring your webhook endpoint (e.g., `https://your-domain.com/webhooks/hyperswitch`).
- `HYPERSWITCH_API_URL`: Use `https://sandbox.hyperswitch.io` for testing, or the production URL for live traffic.

### 2. Hyperswitch Dashboard Configuration

For the **Smart Routing** based on latency and success rate to work, you must configure it in the Hyperswitch dashboard:
1. Navigate to **Routing** in your Hyperswitch dashboard.
2. Create a new **Volume / Smart Routing** rule.
3. Define the rules based on PSP success rates or latency thresholds. 
4. This application initiates a payment intent via the `/payments` API. Hyperswitch will automatically apply your configured routing rules to the transaction.

### 3. Running the Application

Build and start the containers using Docker Compose:

```bash
docker-compose up --build
```

This will start:
- PostgreSQL database on port 5432.
- Redis server on port 6379.
- Node.js API server on port 3000.

*Note: Prisma schema is automatically generated during the Docker build process. If you want to push the schema manually, you can run `docker-compose exec api npx prisma db push`.*

## API Endpoints

### 1. Create Payment
- **POST** `/payments/create`
- **Headers:** `Idempotency-Key: <unique-uuid>`
- **Body:**
```json
{
  "amount": 1000, 
  "currency": "INR",
  "paymentMethod": "UPI",
  "customerEmail": "customer@example.com"
}
```
*Note: Amount is typically in the lowest denomination (e.g., paise).*

### 2. Reconcile Payment
- **POST** `/payments/reconcile/:orderId`
- **Description:** Checks the current status of the payment in Hyperswitch and updates the local DB if they differ.

### 3. Webhook Handler
- **POST** `/webhooks/hyperswitch`
- **Description:** Receives async events from Hyperswitch (e.g., `payment_intent.succeeded`). Ensure this endpoint is publicly accessible and registered in your Hyperswitch dashboard.

## Next Steps for Production

- Implement webhook signature verification using the `HYPERSWITCH_WEBHOOK_SECRET`.
- Add a job queue (like BullMQ) for robust webhook retry logic and delayed reconciliation checks.
- Add comprehensive unit and integration tests.
