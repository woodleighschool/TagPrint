# TagPrint

Ad hoc nametag printer for the Brother QL-820NWB. It uses the same direct
network-printer path as `freshservice-label`, but keeps the labels in code so
you can edit the list, run it, and get stock out of the printer.

Edit `labels` in `main.go`:

```go
var labels = []labelSet{
	names("John Doe", "Joe Doe"),
	numbered("Spare", 1, 6),
}
```

Print all configured labels:

```sh
go run ./...
```

Print only the first configured label:

```sh
go run ./... -limit 1
```

Render a preview PNG without printing:

```sh
go run ./... -preview
```

`PRINTER_ADDR` defaults to `172.19.10.13`, matching the deployed
`freshservice-label` printer. Override it if needed:

```sh
PRINTER_ADDR=172.19.10.13 go run ./...
```
