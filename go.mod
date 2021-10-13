module sounding

go 1.17

require (
	github.com/gorilla/websocket v1.4.2
	github.com/mattn/go-shellwords v1.0.12
	github.com/schollz/progressbar/v3 v3.8.3
	github.com/spf13/pflag v1.0.5
	github.com/valyala/fastjson v1.6.3
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211
	gopkg.in/validator.v2 v2.0.0-20210331031555-b37d688a7fb0
)

require (
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519 // indirect
	golang.org/x/sys v0.0.0-20211013075003-97ac67df715c // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

replace (
	github.com/schollz/progressbar/v3 => github.com/oerlikon/progressbar/v3 v3.8.2-patch4
	github.com/valyala/fastjson => github.com/oerlikon/fastjson v1.6.3-patch3
	gopkg.in/yaml.v2 => github.com/oerlikon/yaml/v2 v2.4.0-patch1
)
