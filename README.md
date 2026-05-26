# TagPrint

Ad hoc nametag printer for the Brother QL-820NWB. It uses the same direct
network-printer path as `freshservice-label`, with labels supplied at runtime.

Print one or more literal labels:

```sh
go run ./... -label "John Doe" -label "Joe Doe"
```

Print an incrementing series:

```sh
go run ./... -series "Spare 1..6"
```

Series keep the padding you type:

```sh
go run ./... -series "CRT 03..08"
```

You can combine literal labels and series:

```sh
go run ./... -label "John Doe" -series "Spare 1..6"
```

Print only the first requested label:

```sh
go run ./... -series "CRT 03..08" -limit 1
```

Render a preview PNG without printing:

```sh
go run ./... -label "CRT 03" -preview
```

`PRINTER_ADDR` defaults to `172.19.10.13`, matching the deployed
`freshservice-label` printer. Override it if needed:

```sh
PRINTER_ADDR=172.19.10.13 go run ./...
```
