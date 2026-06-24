---
title: Get a Threads token
description: One-time Meta app and tester setup to obtain a Threads access token for knit.
owner: rnwolfe
lastReviewed: 2026-06-24
---

`knit` wraps a Threads **access token** that grants full read **and publish** access to your
account. Getting one is a one-time Meta-app setup. The fastest path uses Meta's built-in
**User Token Generator** — no browser OAuth round-trip needed.

:::note[What you do and don't need (mid-2026)]
- **No company / legal entity.** A Meta "business portfolio" is free, takes a name, and can be
  deferred. You do **not** need Business Verification to read your *own* data in dev mode.
- **No Instagram account.** Since Sept 2025 the Threads API works for standalone Threads
  profiles (all metrics restored Jan 2026).
- **Verification only gates production** — accessing *other* users' data, going Live, or
  Advanced Access (e.g. public `search posts`). Your own dev token needs none of it.
- **2FA must be enabled** on the Meta account, or the portfolio step shows
  *"no business portfolios available."*
:::

## 1. Create the Threads app (~5 min)

1. Enable **two-factor auth** on your Meta account first.
2. Go to [developers.facebook.com](https://developers.facebook.com) → **My Apps → Create App**.
3. Choose the use case **"Access the Threads API."** (Or pick **"Other"** to skip the wizard's
   portfolio prompt, then add the Threads product from the dashboard.)
4. When prompted for a **business portfolio**: create a free one (just a name) or **add later**,
   and **skip verification** — you don't need it for your own data.
5. In **Use cases → Access the Threads API → Customize**, add the permissions you need:
   `threads_basic` (required), plus `threads_content_publish`, `threads_read_replies`,
   `threads_manage_replies`, `threads_manage_insights` for the full surface. A read-only smoke
   needs only `threads_basic`.
6. In **App settings → Basic**, copy the **Threads App ID** and **Threads App secret** (these are
   Threads-specific — not the top-level Meta app's). You only need these for the browser flow in
   step 3b.

## 2. Add yourself as a tester (the common gotcha)

1. **App roles → Roles → Add People → Threads Tester**, enter your Threads handle.
2. **Approve the invite in the Threads app**: Threads → Settings → Account → *Website permissions*
   → approve. Until you do, every token call returns `400`. In dev mode only tester accounts
   work — which is all you need.

## 3a. Get the token — the easy way (User Token Generator)

In the dashboard's **Threads use-case panel → User Token Generator**, click **Generate Access
Token** for your tester account and complete the security check. You get a short-lived (1-hour)
token. Hand it to `knit`, which validates it and (if `KNIT_CLIENT_SECRET` is set) upgrades it to
a **60-day** token:

```bash
export KNIT_CLIENT_SECRET=<threads-app-secret>   # optional: enables the 60-day upgrade
printf '%s' "<generated-token>" | knit auth login --token-stdin
knit auth status --json                           # confirms account, expiry, source
```

Or skip storage entirely for a quick smoke: `export KNIT_TOKEN=<generated-token>`.

## 3b. Get the token — the browser way (for end-users)

For tokens belonging to people other than yourself (real end-users), run the OAuth flow:

```bash
export KNIT_CLIENT_ID=<threads-app-id>
export KNIT_CLIENT_SECRET=<threads-app-secret>
# KNIT_REDIRECT_URI defaults to https://localhost/knit/callback (register it in the app;
# it must be HTTPS — Meta rejects http://localhost — but never has to actually resolve)
knit auth login
```

`knit` prints an authorize URL. Open it, approve, and the browser lands on
`https://localhost/knit/callback?code=…#_` (shows "can't connect" — fine, the code is in the URL
bar). Paste the full URL back into `knit`. It validates the CSRF `state`, exchanges the code →
short-lived → **60-day** token, and stores it in your OS keyring.

## What works at this stage

- **Reads of your own posts, profile, replies, insights** work immediately as a tester.
- **`search posts` / `mentions list` against the _public_ corpus need App Review** (advanced
  access). Without it they return only *your own* posts — `knit` flags this with `scope:"self"`
  and a stderr note, so you're never misled.

## Troubleshooting

**"You must select a business portfolio" with no way to skip.**
The Threads use-case wizard gates *Continue* on a portfolio. Either create a free one (a name is
enough — no legal entity), or create the app via the **"Other"** use case and add the Threads
product afterward, which lets you defer it.

**"There are no business portfolios available to connect."**
Enable **2FA** on the Meta account — that's the usual cause, not a missing company.

**"You're no longer allowed to use Meta technologies to advertise… can't create business
portfolios."**
This is an account-level **advertising restriction** on your Meta identity (often a false
positive from an old/linked asset). It blocks portfolio creation, which the Threads API requires
for data access, and there's **no documented API-side bypass**. Options: appeal at
[facebook.com/accountquality](https://facebook.com/accountquality) (low success rate), or set the
app up under a **separate, clean Meta account** (with 2FA enabled) — `knit` only needs the
resulting token via `--token-stdin` / `KNIT_TOKEN`, so the app doesn't have to live on your main
account.

Next: [authenticate in different environments](/guides/authenticate/) or the
[quickstart](/start/quickstart/).
