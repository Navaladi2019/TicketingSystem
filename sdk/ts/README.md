# @nava/sdk

`describeTotal(page, listed?, best?)` returns `{ text, state, value }` for a list's total: `"1,234"`, `"1,000+"`, `"50+"` or `"many"`, never `"0"` for an `unknown` count.
Pass the `TotalView` of an earlier page as `best` so a known total is not replaced by a later `unknown` page.
