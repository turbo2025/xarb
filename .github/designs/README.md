# Designs

This directory stores design documents created during the Brainstorming phase of Superpowers-driven development.

## Structure

Each feature design should follow this format:

```
designs/
├── feature-name.md
└── feature-name/
    ├── diagrams/
    ├── specifications/
    └── decisions.md
```

## Template

```markdown
# Feature: [Name]

## Problem Statement
[What problem does this solve?]

## Design Overview
[High-level architecture]

## Key Decisions
- Decision 1: [Why?]
- Decision 2: [Why?]

## Implementation Tasks
[Linked to implementation plan]

## Acceptance Criteria
- [ ] Criterion 1
- [ ] Criterion 2

## Updated
- Date: YYYY-MM-DD
- Version: 1.0
```

## Examples

- `prometheus-metrics.md` - Prometheus monitoring integration design
- `websocket-optimization.md` - WebSocket performance improvements
- `multi-strategy.md` - Multiple trading strategies support
