---
title: Get a Threads token
description: One-time Meta app and tester setup to obtain a Threads access token for knit.
owner: rnwolfe
lastReviewed: 2026-06-23
---

`knit` wraps a Threads **access token** that grants full read **and publish** access to your
account. Getting one is a one-time app setup plus an OAuth approval. `knit auth login` automates
the token exchange once the app exists.

## 1. Create the Threads app (~5 min)

1. Go to [developers.facebook.com](https://developers.facebook.com) → **My Apps → Create App**.
2. Choose the use case **"Access the Threads API"** (the Threads product is separate from the
   regular Graph API).
3. In **Use cases → Access the Threads API → Customize**, add the permissions you need:
   `threads_basic` (required), and for the full surface `threads_content_publish`,
   `threads_read_replies`, `threads_manage_replies`, `threads_manage_insights`. A read-only
   smoke needs only `threads_basic`.
4. In **App settings → Basic**, copy the **Threads App ID** and **Threads App secret** (these are
   Threads-specific — not the top-level Meta app's).
5. Under the use-case **Settings**, add an OAuth redirect callback URI — exactly:
   ```
   https://localhost/knit/callback
   ```
   It never has to resolve; `knit` only needs the `?code=` it puts in your address bar. It must
   be **HTTPS** — Meta rejects `http://localhost`.

## 2. Add yourself as a tester (the common gotcha)

1. **App roles → Roles → Add People → Threads Tester**, enter your Threads handle.
2. **Accept the invite in the Threads app**: Threads → Settings → Account → *Website permissions*
   (or *Invites*) → accept. Until you accept, every token call returns `400`. While the app is in
   dev mode, only tester accounts work — which is all you need.

## 3. Mint the token with knit

```bash
export KNIT_CLIENT_ID=<threads-app-id>
export KNIT_CLIENT_SECRET=<threads-app-secret>
# KNIT_REDIRECT_URI defaults to https://localhost/knit/callback
knit auth login
```

`knit` prints an authorize URL. Open it in a browser **logged into your tester Threads account**,
approve, and your browser lands on `https://localhost/knit/callback?code=…#_` (it shows "can't
connect" — fine, the code is in the URL bar). **Copy the full URL, paste it back into `knit`,
press Enter.**

`knit` validates the CSRF `state`, exchanges the code → short-lived token → **60-day long-lived
token**, and stores it in your OS keyring.

```bash
knit auth status --json   # confirms account, expiry, source
```

:::tip[Headless / CI]
For an agent or CI you don't need the browser at all — provide a token you already minted:
`printf '%s' "$THREADS_TOKEN" | knit auth login --token-stdin`, or just export `KNIT_TOKEN`.
:::

## What works at this stage

- **Reads of your own posts, profile, replies, insights** work immediately as a tester.
- **`search posts` / `mentions list` against the _public_ corpus need App Review** (advanced
  access). Without it they return only *your own* posts — `knit` flags this with `scope:"self"`
  and a stderr note, so you're never misled.

Next: [authenticate in different environments](/guides/authenticate/) or the
[quickstart](/start/quickstart/).
