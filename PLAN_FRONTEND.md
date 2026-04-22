# Phase 5: Frontend Dashboard — Master Checklist
**Stack:** React 18 + Vite + TypeScript + TanStack Query + Zustand + Recharts  
**UI Layer:** Tailwind CSS + shadcn/ui  
**Forms:** React Hook Form + Zod  
**Anti-Bot:** Cloudflare Turnstile  
**Notifications:** Sonner  

---

## Section 1 — Infrastructure & Project Scaffold

### 1.1 Project Initialization
- [x] Scaffold project: `npm create vite@latest . -- --template react-ts`
- [x] Configure `tsconfig.json` path aliases (`@/api`, `@/stores`, `@/components`, `@/hooks`, `@/lib`)
- [x] Configure `vite.config.ts` dev proxy: forward `/api` → `http://localhost:8080` to eliminate CORS in development

### 1.2 Dependency Installation
- [x] **Routing:** `react-router-dom`
- [x] **Server state / polling:** `@tanstack/react-query` + `@tanstack/react-query-devtools`
- [x] **Client state:** `zustand`
- [x] **HTTP client:** `axios`
- [x] **Form engine:** `react-hook-form`
- [x] **Schema validation:** `zod` + `@hookform/resolvers`
- [x] **UI primitives:** `tailwindcss` + `@tailwindcss/vite` (Vite plugin)
- [x] **Component library:** `shadcn/ui` — init via `npx shadcn@latest init`, configure `components.json`
- [x] **Toast notifications:** `sonner`
- [x] **Stripe (client-side):** `@stripe/stripe-js` + `@stripe/react-stripe-js`
- [x] **Anti-bot:** `@marsidev/react-turnstile`

### 1.3 shadcn/ui Component Installation
Install only components actually used — add more as needed:
- [x] `npx shadcn@latest add button card input label skeleton badge separator`
- [x] `npx shadcn@latest add dialog alert-dialog sheet` (modals and drawers)
- [x] `npx shadcn@latest add table` (transaction history data table)
- [x] `npx shadcn@latest add slider` (risk tolerance input)
- [x] `npx shadcn@latest add tabs` (dashboard section switching)
- [x] `npx shadcn@latest add alert` (inline form-level errors)
- [x] `npx shadcn@latest add tooltip` (chart data point labels)
- [x] `npx shadcn@latest add progress` (onboarding questionnaire step indicator)
- [x] `npx shadcn@latest add radio-group` (multiple-choice questionnaire answers)
- [x] `npx shadcn@latest add checkbox` (chart toggles, e.g. Net Contributions line)
- [x] `npx shadcn@latest add chart` (shadcn recharts wrapper with native themeing)
- [x] `npx shadcn@latest add sonner` (shadcn sonner wrapper with native themeing)


### 1.4 Global Providers (`main.tsx`)
- [x] Wrap app in `QueryClientProvider` (TanStack Query)
- [x] Mount `<Toaster />` from Sonner at root level (single instance, outside routing)
- [x] Configure `QueryClient` defaults: `retry: 1`, `staleTime: 30_000`

### 1.5 Go Backend Modification (Pre-Frontend)

**Security & Infrastructure Updates**
- [x] Modify `LoginHandler`, `Verify2FAHandler`, `RefreshTokenHandler` to set `refresh_token` as an `httpOnly + Secure + SameSite=Strict` cookie scoped to `Path: /api/v1/refresh-token`
- [x] Remove `refresh_token` from all JSON response bodies
- [x] Verify `CORSMiddleware` has `AllowCredentials: true` and `AllowHeaders` includes `Authorization`
- [x] Add Go catch-all `NoRoute` handler to serve `./frontend/index.html` for SPA client-side routing

**Onboarding API Suite**
- [x] `GET /api/v1/onboarding/questions`: Fetch the dynamic list of onboarding questions and their available options.
- [x] `POST /api/v1/onboarding/submit`: Accept selected answer IDs, calculate `riskTolerance` and `investmentHorizon` server-side, and update the User profile.

**Portfolio & Data APIs**
- [x] `GET /api/v1/portfolio/history?range=`: Aggregate `DailyMarketData` (for 1M, 6M, 1Y, YTD, 5Y with one value per day granularity) and `IntradayMarketData` (for 1D, 1W with 15 minute granularity) and user holdings (from `Funding`) to return time-series data (Date, Portfolio Value, Net Contributions) for requested ranges (1D, 1W, 1M, 6M, 1Y, YTD, 5Y).
- [x] `GET /api/v1/transactions`: Execute a unified query (e.g., using UNION) across the `Funding` and `Transaction` tables to return a single, paginated, and chronologically sorted list of all financial events (DEPOSIT, CASHOUT, INVEST, SELL).
- [x] `GET /api/v1/portfolio`: Return the user's currently active `InvestmentRound` and its associated `Holding` array. Calculate and include the live `TotalPortfolioValue` and `AllTimeReturn` by fetching the latest market prices for each holding from `IntradayMarketData`, enriching the payload for the Allocation Pie Chart and Dashboard Header;

