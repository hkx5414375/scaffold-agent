# Token benchmarks

The reproducible full-suite benchmark measures the Blueprint plus every
generated UTF-8 source byte against the Blueprint plus bounded Engine workflow
results. Both sides use `ceil(UTF-8 bytes / 4)`, a stable comparative estimate
rather than any provider's billing tokenizer.

| Backend | Files | Full source estimate | Engine workflow | Saved | Reduction |
| --- | ---: | ---: | ---: | ---: | ---: |
| Go | 185 | 380,954 | 1,913 | 379,041 | 99.50% |
| Java | 251 | 436,270 | 2,104 | 434,166 | 99.52% |
| Python | 234 | 483,496 | 1,982 | 481,514 | 99.59% |

The fixed scenario includes PostgreSQL, Vue administration, Nuxt storefront,
one five-field business entity, and every platform, commerce, CRM, and ERP
capability. CI requires at least 90% estimated reduction and at most 8,000
estimated workflow tokens per backend.

Reproduce it without credentials, network calls, package managers, compilers,
or databases:

```bash
go run ./cmd/scaffold-agent benchmark --json
```

See the [Chinese methodology and limitations](token-benchmarks.zh-CN.md) for the
precise comparison boundary.
