# Vision

<!-- Operator-owned. Drafted from existing docs (README, WORKFLOW) — review and edit; agents read this as authoritative product intent and never write to it. -->

## Purpose

g0router is a single-binary, self-hosted LLM gateway. It unifies many AI provider APIs behind one OpenAI-compatible endpoint so any client speaks one format while g0router handles provider translation, OAuth credential flows, account fallback, token compression, and MCP tool routing.

## Target Users

Individual developers and small teams who run their own gateway: they hold multiple provider accounts (API keys and OAuth subscriptions), want automatic failover between them, need usage/cost visibility, and prefer one deployable binary with an embedded web dashboard over a hosted SaaS proxy.

## Success Looks Like

A user points any OpenAI-format client at g0router and requests route reliably across 43+ providers — rate limits trigger silent fallback, usage and cost show up in the dashboard, credentials stay encrypted at rest, and operating the gateway (auth, tunnels, governance, updates) requires nothing beyond the binary and its web UI.
