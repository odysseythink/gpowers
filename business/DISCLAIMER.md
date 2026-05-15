# gpowers business/ — responsibility notice

The `business/` module ships strategy and automation skills covering customer
discovery, content, ads, outreach, SEO, ops, and finance. Some of these
(`money-outreach`, `money-ads`, `money-social`, `money-seo`) can be used to
generate cold outreach, paid ads, or scaled content. **You are responsible**
for how you use them. By installing this module you confirm:

1. You will follow applicable laws (CAN-SPAM, GDPR, advertising standards, etc.)
   and platform terms of service when running outreach, ads, or content
   campaigns generated with these skills.
2. You will not use these skills to harass, defraud, or deceive.
3. You understand these skills are advisory: they produce drafts and plans —
   they do not absolve you of human review before publication or send.
4. Your employer or organization may have policies that restrict use of
   automation in customer-facing communication. Check those before relying on
   `business/` output for work-related campaigns.

This module is **opt-in**. It is installed only when `gpowers install` is
invoked with `--with-business`. Uninstall with `gpowers uninstall --module business`.

If you are unsure whether `business/` is appropriate for your context, default
to skipping it — the four-module model (`core/`, `roles/`, `tools/`) is fully
functional without it.
