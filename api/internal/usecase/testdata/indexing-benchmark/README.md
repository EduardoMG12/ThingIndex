# Indexing Benchmark Corpus

This controlled corpus backs the public ThingIndex indexing benchmark exposed by `GET /api/v1/meta/indexing-benchmark`.

It exists to keep the baseline explicit and reproducible:

- each document fixture is embedded in the Go binary;
- fixtures may live inline in `corpus.json` or as embedded files referenced by `content_file`;
- the benchmark runs the real deterministic indexer over those fixtures;
- document-tree cases validate exact top-1 node/page matches;
- collection cases validate exact top-1 document matches;
- structure checks validate expected outline titles and table counts.

The benchmark is intentionally small and deterministic. It is a runtime baseline, not a claim of general production accuracy for arbitrary corpora or grounded chat answers.
