# CSV format

Naraberu imports and exports series as UTF-8 CSV. Column order in a typical file:

```text
englishName,japaneseName,synonyms,status,linkedIds,thumbnailPath,startDate,endDate,score,review,synopsis,tags,favorite,owned
```

| Column | Notes |
|--------|--------|
| `englishName` | Required. Used to skip duplicates on import. |
| `japaneseName` | Optional. |
| `synonyms` | `;`-separated. |
| `status` | `to-watch`, `watched`, `in-progress`, `abandoned`, or empty. |
| `linkedIds` | `;`-separated series ids (UUID). Invalid entries are cleared with a warning. |
| `thumbnailPath` | Local path or an `http(s)` image URL (fetched on import). |
| `startDate` / `endDate` | `YYYY`, `YYYY/MM`, or `YYYY/MM/DD` (hyphens also accepted). |
| `score` | 0–10. Out of range values are set to 0 with a warning. |
| `review` | Free text; may be a quoted multi-line field. |
| `synopsis` | Free text; may be a quoted multi-line field. |
| `tags` | `;`-separated. |
| `favorite` | `true` or `false`. |
| `owned` | `true` or `false`. |

The first twelve columns are required for a data row; `favorite` and `owned` may be omitted. When those two names appear in the header, values are read by column name.

Export uses the same header. Import skips rows that already exist by English name and reports them in the result summary.
