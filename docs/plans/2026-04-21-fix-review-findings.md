# Fix TDD/DDD/CA Review Findings — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Reach 9+/10 on TDD, DDD, and CA axes per code review findings.

**Architecture:** Fix DDD violations (sentinel errors, zero-value literals, anemic setters), CA violations (hardcoded providers, ignored constructor error), raise coverage to 100%.

**Tech Stack:** Go 1.26, React/Vite/TypeScript

---

### Task 1: DDD — Move ErrInvalidScore to errors.go
Remove from match_score.go, add to errors.go for consistency.

### Task 2: DDD — TokenUsage zero-value via domain function  
Replace all `domain.TokenUsage{}` outside domain with `domain.ZeroTokenUsage()`.

### Task 3: DDD — Fix ignored constructor error in analyze_tender.go:96
Handle error from `domain.NewTenderAnalysis`.

### Task 4: CA — Extract availableProviders from handler
Move to config or inject via DI.

### Task 5: Coverage — domain getters (Sections, Meta, Log, Summary, Pros, Cons, Requirements, Effort, SetSections, SetMeta, SetLog, SetExtras)

### Task 6: Coverage — HandleListModels, HandleGenerateProposal, HandleAnalyzeTender gaps

### Task 7: Coverage — LLM ListModels (anthropic, gemini, openai), prompts

### Task 8: Coverage — usecase Execute gaps (analyze_tender, generate_proposal)
