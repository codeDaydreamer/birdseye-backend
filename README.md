# Birdseye backend

The API behind Birdseye, poultry farm management for farms in Kenya and across Africa. It serves the farmer web app
([birdseye-client](https://github.com/codeDaydreamer/birdseye-client)), the Android app
([birdseye-mobile](https://github.com/codeDaydreamer/birdseye-mobile)), the admin console
([birdseye-admin](https://github.com/codeDaydreamer/birdseye-admin)) and the public site's pricing
([birdseye-web](https://github.com/codeDaydreamer/birdseye-web)).

Go 1.23 · Gin · GORM · MySQL 8

## What it does

- **Farms as tenants.** A farm has staff in `owner`, `manager` and `worker` roles, invited by link. Every record
  belongs to a farm, and `FarmAuth` checks membership and role on each request, so removing someone takes effect
  immediately.
- **Farm records.** Batches (flocks), egg collections, bird movements, feed, weigh-ins, vaccinations, inventory,
  sales, expenses, budgets, hatchery (incubators, settings, chick bookings), contacts, reports (PDF/Excel) and
  live financials per batch and per month.
- **Billing.** Plans and promo codes set in the console, free trials, M-Pesa payments through
  [Dynapay](https://github.com/codeDaydreamer/dynapay) (STK push, signed webhooks, a reconciler for missed webhooks).
- **Notifications.** In-app over a websocket (`/ws`), browser push (VAPID) and Android push (Firebase), and email
  through Resend. Vaccination reminders run every 48 hours.
- **Admin API** (`/admin/*`, separate admin accounts). Platform insights (MRR, active farms, trial to paid,
  usage), farms and users, payments, an append-only audit log of every admin change, system health, and the
  integration keys below.

## Run it locally

```bash
docker run -d --name birdseye-mysql -e MYSQL_ROOT_PASSWORD=secret -e MYSQL_DATABASE=birdseye -p 3306:3306 mysql:8.0

cat > cmd/birdseye/.env <<'EOF'
DB_HOST=127.0.0.1:3306
DB_USER=root
DB_PASSWORD=secret
DB_NAME=birdseye
JWT_SECRET=a-long-random-string-of-at-least-32-characters
FRONTEND_URL=http://localhost:5173
EOF

go run ./cmd/birdseye        # http://localhost:8080; tables are created on start
```

Create the first admin with `POST /auth/admin/register`, then sign in to the console.

`docker compose up --build` runs the whole stack (MySQL, Postgres, Redis, this API, Dynapay, nginx, phpMyAdmin,
cAdvisor).

## Configuration

**Server environment** (needed to start):

| Variable | |
|---|---|
| `DB_HOST`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` | MySQL (`DB_HOST` is `host:port`) |
| `JWT_SECRET` | Signs user and admin tokens |
| `PORT` | Default `8080` |
| `FRONTEND_URL` | The web app's URL, for links in emails and Google sign-in redirects |
| `BASE_URL` | Public URL of this API, for uploaded file links |
| `SETTINGS_ENCRYPTION_KEY` | Optional: 32 bytes, base64. Encrypts keys saved in the console; without it the key is derived from `JWT_SECRET` |

**Integration keys.** Set these in the admin console under **Keys**, or as environment variables. A value saved in
the console wins, is stored encrypted, and takes effect without a restart. The console never shows saved secrets
again, only their last 4 characters, and saving needs the admin's password.

| Integration | Keys |
|---|---|
| Email (Resend) | `RESEND_API_KEY`, `EMAIL_FROM` |
| Browser push | `VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, `VAPID_SUBSCRIBER` (the console can generate a pair) |
| Android push | `FIREBASE_SERVICE_ACCOUNT_JSON` (or `FIREBASE_SERVICE_ACCOUNT_FILE` in the environment) |
| Google sign-in | `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URL`, `GOOGLE_ALLOWED_CLIENT_IDS` |
| Payments | `DYNAPAY_API_URL`, `DYNAPAY_CLIENT_ID`, `DYNAPAY_CLIENT_SECRET`, `DYNAPAY_WEBHOOK_SECRET` |

If you change `SETTINGS_ENCRYPTION_KEY` (or `JWT_SECRET` when that key isn't set), re-enter the keys in the console.

The web app reads the push key from `GET /push/vapid-public-key`, so a new key pair needs no client rebuild.

## Code layout

```
cmd/birdseye/main.go   wiring: config, migrations, routes, background jobs
pkg/api                Gin handlers, one SetupXRoutes per domain; routes.go has /auth and /admin
pkg/services           business logic, one file per domain (email/ for templates)
pkg/models             GORM models, auto-migrated on start
pkg/appconfig          integration keys from the console or the environment
pkg/middlewares        user, farm and admin auth
pkg/broadcast          websocket fan-out and push senders
pkg/db                 the database connection
deploy/                nightly MySQL/Postgres backups (systemd timer), see deploy/README.md
```

## Checks

```bash
go vet ./... && go build ./... && go test -race ./...
```

GitHub Actions runs the same on every push to `master` and on pull requests.

## Deploying

The API runs on a VPS behind nginx at `server.birdseye-poultry.com`, next to Dynapay. Set up the nightly backups
in [deploy/README.md](deploy/README.md) and keep `SETTINGS_ENCRYPTION_KEY` (or `JWT_SECRET`) with them: without it
the saved integration keys can't be read.
