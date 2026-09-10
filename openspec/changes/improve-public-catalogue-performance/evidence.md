## Artist query-index evidence

### Task 5.1 — candidate selection

Evidence was captured with SQLite 3.53.3 from a consistent backup of the local
production-shaped PocketBase database: 5,829 eligible published artists and
52,866 artworks. The exercised list shape selected `Artists.*`, applied the
canonical published/complete-identity predicate, and used `LIMIT 60 OFFSET 0`.
Only the `ORDER BY` changed:

- name ascending: `filing_name ASC, id ASC`
- name descending: `filing_name DESC, id ASC`
- birth: `(year_of_birth = 0) ASC, year_of_birth ASC, filing_name ASC, id ASC`

Before the candidates, all three shapes searched
`pbx_artist_published_name (published=?)` and reported
`USE TEMP B-TREE FOR ORDER BY`.

The selected candidates and resulting plans are:

| Index                                                                                              | Exercised result                                                                                                                                                                                                     |
| -------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `pbx_artist_published_filing` on `(published, filing_name, id)`                                    | Name ascending searches this index with no temporary sort. Name descending uses it but retains `USE TEMP B-TREE FOR LAST TERM OF ORDER BY` because the deterministic `id ASC` direction differs from a reverse scan. |
| `pbx_artist_published_filing_desc` on `(published, filing_name DESC, id ASC)`                      | Name descending searches this index with no temporary sort.                                                                                                                                                          |
| `pbx_artist_published_birth` on `(published, (year_of_birth = 0), year_of_birth, filing_name, id)` | Birth order searches this index with no temporary sort.                                                                                                                                                              |

A plain birth candidate on
`(published, year_of_birth, filing_name, id)` was rejected: the unknown-year
ordering expression still produced `USE TEMP B-TREE FOR ORDER BY`.

Database-size comparison used a compact backup (`VACUUM`, 4,096-byte pages) so
free pages could not hide index allocation:

| State                    |      Pages |           Bytes |                      Delta |
| ------------------------ | ---------: | --------------: | -------------------------: |
| Baseline                 |     45,255 |     185,364,480 |                          — |
| Ascending name added     |     45,319 |     185,626,624 |                   +262,144 |
| Descending name added    |     45,383 |     185,888,768 |                   +262,144 |
| Birth order added        |     45,454 |     186,179,584 |                   +290,816 |
| **All selected indexes** | **45,454** | **186,179,584** | **+815,104 bytes (0.44%)** |

The un-compacted working snapshot had 146 free pages; the first candidate reused
64 of them and therefore showed no physical file growth. The compact comparison
above records the actual allocation rather than that incidental file-size result.

## Artwork list/count experiment

### Task 6.1 — combined window count retained

The candidate selects the ordered page IDs and `COUNT(*) OVER()` in one filtered
query, then hydrates only those sixteen IDs. The previous implementation ran the
complete filter predicates once for `COUNT(DISTINCT id)` and again for the page.
An out-of-range page performs a fallback window read to recover the total and
continues to clamp to the canonical final page.

Deterministic equivalence passed for unfiltered, title/artist text, exact artist
(including a synthetic co-authored match), venue, and date-range cases. The same
five cases also matched total counts and ordered first-page IDs against the
production-shaped 52,866-artwork database. Existing handler tests continued to
pass the empty-result and out-of-range-page contracts.

Benchmark command:

```text
WGA_PERF_DATA_DIR=<isolated-production-shaped-data> go test ./internal/handlers/artworks \
  -run '^$' -bench '^BenchmarkArtworkListCountExperiment$' \
  -benchmem -benchtime=10x -count=3
```

Median results on an AMD Ryzen 7 7800X3D were:

| Case             | Separate ns/op | Combined ns/op |   Time | Bytes/op | Allocs/op |
| ---------------- | -------------: | -------------: | -----: | -------: | --------: |
| Unfiltered       |    206,875,197 |    153,419,884 | -25.8% |    -9.3% |     -4.3% |
| Text (`Madonna`) |    339,097,616 |    185,719,145 | -45.2% |   -15.5% |     -9.0% |
| Exact artist     |    199,656,008 |    127,544,144 | -36.1% |   -12.1% |     -6.9% |
| Venue            |     14,114,029 |     10,837,172 | -23.2% |    -8.4% |     -4.8% |
| Date range       |    190,994,341 |    111,192,226 | -41.8% |    -8.9% |     -5.4% |

A five-case, five-iteration process-level comparison reported 5.11 seconds of
user CPU for the separate path and 3.49 seconds for the combined path (-31.7%);
system CPU fell from 1.73 to 1.15 seconds. Every required case improved elapsed
time, bytes, and allocations, so the combined path was retained.

