# Autoclose

The autoclose plugin automatically closes brackets, quotes and the like,
and it can add indentation between such symbols. The plugin can be configured
on a per-buffer basis via the following options:

- `autoclose.pairs`: When the first rune in a pair is entered, autoclose will
  add the second automatically. Moreover, when the first rune is deleted while
  the cursor is on the second, then this second rune is also deleted.
  The default value is ```{"\"\"", "''", "``", "()", "{}", "[]"}```.
- `autoclose.newlinePairs`: When `Enter` is pressed between such a pair,
  autoclose will put the closing rune on a separate line and add indentation.
  The default value is `{"()", "{}", "[]"}`.