### 1.6 Environment Variables (`.env`)
- [ ] `VITE_API_BASE_URL` — production Go VPS base URL
- [x] `VITE_STRIPE_PUBLISHABLE_KEY` — Stripe `pk_test_...` key
- [ ] `VITE_TURNSTILE_SITE_KEY` — Cloudflare Turnstile site key
- [x] Create `.env.example` committed to repo; `.env` added to `.gitignore`

### 1.7 Zod Schema Library (`src/lib/schemas.ts`)
Define and export all form validation schemas in one file — single source of truth for client-side rules, mirroring Go's `binding` tags:
- [x] `registerSchema` — email (valid format), password (min 8, complexity rules matching Go's validator)
- [x] `loginSchema` — email, password (required)
- [x] `verify2FASchema` — token (exactly 6 digits, numeric)
- [x] `enable2FASchema` — token (exactly 6 digits, numeric)
- [x] `forgotPasswordSchema` — email (valid format)
- [x] `resetPasswordSchema` — token (required), newPassword (min 8, complexity), confirmPassword (must match)
- [x] `onboardingSchema` — all questionnaire answer fields (required, each maps to a discrete option set)
- [x] `depositSchema` — amount (number, greater than 0)
- [x] `cashoutSchema` — amount (number, greater than 0)
- [x] `investSchema` — amount (number, greater than 0)
- [x] `forecastSchema` — initialInvestment (min 0), monthlyContribution (min 0), years (integer 1–50)

---

## Section 2 — API Client Layer

### 2.1 Axios Instance (`src/api/client.ts`)
- [ ] Create singleton `axios` instance with `baseURL`, `withCredentials: true` (required for httpOnly cookie to be sent)
- [ ] **Request interceptor:** attach `Authorization: Bearer {accessToken}` from Zustand store on every outgoing request
- [ ] **Response interceptor — 401 handling:**
  - Check if a refresh is already in-flight (`isRefreshing` flag)
  - If yes: push original request into a `requestQueue` promise and wait
  - If no: set `isRefreshing = true`, call `POST /api/v1/refresh-token` (browser sends httpOnly cookie automatically)
    - **200:** update `accessToken` in Zustand, flush queue with new token, retry original request
    - **409 Conflict** (`"Concurrent request detected"`): wait 500ms, retry refresh exactly once
    - **401 on refresh:** clear Zustand store, redirect to `/login` via `window.location`
    - **"Token reuse detected" message:** clear store, redirect to `/login`, set a `securityAlert: true` flag in Zustand so the Login page renders the security warning banner
- [ ] **Response interceptor — 429 handling:** call `toast.error("Too many requests. Please slow down.")` via Sonner and reject the promise
- [ ] **Response interceptor — 423/403 Account Lockout:** call `toast.error("Account temporarily locked. Try again in 15 minutes.")` and reject
- [ ] **Response interceptor — 5xx handling:** call `toast.error("Server error. Please try again shortly.")` for unhandled 500+ responses
- [ ] Export typed API call functions from domain modules: `src/api/auth.ts`, `src/api/user.ts`, `src/api/portfolio.ts`, `src/api/forecast.ts`

### 2.2 Auth API (`src/api/auth.ts`)
- [ ] `register(email, password, turnstileToken)` → `POST /api/v1/register`
- [ ] `verifyEmail(token)` → `GET /api/v1/verify-email?token=`
- [ ] `login(email, password, turnstileToken)` → `POST /api/v1/login`
- [ ] `verify2FA(email, password, totpToken)` → `POST /api/v1/verify-2fa`
- [ ] `logout(refreshToken)` → `POST /api/v1/logout`
- [ ] `refreshToken()` → `POST /api/v1/refresh-token` (no body — browser sends httpOnly cookie)
- [ ] `forgotPassword(email, turnstileToken)` → `POST /api/v1/forgot-password`
- [ ] `resetPassword(token, newPassword)` → `POST /api/v1/reset-password`
- [ ] `setup2FA()` → `GET /api/v1/2fa/setup` (protected)
- [ ] `enable2FA(token)` → `POST /api/v1/2fa/enable` (protected)

### 2.3 User API (`src/api/user.ts`)
- [ ] `getUser()` → `GET /api/v1/user`
- [ ] `updateProfile(riskTolerance, investmentHorizon)` → `PUT /api/v1/user/profile`
- [ ] `deposit(amount)` → `POST /api/v1/deposit`
- [ ] `cashout(amount)` → `POST /api/v1/cashout`
- [ ] `createDepositIntent(amount)` → `POST /api/v1/deposit/intent`

### 2.4 Portfolio API (`src/api/portfolio.ts`)
- [ ] `invest(amount)` → `POST /api/v1/invest`

### 2.5 Forecast API (`src/api/forecast.ts`)
- [ ] `requestForecast(payload)` → `POST /api/v1/forecast` → returns `{ task_id }`
- [ ] `getForecastStatus(taskId)` → `GET /api/v1/forecast/status/:task_id` → returns `{ status, payload? }`

---

## Section 3 — Auth Store & Route Guards

### 3.1 Zustand Auth Store (`src/stores/authStore.ts`)
- [ ] State shape: `{ accessToken, user, status, securityAlert }`
  - `status`: `"loading" | "authenticated" | "unauthenticated"`
  - `securityAlert`: `boolean` — set true when token reuse is detected; Login page reads and renders the warning banner, then resets it
- [ ] Actions: `setAccessToken`, `setUser`, `setStatus`, `setSecurityAlert`, `clearAuth`

### 3.2 Silent Token Restore (`src/hooks/useSilentRestore.ts`)
- [ ] On app mount (`useEffect` with empty deps), call `POST /refresh-token`
- [ ] On success: store new `accessToken` in Zustand, call `GET /user` to populate user profile, set `status = "authenticated"`
- [ ] On failure: set `status = "unauthenticated"` (do not redirect here — let `ProtectedRoute` handle it)
- [ ] Keep `status = "loading"` during this entire sequence so the UI shows a skeleton, never a flash of unauthenticated content

### 3.3 Protected Route Guard (`src/components/layout/ProtectedRoute.tsx`)
- [ ] If `status === "loading"`: render full-page skeleton loader
- [ ] If `status === "unauthenticated"`: `<Navigate to="/login" replace />`
- [ ] If `status === "authenticated"`: render `<Outlet />`
- [ ] Bonus: if `user.riskTolerance === 0` (profile not yet set up), redirect to `/onboarding`

---

## Section 4 — Authentication Pages & Flows

All auth forms use `react-hook-form` + `zod` via `zodResolver`. Error messages render inline beneath each field using shadcn/ui `<FormMessage>`. Turnstile widget is mounted where indicated.

### 4.1 Register Page (`/register`)
- [ ] Fields: Email, Password — validated by `registerSchema`
- [ ] Mount `<Turnstile siteKey={VITE_TURNSTILE_SITE_KEY} onSuccess={setToken} />` above submit button
- [ ] On submit: call `register(email, password, turnstileToken)`; disable submit until Turnstile token is present
- [ ] On success (`200`): navigate to a `/register-success` static page ("Check your inbox")
- [ ] On error (`409` email conflict): set field-level error on email field via `form.setError`
- [ ] On Turnstile failure or missing token: show `toast.error("Anti-bot check failed. Please try again.")`

### 4.2 Verify Email Page (`/verify-email`)
- [ ] Read `?token=` from URL via `useSearchParams`
- [ ] On mount, call `verifyEmail(token)` — no user interaction needed
- [ ] Show loading state, then success ("Email verified. You can log in.") or error ("Link invalid or expired.")
- [ ] Success state shows a button linking to `/login`

### 4.3 Login Page (`/login`)
- [ ] **Security Alert Banner:** if `authStore.securityAlert === true`, render a prominent destructive `<Alert>` at the top: "Your session was invalidated due to suspicious activity. Please log in again." Reset `securityAlert` to false after display.
- [ ] Fields: Email, Password — validated by `loginSchema`
- [ ] Mount `<Turnstile />` widget; disable submit until token is resolved
- [ ] On submit: call `login(email, password, turnstileToken)`
  - `200` with `status: "success"`: store `accessToken`, fetch user, navigate to `/dashboard`
  - `200` with `status: "2fa_required"`: transition local state to `"2fa"` step (do not navigate)
  - `401`: set form-level error "Invalid email or password"
  - `429`: interceptor handles toast; additionally disable form for 5 seconds
- [ ] **2FA Gate (conditional render within same page):**
  - Renders when local state === `"2fa"`; shows the original email for context
  - Field: TOTP code (6-digit numeric) — validated by `verify2FASchema`
  - On submit: call `verify2FA(email, password, totpToken)` — re-sends credentials because the Go handler requires them
  - On success: store `accessToken`, navigate to `/dashboard`
  - On `401`: inline error "Invalid code"
  - "Back" link resets local state to `"credentials"`
- [ ] "Forgot password?" link navigates to `/forgot-password`

### 4.4 Forgot Password Page (`/forgot-password`)
- [ ] Field: Email — validated by `forgotPasswordSchema`
- [ ] Mount `<Turnstile />` widget
- [ ] On submit: call `forgotPassword(email, turnstileToken)`
- [ ] Always show success message regardless of response (Go intentionally obscures whether email exists): "If an account with that email exists, a reset link has been sent."

### 4.5 Reset Password Page (`/reset-password`)
- [ ] Read `?token=` from URL via `useSearchParams`
- [ ] Fields: New Password, Confirm Password — validated by `resetPasswordSchema` (passwords must match via Zod `.refine()`)
- [ ] On submit: call `resetPassword(token, newPassword)`
- [ ] `200`: show success message and link to `/login`
- [ ] `400` (token invalid/expired): show error with link to `/forgot-password`

### 4.6 Logout
- [ ] Logout `<Button>` in sidebar/header calls `logout()` (fires-and-forgets — Go will delete the session server-side)
- [ ] Immediately call `clearAuth()` on Zustand store regardless of API response
- [ ] Navigate to `/login`

---

## Section 5 — 2FA Management

### 5.1 2FA Setup Flow (within Settings/Security page)
- [ ] `GET /2fa/setup` returns `{ secret, uri, qr_code_b64 }`
- [ ] Render QR code: `<img src={qr_code_b64} alt="Scan with authenticator app" />` (Go already prepends `data:image/png;base64,`)
- [ ] Display plaintext secret below QR for manual entry
- [ ] Field: TOTP confirmation code — validated by `enable2FASchema`
- [ ] On submit: call `enable2FA(token)`; on success show `toast.success("2FA enabled successfully")`
- [ ] Error `400` (invalid code): inline field error "Incorrect code. Authenticator not linked."
- [ ] Error `400` (already enabled): show info state "2FA is already active on your account"

---

## Section 6 — Dashboard Shell

### 6.1 App Shell Layout (`src/components/layout/AppShell.tsx`)
- [ ] Sidebar with navigation links: Dashboard, Portfolio, Forecast, Settings
- [ ] Sidebar top: App Logo and Brand Name (acts as home link)
- [ ] Header: wallet balance badge (live query, `staleTime: 10_000`), user email, Logout button
- [ ] Mobile-responsive: sidebar collapses to a `<Sheet>` drawer on small screens (shadcn/ui `<Sheet>`)
- [ ] Active route highlighted in sidebar using `NavLink` from `react-router-dom`

### 6.2 Dashboard Overview Page (`/dashboard`)
- [ ] Wallet balance card (from `GET /user`)
- [ ] Quick action buttons: Deposit, Invest, Cashout (open shadcn/ui `<Dialog>` modals)
- [ ] Portfolio allocation chart preview (teaser `<AllocationPie />` linking to full `/portfolio` page)
- [ ] Onboarding callout card: if `user.riskTolerance === 0`, show a prominent prompt to complete financial profile

---

## Section 7 — Onboarding Questionnaire & Settings

### 7.1 Multi-Step Onboarding Questionnaire (`/onboarding`)

The onboarding flow replaces the client-side scoring logic with a dynamic, backend-driven architecture. The frontend acts purely as a presentation layer that fetches questions, collects user responses, and submits the selected answer IDs to the server for evaluation.

**Dynamic Fetching & State Initialization**
- [ ] On component mount, call `GET /api/v1/onboarding/questions` to fetch the complete JSON payload of the questionnaire.
- [ ] Initialize a local state (or `react-hook-form` state) to store the user's selected option ID for each question ID (e.g., `{ q1: "opt_3", q2: "opt_1" }`).

**Dynamic UI Rendering & Navigation**
- [ ] Render a shadcn/ui `<Progress>` bar at the top showing the current step based on the total number of questions fetched.
- [ ] Iterate over the fetched JSON payload to dynamically render each question prompt and its corresponding options using shadcn/ui `<RadioGroup>`.
- [ ] Implement "Back" and "Next" buttons to navigate between steps sequentially; hide "Back" on the first step.
- [ ] Disable the "Next" button until the user selects an answer for the current step's question.
- [ ] Preserve all intermediate answers in the frontend state without making API calls until the final submission.

**Submission & Backend Scoring (Final Step)**
- [ ] On the final question step, swap the "Next" button for a "Calculate My Profile" button.
- [ ] On click, submit the answers payload (`{ answers: { "q1": "opt_3", [...] } }`) via `POST /api/v1/onboarding/submit`.
- [ ] Handle loading states during the API call (e.g., show a spinner and disable the button).
- [ ] Ensure the backend independently computes both `riskTolerance` and `investmentHorizon` and persists them to the database.

**Success Summary Screen**
- [ ] Upon a successful `POST` response, transition the UI to a "Summary Screen".
- [ ] Extract the newly calculated `riskTolerance` (e.g., 3) and `investmentHorizon` (e.g., 18 years) from the backend response.
- [ ] Display these results clearly to the user (e.g., "Your Risk Level is 3" and "Your Investment Horizon is 18 years") along with a brief, plain-language description of what this means for their portfolio.
- [ ] Provide a "Go to Dashboard" button to navigate the user to `/dashboard`.
- [ ] On API error during submission, display a `toast.error(...)` and keep the user on the final step so their answers are not lost.

### 7.2 Profile Edit (within `/settings`)
- [ ] Allow the user to view their current `riskTolerance` and `investmentHorizon` read-only within the settings page.
- [ ] Provide a "Retake Questionnaire" button that navigates to `/onboarding?edit=true`.
- [ ] On completion of the re-take flow (hitting the Summary Screen), modify the final button to say "Return to Settings" and navigate back to `/settings` instead of the dashboard.
- [ ] If the backend `GET /api/v1/onboarding/questions` provides the user's previously selected answer IDs as default values, pre-populate the form state when `?edit=true` is present.

---

## Section 8 — Portfolio & Transactions

### 8.1 Deposit Flow
- [ ] **Paper Trading path:** `<DepositDialog>` modal with amount input (`depositSchema`); calls `POST /api/v1/deposit`; on success `toast.success("Funds added")`; invalidate `getUser` query to refresh wallet balance
- [ ] **Stripe path:** `<StripeDepositDialog>` modal; call `createDepositIntent(amount)` to get `clientSecret`; mount `<PaymentElement>` from `@stripe/react-stripe-js`; on Stripe confirmation success show "Deposit submitted — funds arrive after webhook confirmation"
- [ ] Handle Stripe JS loading state (Stripe.js loads asynchronously)

### 8.2 Cashout Form
- [ ] `<CashoutDialog>` modal with amount input (`cashoutSchema`)
- [ ] Display current wallet balance inside modal for reference
- [ ] Calls `POST /api/v1/cashout`; on `400` insufficient balance: inline error "Insufficient balance"; on success `toast.success("Withdrawal processed")`; invalidate `getUser` query

### 8.3 Invest Form
- [ ] `<InvestDialog>` modal with amount input (`investSchema`)
- [ ] Calls `POST /api/v1/invest`; on success `toast.success("Investment added to portfolio")`; invalidate portfolio and user queries
- [ ] On `400` insufficient balance: inline error

### 8.4 Portfolio History Charts (`src/components/charts/`)

Both charts share a common **Time Range Selector** component and are driven by a `GET /api/v1/portfolio/history` endpoint (add to Go backend) that returns time-series data keyed by range. Both must be wrapped in `<ResponsiveContainer width="100%" height={400}>`.

**Shared Time Range Selector (`src/components/charts/TimeRangeSelector.tsx`)**
- [ ] Render a segmented control (styled `<Button>` group) with options: `1D | 1W | 1M | 6M | 1Y | YTD | 5Y`
- [ ] Selected range stored in component state, passed as a query param to the API call
- [ ] Changing range triggers a TanStack Query refetch; show a subtle loading indicator on the chart during transition (do not replace chart with skeleton — use `isFetching` opacity dim instead)
- [ ] Default range on page load: `1M`

---

**Chart A — Value Over Time (`src/components/charts/ValueOverTime.tsx`)**

Shows the absolute portfolio market value in USD over the selected time range, with an optional "Net Contributions" overlay.

- [ ] Recharts `<AreaChart>` with:
  - Primary `<Area>` — `Portfolio Value (USD)`: filled area line representing the total market value of the active `InvestmentRound` at each data point. Use a brand accent gradient fill.
  - Secondary `<Line>` — `Net Contributions`: the cumulative sum of all deposits and investments made up to that point in time (i.e., money actually put in, with no market returns). Rendered as a dashed line in a neutral color.
- [ ] **Net Contributions Toggle:** a shadcn/ui `<Checkbox>` labeled "Show Net Contributions" rendered below the chart. When unchecked, the `<Line>` is hidden (`hide` prop on the Recharts `<Line>` or conditional render). Checked by default.
- [ ] The gap between the two lines is the visual representation of total gain or loss — label this area in the legend as "Unrealized Gain / Loss"
- [ ] Y-axis: formatted as USD (e.g., `$12,450`); X-axis: date labels appropriate to the selected range (e.g., "Jan 12" for 1M, "2022" for 5Y)
- [ ] Custom `<Tooltip>` on hover shows: Date, Portfolio Value, Net Contributions (if toggled on), Gain/Loss (calculated as `portfolioValue − netContributions`), and Gain/Loss as a percentage
- [ ] Skeleton placeholder while loading; `<ChartErrorBoundary>` wrapping the component

---

**Chart B — Performance Chart (`src/components/charts/PerformanceChart.tsx`)**

Shows percentage-based returns relative to the starting value of the selected time range (not all-time).

- [ ] Recharts `<LineChart>` (no fill, clean line) with:
  - Single `<Line>` — `Return (%)`: at each data point, value = `((currentValue − startValue) / startValue) × 100`, where `startValue` is the portfolio value at the beginning of the selected time range
  - This means the line always starts at `0%` on the Y-axis for every range selection — the chart answers "how has this portfolio performed *since* [range start]?"
- [ ] `<ReferenceLine y={0}` rendered as a solid horizontal baseline; values above are styled green, values below are styled red
  - Implement via a custom `<linearGradient>` in SVG defs: stroke above 0 uses a green color variable, stroke below 0 uses a red color variable; alternatively use two overlapping `<Line>` segments clipped at y=0 for a clean two-tone effect
- [ ] Y-axis: formatted as percentage (e.g., `+4.2%`, `-1.8%`)
- [ ] Custom `<Tooltip>` shows: Date, Return % since range start, Absolute value change in USD
- [ ] Skeleton placeholder while loading; `<ChartErrorBoundary>` wrapping the component

---

**Data Requirements for Both Charts**
- [ ] Add `GET /api/v1/portfolio/history?range=1m` endpoint to Go backend returning `{ date, portfolioValue, netContributions }[]` pre-computed per range — keeps heavy aggregation server-side
- [ ] TanStack Query key includes the selected range: `["portfolio-history", range]` — switching ranges hits the cache if previously loaded in the session
- [ ] If user has no active `InvestmentRound`, render an empty state card: "Invest to start tracking your portfolio performance" with a link to the Invest action

### 8.5 Allocation Pie Chart (`src/components/charts/AllocationPie.tsx`)
- [ ] Fetch active portfolio holdings (requires a `GET /api/v1/portfolio` endpoint — add to Go backend if not present; returns active `InvestmentRound` with `Holding[]`)
- [ ] Transform `Holding[]` into `{ name: ticker, value: weight }` array for Recharts `<PieChart>`
- [ ] USD ("cash") holding rendered in a visually distinct neutral color
- [ ] Custom `<Tooltip>` shows: ticker, weight as percentage, allocated dollar amount
- [ ] Legend below chart lists all tickers with their weights
- [ ] Skeleton placeholder (shadcn/ui `<Skeleton>`) while loading

### 8.6 Transaction History Data Table (`src/components/portfolio/TransactionTable.tsx`)

Displays the complete financial event log for the user, covering all four transaction types sourced from two backend models: `Transaction` (invest/sell) and `Funding` (deposit/cashout).

- [ ] Requires a `GET /api/v1/transactions` endpoint on the Go backend that **unifies and returns** all four event types as a single sorted list: `{ id, type, amount, createdAt }[]`
  - `type` values: `"DEPOSIT"` (wallet funded via Stripe or paper trading), `"CASHOUT"` (wallet withdrawal), `"INVEST"` (wallet → portfolio, buying assets), `"SELL"` (portfolio → wallet, selling assets)
  - Sourced from two tables: `Funding` records (`DEPOSIT`, `CASHOUT`) and `Transaction` records (`invest` → mapped to `"INVEST"`, `sell` → mapped to `"SELL"`)
  - Unified, sorted descending by `createdAt`; supports pagination via `?page=` and `?limit=` query params
- [ ] Render using shadcn/ui `<Table>` with columns: **Date**, **Type** (badge), **Amount** (formatted as USD), **Direction** (implicit in badge color)
- [ ] **Type badge color coding:**
  - `DEPOSIT` → blue (money arriving in wallet from outside)
  - `CASHOUT` → orange (money leaving wallet to outside)
  - `INVEST` → green (money moving from wallet into active portfolio)
  - `SELL` → red (portfolio liquidation back to wallet)
- [ ] **Client-side filtering:** `<Select>` or tab group above the table to filter by type: "All", "Deposits & Cashouts", "Investments & Sells"
- [ ] **Client-side sorting** by Date (default: newest first) and Amount
- [ ] **Pagination:** display 10 rows per page; shadcn/ui pagination controls below the table
- [ ] Empty state: "No transactions yet" with a prompt to make a first deposit
- [ ] Skeleton rows (5 placeholder rows) while loading

---

## Section 9 — Forecast Engine

### 9.1 Forecast Request Form (`/forecast`)
- [ ] Fields: Initial Investment (number), Monthly Contribution (number, optional — min 0), Years (1–50 slider) — validated by `forecastSchema`
- [ ] Display user's current risk tolerance and investment horizon as read-only context (so they understand the forecast is personalized)
- [ ] On submit: call `requestForecast(payload)`; store returned `task_id` in component state; transition UI to polling view

### 9.2 Forecast Polling Hook (`src/hooks/useForecast.ts`)
- [ ] Internal state machine: `"idle" | "submitting" | "polling" | "complete" | "error"`
- [ ] After receiving `task_id`, enable TanStack Query `useQuery` for `getForecastStatus(taskId)` with:
  - `refetchInterval: 2000` (2 seconds)
  - `enabled: !!taskId && forecastStatus !== "complete" && forecastStatus !== "error"`
  - TanStack Query stops polling automatically when `enabled` becomes false
- [ ] On `status === "pending"`: show progress indicator with elapsed time counter
- [ ] On `status === "complete"`: parse `payload` JSON, set state to `"complete"`, pass data to chart
- [ ] On `status === "error"`: set state to `"error"`, show `toast.error("Forecast computation failed")`
- [ ] Expose `reset()` action to allow user to request a new forecast

### 9.3 Cone of Uncertainty Chart (`src/components/charts/ConeOfUncertainty.tsx`)
- [ ] Composite Recharts `<AreaChart>` with multiple `<Area>` layers:
  - Outer band: P10–P90 (wide, low-opacity fill) — "full uncertainty range"
  - Inner band: P25–P75 (medium-opacity fill) — "likely range" *(if Python payload includes these)*
  - Median line: P50 `<Line>` — "expected outcome"
- [ ] Data transformation: Python payload percentile arrays → `{ year: N, p10: X, p25: X, p50: X, p75: X, p90: X }[]`
- [ ] X-axis: years (0 → input years); Y-axis: portfolio value formatted as USD
- [ ] `<ReferenceLine>` at initial investment amount labeled "Initial Investment"
- [ ] Custom `<Tooltip>` on hover: displays all percentile values for that year
- [ ] Responsive container wrapping (`<ResponsiveContainer width="100%" height={400}>`)

---

## Section 10 — Polish, Error Handling & Security

### 10.1 Global Toast Notifications (Sonner)
- [ ] `<Toaster />` mounted once in `main.tsx` — never instantiated inside components
- [ ] `toast.error(message)` used in the axios interceptor for all global HTTP errors:
  - `429`: "Too many requests. Please wait before trying again."
  - `423` / account lockout `400`: "Account locked. Please try again in 15 minutes."
  - `5xx`: "A server error occurred. Please try again shortly."
- [ ] `toast.success(message)` used for key user actions: deposit, invest, cashout, profile update, 2FA enable
- [ ] `toast.loading(message)` used for async operations: forecast submission
- [ ] Never show raw API error messages or stack traces in toasts — use pre-written, user-friendly strings

### 10.2 Loading Skeleton States
- [ ] Full-page skeleton during silent token restore (`status === "loading"`)
- [ ] Wallet balance card skeleton (1 line)
- [ ] `<AllocationPie>` skeleton: circular grey placeholder
- [ ] `<ValueOverTime>` skeleton: rectangular chart placeholder
- [ ] `<PerformanceChart>` skeleton: rectangular chart placeholder
- [ ] `<TransactionTable>` skeleton: 5 placeholder grey rows
- [ ] `<ConeOfUncertainty>` skeleton: rectangular chart placeholder
- [ ] All skeletons use shadcn/ui `<Skeleton>` component for visual consistency

### 10.3 React Error Boundaries
- [ ] Create `<ChartErrorBoundary>` wrapping each chart component — catches runtime errors from malformed payload data without crashing the entire page
- [ ] Fallback UI: a shadcn/ui `<Alert variant="destructive">` with "Chart data could not be displayed"

### 10.4 Security Alert UX (Token Reuse Detection)
- [ ] When the axios interceptor receives `"Security Alert: Token reuse detected"` from `/refresh-token`: set `authStore.securityAlert = true`, clear all auth state, redirect to `/login`
- [ ] Login page reads `securityAlert` from Zustand on mount; renders a destructive `<Alert>`: "Your account security has been protected. All sessions were signed out due to suspicious activity."
- [ ] Reset `securityAlert` to `false` after the banner is displayed (one-time display)
- [ ] This is a **distinct, named UX event** — not a generic "session expired" message

### 10.5 XSS Discipline
- [ ] All dynamic data rendered via JSX interpolation `{variable}` — React escapes by default
- [ ] Forecast `payload` data: always `JSON.parse()` and access typed fields — never `dangerouslySetInnerHTML`
- [ ] QR code rendered as `<img src={qr_code_b64} />` — `src` attribute, not injected HTML
- [ ] No `eval()`, no dynamic `<script>` injection anywhere

### 10.6 Content Security Policy
- [ ] Add `<meta http-equiv="Content-Security-Policy">` to `index.html`:
  - `default-src 'self'`
  - `script-src 'self' https://challenges.cloudflare.com` (Turnstile)
  - `frame-src https://challenges.cloudflare.com` (Turnstile iframe)
  - `connect-src 'self' https://api.stripe.com`
  - `style-src 'self' 'unsafe-inline'` (required for Tailwind in dev; tighten in prod)

---

## Section 11 — Build & Deployment

### 11.1 Build Pipeline
- [ ] `npm run build` → outputs to `dist/`
- [ ] Copy `dist/` contents into Go's `./frontend/` directory (or configure `vite.config.ts` `outDir` to point there directly)
- [ ] Verify Go's `r.Static("/static", "./frontend")` and `r.GET("/")` serve correctly after build copy
- [ ] Verify `r.NoRoute(...)` catch-all serves `index.html` for all non-API paths (enables React Router client-side navigation on hard refresh)

### 11.2 VPS Configuration
- [ ] Configure Nginx or Caddy as a reverse proxy in front of Go on Droplet 1 — provides automatic HTTPS/TLS (Let's Encrypt) required for:
  - `Secure` flag on the httpOnly refresh token cookie
  - Stripe.js (requires HTTPS)
  - Cloudflare Turnstile (requires HTTPS in production)
- [ ] Set `GIN_MODE=release` environment variable on Droplet 1
- [ ] Confirm `FRONTEND_BASE_URL` in Go config matches the exact production domain (CORS will silently reject requests on mismatch)
- [ ] Python Droplet (Droplet 2): no public port exposed; accessible only via DigitalOcean VPC private IP

### 11.3 Production Smoke Tests
- [ ] Full auth flow: Register → Verify Email → Login → 2FA (if enabled) → Dashboard loads
- [ ] Onboarding: complete all 7 questionnaire steps, verify derived `riskTolerance` and `investmentHorizon` values are correctly submitted to `PUT /user/profile`
- [ ] Token refresh: manually expire access token (reduce JWT TTL temporarily), confirm silent restore fires and succeeds
- [ ] httpOnly cookie: open DevTools → Application → Cookies; confirm `refresh_token` cookie has `HttpOnly` and `Secure` flags set; confirm it is **not** readable via `document.cookie`
- [ ] Turnstile: confirm widget appears and blocks form submission until solved
- [ ] CORS: confirm no CORS errors in browser console on API calls from production domain
- [ ] Portfolio charts: verify both `ValueOverTime` and `PerformanceChart` render correctly for all 7 time ranges; verify Net Contributions toggle shows/hides the dashed line
- [ ] Transaction table: create one of each type (DEPOSIT, CASHOUT, INVEST, SELL) and verify all four appear with correct badge colors
- [ ] Forecast: submit request, confirm polling resolves and `<ConeOfUncertainty>` renders

---

## Dependency Reference

| Package | Purpose |
|---|---|
| `react-router-dom` | Client-side routing, `<ProtectedRoute>` |
| `@tanstack/react-query` | Server state, caching, forecast polling |
| `zustand` | Auth store (accessToken, user, status) |
| `axios` | HTTP client with request/response interceptors |
| `react-hook-form` | Form state management, submission handling |
| `zod` + `@hookform/resolvers` | Schema validation for all forms |
| `tailwindcss` | Utility-first styling |
| `shadcn/ui` | Pre-built accessible UI components |
| `recharts` | Pie, Area, Line, Bar charts |
| `sonner` | Global toast notification system |
| `@stripe/stripe-js` + `@stripe/react-stripe-js` | Stripe.js Elements for card input |
| `@marsidev/react-turnstile` | Cloudflare Turnstile anti-bot widget |