# Prompt Regression CLI

This command runs the same versioned dataset against `baseline` and `candidate`, scores each output with deterministic rules, and writes JSON plus Markdown reports.

## Run

From the repository root, with `DEEPSEEK_API_KEY` configured:

```powershell
go run ./cmd/promptregression -eval-config prompts/regression/config.yaml
```

Run only selected task types:

```powershell
go run ./cmd/promptregression -eval-config prompts/regression/config.yaml -task-types candidate_extraction,failure_response
```

Useful overrides are `-dataset`, `-task-types`, `-out-json`, and `-out-md`. Model, parameters, prompt paths, prompt versions, report paths, and optional LLM Judge settings live in `prompts/regression/config.yaml`.

## Dataset v1

The dataset envelope contains:

- `schema_version`: currently `evaluation-dataset/v1`
- `dataset_version`: immutable version of the case collection
- `name`: human-readable dataset name
- `cases`: fixed evaluation cases

Every case contains `id`, `task_type`, `category`, `input`, `context`, `expected`, and `assertions`. Context can carry prior messages, prompt variables, source chunks, and confirmed user facts.

Deterministic assertions support required alternatives, required text, forbidden text, question detection, valid JSON, and maximum length. Failed rules are authoritative and make the case fail. `manual_review` marks subjective cases for review. LLM Judge is disabled by default and remains auxiliary; it never overrides deterministic status.

## Reports

Each run records model, prompt versions by task, parameters, dataset version, duration, provider-reported Token usage, raw output, deterministic failures, manual-review status, auxiliary Judge output, and invocation errors. The comparison section aligns baseline and candidate by case ID and identifies exact regressions and improvements.