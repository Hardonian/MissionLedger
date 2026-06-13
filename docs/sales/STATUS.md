# Release Status

v0.1.0: READY
- README, docs, sales, checkout, packaging files present
- Enrollment: open

Blockers:
- Git push not completed because HTTPS auth is still asking for username on this machine
- Finishing the Stripe setup is still blocked by missing live price IDs/local execution artifacts

Verification commands:
- git status
- rg -n "buy.stripe.com" docs/checkout.html
- rg -n "price_" docs/pricing.html
