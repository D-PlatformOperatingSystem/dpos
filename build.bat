go env -w CGO_ENABLED=0
go build -o build/dplatformos.exe github.com/D-PlatformOperatingSystem/dpos/cmd/dplatformos
go build -o build/dplatformos-cli.exe github.com/D-PlatformOperatingSystem/dpos/cmd/cli
