module github.com/oerlikon/sounding

go 1.19

require (
	github.com/fatih/structs v1.1.0
	github.com/gorilla/websocket v1.5.0
	github.com/mattn/go-shellwords v1.0.12
	github.com/rs/zerolog v1.28.0
	github.com/spf13/pflag v1.0.5
	github.com/valyala/fastjson v1.6.3
	golang.org/x/term v0.2.0
	gopkg.in/validator.v2 v2.0.1
)

require (
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.16 // indirect
	golang.org/x/sys v0.2.0 // indirect
)

replace (
	github.com/fatih/structs => github.com/oerlikon/structs v1.2.0
	github.com/valyala/fastjson => github.com/oerlikon/fastjson v1.6.3-patch3
)
