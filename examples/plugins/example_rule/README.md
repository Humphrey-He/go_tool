# Example plugin

Build:

```
go build -buildmode=plugin -o example_rule.so ./examples/plugins/example_rule
```

Use:

```
sqlsafelint scan --config .sqlsafelint.json --baseline baseline.json --out-json report.json
```

Set config:

```
{
  "rules": {
    "plugins": ["/absolute/path/to/example_rule.so"]
  }
}
```
