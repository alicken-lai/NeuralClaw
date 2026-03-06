# Retrieval Evaluation

This document describes the `neuralclaw eval retrieval` command and how retrieval quality is measured.

## Command

```sh
./neuralclaw eval retrieval --golden ./eval/golden.yaml --k 10 --output ./eval/report.json
```

Flags:
- `--golden <path>` (required): YAML file with golden queries
- `--k <int>` (optional, default `10`): evaluate top-K hits
- `--output <path>` (optional): write aggregated JSON report

## Golden Query YAML Format

Each query entry:

```yaml
- id: q_login_flow
  text: "how does login token refresh work"
  expected_ids:
    - mem-123
    - mem-456
  scope: "project:auth"
```

Fields:
- `id`: query identifier
- `text`: retrieval query text
- `expected_ids`: relevant memory IDs expected to be returned
- `scope`: retrieval scope used during search

## Metrics

For each query:

- `Recall@K`
  - `hits_in_top_k / total_expected`
- `MRR@K` (Mean Reciprocal Rank at K)
  - reciprocal rank of first relevant hit in top-K, else `0`
- `NDCG@K`
  - `DCG@K / IDCG@K` with binary relevance

Aggregated suite metrics are arithmetic means across all query results.

## Output

The command prints:
- a tabular per-query report
- an `AVG` row for `Recall@K`, `MRR@K`, `NDCG@K`

When `--output` is set, it also writes a JSON payload containing:
- per-query `EvalResult[]`
- suite-level `AvgRecall`, `AvgMRR`, `AvgNDCG`, `Total`