## Final production-shaped profile matrix

### Task 7.1 — before/after concurrent profiles

The baseline was commit `83e6a35e`, immediately before this change. Both builds
used Go 1.27.0 and independent copies of the same database: 5,829 artists,
52,866 artworks, and 957 locations. Each route received one warm-up request,
then eight concurrent clients for ten seconds with request protection disabled.
`/artworks/results` carried `HX-Request: true` and targeted
`artwork-search-results`; the other routes were ordinary public GET requests.
All measured requests returned HTTP 200 with zero load-generator errors.

Unprofiled load results were:

| Route | Baseline req/s | Final req/s | Throughput | Baseline mean | Final mean | Baseline p95 | Final p95 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `/artworks` | 5.65 | 16.81 | +197.5% | 1,376.08 ms | 472.60 ms | 1,608.83 ms | 552.97 ms |
| `/artworks/results` | 5.44 | 17.04 | +213.2% | 1,439.95 ms | 467.34 ms | 1,652.08 ms | 507.69 ms |
| `/dual-mode` | 18.39 | 97.57 | +430.6% | 434.93 ms | 81.83 ms | 429.82 ms | 86.33 ms |

Ten-second CPU profiles ran under the same eight-client load. Total samples can
exceed wall time because requests execute across cores; dividing by completed
requests gives the comparable CPU cost:

| Route | Baseline samples / requests | Final samples / requests | CPU per request | Change |
| --- | ---: | ---: | ---: | ---: |
| `/artworks` | 39.59 s / 56 | 43.15 s / 176 | 706.96 → 245.17 ms | -65.3% |
| `/artworks/results` | 39.26 s / 57 | 40.34 s / 168 | 688.77 → 240.12 ms | -65.1% |
| `/dual-mode` | 50.09 s / 184 | 39.38 s / 977 | 272.23 → 40.31 ms | -85.2% |

Exact `runtime.MemStats` deltas around separate ten-second runs supplied
allocation counts rather than relying on sampled-profile totals:

| Route | Baseline bytes/request | Final bytes/request | Baseline mallocs/request | Final mallocs/request |
| --- | ---: | ---: | ---: | ---: |
| `/artworks` | 1,451,462 | 1,415,550 (-2.5%) | 10,567 | 9,790 (-7.4%) |
| `/artworks/results` | 777,719 | 725,909 (-6.7%) | 7,326 | 6,545 (-10.7%) |
| `/dual-mode` | 2,140,278 | 2,100,161 (-1.9%) | 19,564 | 19,109 (-2.3%) |

An initial sampled allocation profile exposed avoidable allocation in the new
in-memory collection ordering: repeatedly constructing folded labels and growing
the option slice accounted for about 122 KiB per full-page request. The final
implementation preallocates from the bounded 957-location projection and uses an
allocation-free ASCII `NOCASE` comparator. Focused race tests preserved the
ordering and forty-option contracts, and the exact counters above show that the
apparent full-page regression was removed.

Forced-GC post-load `HeapAlloc` was 3.94 → 4.00 MiB for `/artworks`, 3.64 →
3.96 MiB for `/artworks/results`, and 3.87 → 4.59 MiB for `/dual-mode`.
The largest increase was 0.72 MiB and includes the deliberately bounded shared
artist-availability set. Post-warm heap profile differences contained runtime,
regular-expression, buffer, and SQLite sampling buckets, but no growing
route-owned response cache or request-keyed catalogue projection.

Dominant call-path comparison confirmed that the removed work no longer occurs
per request:

- Baseline `/artworks` spent 45.72% cumulative CPU in the catalogue-wide
  `getVenueOptions` SQL query; the warmed final profile spent 0.18% in its
  in-memory ordering, with no collection-holdings loader query sampled.
- Baseline `/artworks/results` ran the full search-view builder for 96.84% of
  sampled CPU, including facets. The final route ran only
  `buildArtworkSearchResultsViewContext`; its remaining dominant work was the
  combined ordered page/window-count query.
- Baseline `/dual-mode` spent 70.79% cumulative CPU in the per-request
  `publishedArtworkAuthorIDs` catalogue scan. The final profile contained no
  sample for `loadPublishedArtworkAuthorIDs`; remaining application work was
  headed by the bounded artist count (40.73%) and list (2.89%) queries.

SQLite remains the dominant execution engine—96.08% cumulative CPU for the full
artwork route, 95.98% for results-only, and 85.98% for Dual Mode—but it now serves
substantially more requests with lower CPU and allocation cost per request. No
material throughput, latency, CPU, allocation, or retained-heap regression
remained after investigation.
